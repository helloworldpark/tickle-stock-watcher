package analyser

import (
	"fmt"
	"image/color"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
)

const savePath = "images/candle1.png"

func NewCandlePlotter(dbClient *database.DBClient, days int, stockID string, stockAccess *watcher.StockItemChecker) bool {

	stockInfo, isValid := stockAccess.StockFromID(stockID)
	if !isValid {
		logger.Error("[CandlePlotter] Error: No valid stock item corresponding to %s", stockID)
		return false
	}

	ana := NewAnalyser(stockInfo.StockID)
	timestampFrom := commons.MaxInt64(ana.NeedPriceFrom(), commons.Now().Unix()-60*60*24*int64(days+1))
	var prices []structs.StockPrice
	_, err := dbClient.Select(&prices,
		"where StockID=? and Timestamp>=? order by Timestamp",
		stockID, timestampFrom)
	if err != nil {
		logger.Error("[CandlePlotter] Error: +v", err)
		return false
	}

	candles := Candles{}
	for i := range prices {
		ana.AppendPastPrice(prices[i])
		candles = append(candles, Candle{
			Timestamp: float64(prices[i].Timestamp),
			Open:      float64(prices[i].Open),
			Close:     float64(prices[i].Close),
			High:      float64(prices[i].High),
			Low:       float64(prices[i].Low),
		})
	}

	// Plot Candles
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = fmt.Sprintf("Candles(#%s)", stockInfo.StockID)
	fmt.Println(p.Title.Text)
	p.X.Label.Text = "Time"
	p.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02"}
	p.Y.Label.Text = "Price"

	if len(candles) >= (days + 1) {
		candles = candles[len(candles)-days-1:]
	}

	cs := NewCandleSticks(candles, ana.timeSeries, color.RGBA{R: 128, A: 255}, color.RGBA{B: 120, A: 255})
	p.Add(cs)

	if err := p.Save(vg.Length(days)*vg.Centimeter, vg.Length(days)*vg.Centimeter, savePath); err != nil {
		panic(err)
	}

	return true
}
