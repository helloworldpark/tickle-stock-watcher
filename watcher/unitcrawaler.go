package watcher

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/anaskhan96/soup"
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

const (
	pastURLormat = "https://finance.naver.com/item/sise_day.nhn?code=%s&page=%d"
)

// UnitCrawler is a crawler for a single stock
type UnitCrawler struct {
	Stock commons.Stock
}

// Crawl actually performs crawling
func (worker UnitCrawler) Crawl(page int) {
	response, err := soup.Get(fmt.Sprintf(pastURLormat, worker.Stock.StockID, page))
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
		rowDate := getDate(rowContents[0])
		rowClose := getInteger(rowContents[1])
		rowOpen := getInteger(rowContents[3])
		rowHigh := getInteger(rowContents[4])
		rowLow := getInteger(rowContents[5])
		rowVolumn := getFloat(rowContents[6])

		logger.Info("[Watcher] %s %d %d %d %d %f", rowDate, rowClose, rowOpen, rowHigh, rowLow, rowVolumn)
	}

}

func getInteger(r soup.Root) int {
	val, err := strconv.ParseInt(strings.ReplaceAll(r.Text(), ",", ""), 10, 32)
	if err != nil {
		logger.Panic("[Watcher] %s", err.Error())
	}
	return int(val)
}

func getDate(r soup.Root) string {
	return r.Text()
}

func getFloat(r soup.Root) float64 {
	val, err := strconv.ParseFloat(strings.ReplaceAll(r.Text(), ",", ""), 64)
	if err != nil {
		logger.Panic("[Watcher] %s", err.Error())
	}
	return val
}
