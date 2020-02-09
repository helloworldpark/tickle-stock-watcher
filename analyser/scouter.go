package analyser

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/storage"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

func FindProspects(dbClient *database.DBClient, itemChecker *watcher.StockItemChecker, onFind func(msg, savePath string)) {
	// Delete all past records(remote)
	if err := storage.Clean(magicString); err != nil {
		logger.Error("[Analyser][Prospects] Error while cleaning google storage: %+v", err)
	}

	// Delete all past records(local)
	if err := CleanupOldCandleplots(); err != nil {
		logger.Error("[Analyser][Prospects] Error while cleaning local: %+v", err)
		onFind("No prospects today!", "")
		return
	}
	if err := MkCandlePlotDir(); err != nil {
		logger.Error("[Analyser][Prospects] Error while making directory for candleplot: %+v", err)
		onFind("No prospects today!", "")
		return
	}

	stocks := itemChecker.AllStockID()
	logger.Info("[Analyser][Prospects] Finding from %d stocks", len(stocks))
	var count = 0
	const days = 10
	for _, stockID := range stocks {
		prospects := NewProspect(dbClient, days, stockID)
		if len(prospects) > 0 {
			var buf bytes.Buffer
			addLine := func(str string, args ...interface{}) {
				if len(args) > 0 {
					str = fmt.Sprintf(str, args...)
				}
				buf.WriteString(str)
				buf.WriteString("\n")
			}

			stockInfo, _ := itemChecker.StockFromID(stockID)
			addLine("[Prospect] #%s: %s", stockID, stockInfo.Name)
			for _, prospect := range prospects {
				y, m, d := commons.Unix(prospect.Timestamp).Date()
				addLine("    %4d년 %02d월 %02d일", y, m, d)
			}

			stockItemChecker := watcher.NewStockItemChecker(dbClient)
			didPlot, savePath := NewCandlePlot(dbClient, days, stockID, stockItemChecker)
			if didPlot {
				savePath, err := uploadLocalImage(savePath)
				if err == nil {
					url := "https://storage.googleapis.com/ticklemeta-storage/" + savePath
					logger.Error("[Scouter] %s", url)
					onFind(buf.String(), url)
				} else {
					onFind(buf.String(), "")
				}
				onFind(buf.String(), savePath)
			} else {
				onFind(buf.String(), "")
			}

			count++
		}
	}
	if count == 0 {
		onFind("No prospects today", "")
	}
}

func uploadLocalImage(localPath string) (string, error) {
	// Upload new images(today)
	png := storage.PNGtoBytes(localPath)
	contentType := http.DetectContentType(png)
	if !strings.HasSuffix(contentType, "png") {
		return "", newError("PNG file is not PNG")
	}
	splits := strings.Split(localPath, "/")
	splits = splits[len(splits)-2:]
	savePath := strings.Join(splits, "/")
	savePath, err := storage.Write(png, savePath)
	return savePath, err
}
