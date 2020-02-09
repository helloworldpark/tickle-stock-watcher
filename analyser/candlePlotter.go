package analyser

import (
	"fmt"
	"image/color"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
	"github.com/sdcoffey/techan"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
)

const savePath = "images/candle1.png"

// NewCandlePlot draws and saves a new candle plot of Stock ID
func NewCandlePlot(dbClient *database.DBClient, days int, stockID string, stockAccess *watcher.StockItemChecker) (bool, string) {

	stockInfo, isValid := stockAccess.StockFromID(stockID)
	if !isValid {
		logger.Error("[CandlePlot] Error: No valid stock item corresponding to %s", stockID)
		return false, ""
	}

	ana := NewAnalyser(stockInfo.StockID)
	timestampFrom := commons.MaxInt64(ana.NeedPriceFrom(), commons.Now().Unix()-60*60*24*int64(days+1))
	var prices []structs.StockPrice
	_, err := dbClient.Select(&prices,
		"where StockID=? and Timestamp>=? order by Timestamp",
		stockID, timestampFrom)
	if err != nil {
		logger.Error("[CandlePlot] Error: +v", err)
		return false, ""
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
	y, m, d := commons.Now().Date()
	p.Title.Text = fmt.Sprintf("%4d.%02d.%02d#%s", y, m, d, stockInfo.StockID)
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

	return true, savePath
}

// NewProspect find new prospect of the day
func NewProspect(dbClient *database.DBClient, days int, stockID string) []structs.StockPrice {
	ana := NewAnalyser(stockID)
	timestampFrom := commons.MaxInt64(ana.NeedPriceFrom(), commons.Now().Unix()-60*60*24*int64(days+10))
	var prices []structs.StockPrice
	_, err := dbClient.Select(&prices,
		"where StockID=? and Timestamp>=? order by Timestamp",
		stockID, timestampFrom)
	if err != nil {
		logger.Error("[CandlePlotter] Error: +v", err)
		return nil
	}

	indiFuncs := func(name string, args ...interface{}) techan.Indicator {
		generator := indicatorMap[name]
		f, err := generator(ana.timeSeries, args...)
		if err != nil {
			logger.Error("Error at %s: +v", name, err)
		}
		return f
	}
	// MACD > 0 && MACDHist == 0
	const zeroLag = 1
	const zeroSamples = 7
	f0 := indiFuncs("macd", 12.0, 26.0)
	f1 := indiFuncs("macdhist", 12.0, 26.0, 9.0)
	smoothSpline := newSmoothSplineCalculator(f1, zeroLag, zeroSamples)

	var promisingPrices []structs.StockPrice
	for i := range prices {
		ana.AppendPastPrice(prices[i])

		v0 := f0.Calculate(i).Float()
		if v0 <= 0 {
			continue
		}

		g := smoothSpline.Graph(i)
		if len(g) < 7 {
			continue
		}

		// if g[6] > 0 {
		// 	continue
		// }

		isIncreasing := g[6] > g[5] && g[5] > g[4]
		if !isIncreasing {
			continue
		}

		promisingPrices = append(promisingPrices, prices[i])
	}

	return promisingPrices
}
