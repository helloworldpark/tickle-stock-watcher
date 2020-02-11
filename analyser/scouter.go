package analyser

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/storage"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

const days = 10

func findProspects(dbClient *database.DBClient, itemChecker *watcher.StockItemChecker) map[string]string {
	stocks := itemChecker.AllStockID()
	logger.Info("[Analyser][Prospects] Finding from %d stocks", len(stocks))

	result := make(map[string]string)
	for _, stockID := range stocks {
		prospects := NewProspect(dbClient, days, stockID)
		if len(prospects) > 0 {
			didPlot, savePath := NewCandlePlot(dbClient, days, stockID, itemChecker)
			if didPlot {
				savePath, err := uploadLocalImage(savePath)
				if err == nil {
					url := "https://storage.googleapis.com/ticklemeta-storage/" + savePath
					result[stockID] = url

					logger.Error("[Scouter] %s", url)
				} else {
					result[stockID] = ""
				}
			} else {
				result[stockID] = ""
			}
		}
	}
	return result
}

func runOnFind(stockID, picURL string, itemChecker *watcher.StockItemChecker, now time.Time, onFind func(msg, picURL string)) {
	var buf bytes.Buffer
	addLine := func(str string, args ...interface{}) {
		if len(args) > 0 {
			str = fmt.Sprintf(str, args...)
		}
		buf.WriteString(str)
		buf.WriteString("\n")
	}

	stockInfo, _ := itemChecker.StockFromID(stockID)
	addLine("[Prospect] %v", now)
	addLine("[Prospect] #%s: %s", stockID, stockInfo.Name)
	if len(picURL) > 0 {
		onFind(buf.String(), picURL)
	} else {
		onFind(buf.String(), "")
	}
}

func cleanupLocal(onFind func(msg, url string)) {
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
}

// FindProspects Find prospects using this function. This function uses cache.
func FindProspects(dbClient *database.DBClient, itemChecker *watcher.StockItemChecker, onFind func(msg, savePath string)) {
	now := commons.Now()
	hour := now.Hour()

	var y, d int
	var m time.Month
	if hour < 20 {
		// 당일 20:00 전
		y, m, d = now.AddDate(0, 0, -1).Date()
	} else {
		// 당일 20:00 이후
		y, m, d = now.Date()
	}
	storagePath := fmt.Sprintf(saveDirFormat, y, m, d)
	filesAttrs := storage.FilesInDirectory(storagePath)

	if len(filesAttrs) == 0 {
		// 캐시가 없다
		// 새로 만들어서 내려보낸다
		cleanupLocal(onFind)
		prospects := findProspects(dbClient, itemChecker)
		for stockID, url := range prospects {
			runOnFind(stockID, url, itemChecker, now, onFind)
		}
		onFind(fmt.Sprintf("%d prospects recommended", len(prospects)), "")

		// 다 만들었으니 로컬 파일은 삭제
		cleanupLocal(onFind)
		return
	}

	// 캐시가 있다
	latest := filesAttrs[0].Updated
	for _, attr := range filesAttrs {
		if attr.Updated.After(latest) {
			latest = attr.Updated
		}
	}
	// 유효한 캐시: referenceTime에서 24시간 이내, 20:00 기준
	referenceTime := time.Date(y, m, d, 20, 0, 0, 0, commons.AsiaSeoul)
	isValidCache := latest.After(referenceTime) && latest.Before(referenceTime.AddDate(0, 0, 1))
	if isValidCache {
		// 유효한 캐시라면 그 캐시값을 내려보낸다
		result := make(map[string]string)
		for _, attrs := range filesAttrs {
			paths := strings.Split(attrs.Name, "/")
			if len(paths) >= 3 {
				stockID := paths[len(paths)-1]
				stockID = strings.Trim(stockID, "candle")
				savePath := strings.Join(paths[len(paths)-3:], "/")
				url := "https://storage.googleapis.com/ticklemeta-storage/" + savePath
				result[stockID] = url
			}
		}

		for stockID, url := range result {
			runOnFind(stockID, url, itemChecker, now, onFind)
		}
		onFind(fmt.Sprintf("%d prospects recommended", len(result)), "")
		return
	}

	// 무효한 캐시라면 새로 만들어서 내려보낸다
	cleanupLocal(onFind)
	prospects := findProspects(dbClient, itemChecker)
	for stockID, url := range prospects {
		runOnFind(stockID, url, itemChecker, now, onFind)
	}
	onFind(fmt.Sprintf("%d prospects recommended", len(prospects)), "")

	// 다 만들었으니 로컬 파일은 삭제
	cleanupLocal(onFind)
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
