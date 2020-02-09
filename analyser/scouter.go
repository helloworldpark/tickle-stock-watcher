package analyser

import (
	"bytes"
	"fmt"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

func FindProspects(dbClient *database.DBClient, itemChecker *watcher.StockItemChecker, onFind func(msg, savePath string)) {
	// Delete all past records
	if err := CleanupOldCandleplots(); err != nil {
		logger.Error("[Analyser][Prospects] Error while cleanup: %+v", err)
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
