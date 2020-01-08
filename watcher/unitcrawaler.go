package watcher

import (
	"fmt"
	"net/http"
	"time"

	"github.com/anaskhan96/soup"
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

const (
	dateFormat    = "2006.01.02"
	pastURLFormat = "https://finance.naver.com/item/sise_day.nhn?code=%s&page=%d"
	nowURLFormat  = "https://finance.naver.com/item/main.nhn?code=%s"
)

// CrawlPast actually performs crawling for the past prices
func CrawlPast(stockID string, page int) []structs.StockPrice {
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := soup.GetWithClient(fmt.Sprintf(pastURLFormat, stockID, page), client)
	if err != nil {
		logger.Error("[Watcher] Error while CrawlPast: URL=%s, StockID=%s, Page=%d, error=%+v",
			pastURLFormat, stockID, page, err,
		)
		return nil
	}

	daySise := soup.HTMLParse(response)
	handleSoupError(daySise)

	daySiseContent := daySise.Find("table", "class", "type2")
	handleSoupError(daySiseContent)

	daySiseContent = daySiseContent.Find("tbody")
	handleSoupError(daySiseContent)

	priceContents := daySiseContent.FindAll("tr", "onmouseover", "mouseOver(this)")
	if priceContents == nil || len(priceContents) == 0 {
		return nil
	}
	result := make([]structs.StockPrice, len(priceContents))
	for i, row := range priceContents {
		rowContents := row.FindAll("span")
		if len(rowContents) == 0 {
			return nil
		}
		rowTimestamp := commons.GetTimestamp(dateFormat, rowContents[0].Text())
		rowClose := commons.GetInt(rowContents[1].Text())
		rowOpen := commons.GetInt(rowContents[3].Text())
		rowHigh := commons.GetInt(rowContents[4].Text())
		rowLow := commons.GetInt(rowContents[5].Text())
		rowVolumn := commons.GetDouble(rowContents[6].Text())

		result[i] = structs.StockPrice{
			StockID:   stockID,
			Timestamp: rowTimestamp,
			Open:      rowOpen,
			Close:     rowClose,
			High:      rowHigh,
			Low:       rowLow,
			Volume:    rowVolumn,
		}
	}
	return result
}

// CrawlNow actually performs crawling for the current prices
func CrawlNow(stockID string, page int) structs.StockPrice {
	response, err := soup.Get(fmt.Sprintf(nowURLFormat, stockID))
	if err != nil {
		logger.Error("[Watcher] Error while CrawlNow: URL=%s, StockID=%s, response=%s, error=%+v",
			pastURLFormat, stockID, response, err,
		)
		return structs.StockPrice{Close: -1}
	}

	daySise := soup.HTMLParse(response)
	handleSoupError(daySise)

	nowSise := daySise.Find("div", "id", "chart_area")
	handleSoupError(nowSise)

	nowSise = daySise.Find("div", "class", "today")
	handleSoupError(nowSise)

	nowSise = nowSise.Find("em")
	handleSoupError(nowSise)

	nowSise = nowSise.Find("span", "class", "blind")
	handleSoupError(nowSise)

	price := commons.GetInt(nowSise.Text())
	timestamp := commons.Now().Unix()

	stockPrice := structs.StockPrice{Close: price, Timestamp: timestamp, StockID: stockID}
	return stockPrice
}

func handleSoupError(r soup.Root) {
	if r.Pointer == nil {
		logger.Panic("[Watcher] handleSoupError: %+v", r.Error)
	}
}
