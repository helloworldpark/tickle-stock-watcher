package analyser

import (
	"bytes"
	"fmt"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

func FindProspects(dbClient *database.DBClient, itemChecker *watcher.StockItemChecker, onFind func(msg string)) {
	stocks := itemChecker.AllStockID()
	logger.Info("[Analyser] Finding Prospect from %d stocks", len(stocks))
	var count = 0
	for _, stockID := range stocks {
		prospects := NewProspect(dbClient, 10, stockID)
		fmt.Println("Stock ID: ", stockID)
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

			onFind(buf.String())
			count++
		}
	}
	if count == 0 {
		onFind("No prospects today")
	}
}
