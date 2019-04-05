package watcher

import (
	"fmt"

	"github.com/anaskhan96/soup"
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

const (
	dateFormat   = "2006.01.02"
	pastURLormat = "https://finance.naver.com/item/sise_day.nhn?code=%s&page=%d"
)

// CrawlPast actually performs crawling
func CrawlPast(stockID string, page int) {
	response, err := soup.Get(fmt.Sprintf(pastURLormat, stockID, page))
	if err != nil {
		logger.Error("[Watcher] %s", err.Error())
		return
	}

	daySise := soup.HTMLParse(response)
	if daySise.Pointer == nil {
		logger.Error("[Watcher] %s", daySise.Error.Error())
		return
	}

	daySiseContent := daySise.Find("table", "class", "type2").Find("tbody")
	if daySiseContent.Pointer == nil {
		logger.Error("[Watcher] %s", daySiseContent.Error.Error())
		return
	}

	priceContents := daySiseContent.FindAll("tr", "onmouseover", "mouseOver(this)")
	for _, row := range priceContents {
		rowContents := row.FindAll("span")
		rowDate := commons.GetTimestamp(dateFormat, rowContents[0].Text())
		rowClose := commons.GetInt(rowContents[1].Text())
		rowOpen := commons.GetInt(rowContents[3].Text())
		rowHigh := commons.GetInt(rowContents[4].Text())
		rowLow := commons.GetInt(rowContents[5].Text())
		rowVolumn := commons.GetDouble(rowContents[6].Text())

		logger.Info("[Watcher] %d %d %d %d %d %f", rowDate, rowClose, rowOpen, rowHigh, rowLow, rowVolumn)
	}

}
