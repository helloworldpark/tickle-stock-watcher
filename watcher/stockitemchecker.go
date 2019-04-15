package watcher

import (
	"github.com/anaskhan96/soup"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

type StockItemChecker struct {
	stocks map[string]structs.Stock
}

// NewStockItemChecker returns a new StockItemChecker with stocks unfilled.
// User must update the stocks.
func NewStockItemChecker() StockItemChecker {
	checker := StockItemChecker{
		stocks: make(map[string]structs.Stock),
	}
	return checker
}

func (checker *StockItemChecker) IsValid(stockid string) bool {
	_, ok := checker.stocks[stockid]
	return ok
}

func (checker *StockItemChecker) StockFromID(stockid string) structs.Stock {
	stock, _ := checker.stocks[stockid]
	return stock
}

func (checker *StockItemChecker) UpdateStocks() {
	kospi := downloadStockSymbols(structs.KOSPI)
	for _, v := range kospi {
		checker.stocks[v.StockID] = v
	}
	kospi = nil
	kosdaq := downloadStockSymbols(structs.KOSDAQ)
	for _, v := range kosdaq {
		checker.stocks[v.StockID] = v
	}
	kosdaq = nil
}

// https://minjejeon.github.io/learningstock/2017/09/07/download-krx-ticker-symbols-at-once.html
func downloadStockSymbols(market structs.Market) []structs.Stock {
	marketType := map[structs.Market]string{
		"kospi":  "stockMkt",
		"kosdaq": "kosdaqMkt",
		// "konex":  "konexMkt",
	}

	u := "http://kind.krx.co.kr/corpgeneral/corpList.do?method=download&searchType=13"
	if market != "" {
		u += "&marketType="
		u += marketType[market]
	}

	response, err := soup.Get(u)
	if err != nil {
		logger.Error("[Watcher] %s", err.Error())
		return nil
	}

	symbolHTML := soup.HTMLParse(response)
	handleSoupError(symbolHTML)

	table := symbolHTML.Find("table")
	handleSoupError(table)

	trs := table.FindAll("tr")
	trs = trs[1:]

	result := make([]structs.Stock, len(trs))
	for i, v := range trs {
		tds := v.FindAll("td")
		name := tds[0].Text()
		id := tds[1].Text()
		result[i] = structs.Stock{StockID: id, Name: name, MarketType: market}
	}

	return result

}
