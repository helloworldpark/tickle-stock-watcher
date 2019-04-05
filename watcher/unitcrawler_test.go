package watcher_test

import (
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

func TestCrawl(t *testing.T) {
	stock := commons.Stock{Name: "Samsung Electronics", MarketType: commons.KOSPI, StockID: "005930"}
	watcher.CrawlPast(stock.StockID, 2)
}

func TestCrawlNow(t *testing.T) {
	stock := commons.Stock{Name: "Samsung Electronics", MarketType: commons.KOSPI, StockID: "005930"}
	watcher.CrawlNow(stock.StockID, 0)
}
