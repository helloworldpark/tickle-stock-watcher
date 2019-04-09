package watcher_test

import (
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

func TestCrawl(t *testing.T) {
	stock := structs.Stock{Name: "Samsung Electronics", MarketType: structs.KOSPI, StockID: "005930"}
	watcher.CrawlPast(stock.StockID, 2)
}

func TestCrawlNow(t *testing.T) {
	stock := structs.Stock{Name: "Samsung Electronics", MarketType: structs.KOSPI, StockID: "005930"}
	watcher.CrawlNow(stock.StockID, 0)
}
