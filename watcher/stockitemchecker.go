package watcher

import (
	"github.com/anaskhan96/soup"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

// StockItemChecker is a simple struct holding stock info and a DB client.
type StockItemChecker struct {
	stocks   map[string]structs.Stock
	dbClient *database.DBClient
}

// NewStockItemChecker returns a new StockItemChecker with stocks unfilled.
// User must update the stocks.
func NewStockItemChecker(dbClient *database.DBClient) StockItemChecker {
	checker := StockItemChecker{
		stocks:   make(map[string]structs.Stock),
		dbClient: dbClient,
	}
	return checker
}

// IsValid checks if the given stock ID exists in the list.
func (checker *StockItemChecker) IsValid(stockid string) bool {
	_, ok := checker.stocks[stockid]
	return ok
}

// StockFromID finds structs.Stock from stock id
func (checker *StockItemChecker) StockFromID(stockid string) structs.Stock {
	stock, _ := checker.stocks[stockid]
	return stock
}

// UpdateStocks updates stock info from the KRX server.
func (checker *StockItemChecker) UpdateStocks() {
	stocksDB := make([]interface{}, 0)
	kospi := downloadStockSymbols(structs.KOSPI)
	for _, v := range kospi {
		checker.stocks[v.StockID] = v
		stocksDB = append(stocksDB, v)
	}
	kospi = nil
	kosdaq := downloadStockSymbols(structs.KOSDAQ)
	for _, v := range kosdaq {
		checker.stocks[v.StockID] = v
		stocksDB = append(stocksDB, v)
	}
	kosdaq = nil
	checker.dbClient.Upsert(stocksDB...)
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
