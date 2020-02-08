package analyser

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
)

func TestCandlePlotterValidity(t *testing.T) {
	dbClient := prepareDBClient()
	defer func() {
		dbClient.Close()
	}()

	const savePath = "images/candle1.png"

	info := structs.Stock{Name: "Korean Air", StockID: "003490", MarketType: structs.KOSPI}
	ana := NewAnalyser(info.StockID)

	timestampFrom := ana.NeedPriceFrom()
	var prices []structs.StockPrice
	_, err := dbClient.Select(&prices,
		"where StockID=? and Timestamp>=? order by Timestamp",
		info.StockID, timestampFrom)
	if err != nil {
		t.Fatal(err)
	}

	// indiFuncs := func(name string, args ...interface{}) techan.Indicator {
	// 	generator := indicatorMap[name]
	// 	f, err := generator(ana.timeSeries, args...)
	// 	if err != nil {
	// 		t.Fatal(name, err)
	// 	}
	// 	return f
	// }

	// Price
	// f0 := indiFuncs("price()")

	candles := Candles{}
	for i := range prices {
		ana.AppendPastPrice(prices[i])
		candles = append(candles, Candle{
			Timestamp: float64(prices[i].Timestamp),
			Open:      float64(prices[i].Open),
			Close:     float64(prices[i].Close),
			High:      float64(prices[i].High),
			Low:       float64(prices[i].Low)})
	}

	// Plot Candles
	p, err := plot.New()
	if err != nil {
		panic(err)
	}
	p.Title.Text = fmt.Sprintf("Candles(%s)", info.Name)
	p.X.Label.Text = "Time"
	p.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02"}
	p.Y.Label.Text = "Price"

	candles = candles[len(candles)-100:]
	cs := NewCandleSticks(candles, color.RGBA{R: 128, A: 255}, color.RGBA{B: 120, A: 255}, 0, 10)
	p.Add(cs)

	if err := p.Save(15*vg.Inch, 5*vg.Inch, savePath); err != nil {
		panic(err)
	}

}
