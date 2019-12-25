package analyser

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

func newTestAnalyser() *Analyser {
	coininfo := structs.CoinInfo{Vendor: "Gopax", Currency: "BTC"}
	analyser := NewAnalyser(coininfo)
	for i := 0; i < 100; i++ {
		start := time.Date(0, 0, i, 0, 0, 0, 0, time.UTC)
		candle := techan.NewCandle(techan.NewTimePeriod(start, time.Hour*6))
		base := (math.Sin(float64(0.1*float64(i)))+1)*10000 + 200
		candle.ClosePrice = big.NewDecimal(base)
		candle.MaxPrice = big.NewDecimal(base + rand.Float64()*100)
		candle.MinPrice = big.NewDecimal(base - rand.Float64()*100)
		candle.Volume = big.NewDecimal(rand.Float64() * 1000)
		analyser.timeSeries.AddCandle(candle)
	}
	return analyser
}

func prepareDBClient() *database.DBClient {
	credential := database.LoadCredential("/Users/shp/Documents/projects/github.com/helloworldpark/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()
	client.RegisterStructFromRegisterables([]database.DBRegisterable{
		// structs.CoinInfo{},
		structs.CoinPrice{},
		// structs.Invitation{},
		// structs.TradeCredential{},
		// structs.User{},
		// structs.UserStrategy{},
	})
	return client
}

func TestAppendPastPrice(t *testing.T) {
	dbClient := prepareDBClient()
	defer dbClient.Close()

	coininfo := structs.CoinInfo{Vendor: "Gopax", Currency: "BTC"}

	analyser := NewAnalyser(coininfo)
	analyser.NeedPriceFrom()

	timestampFrom := analyser.NeedPriceFrom()

	var prices []structs.CoinPrice
	_, err := dbClient.Select(&prices,
		"where Vendor=? and Currency=? and Timestamp>=? order by Timestamp",
		coininfo.Vendor, coininfo.Currency, timestampFrom)
	if err != nil {
		t.Fatal(err)
	}
	for i := range prices {
		analyser.AppendPastPrice(prices[i])
		fmt.Printf("Price[%d]:%v\n%v\n", i, commons.Unix(prices[i].Timestamp), analyser.timeSeries.LastCandle())
	}
}

func TestTechanValidity(t *testing.T) {
	dbClient := prepareDBClient()
	defer func() {
		dbClient.Close()
	}()

	coininfo := structs.CoinInfo{Vendor: "Gopax", Currency: "BCH"}
	ana := NewAnalyser(coininfo)

	timestampFrom := ana.NeedPriceFrom()
	var prices []structs.CoinPrice
	_, err := dbClient.Select(&prices,
		"where Vendor=? and Currency=? and Timestamp>=? order by Timestamp",
		coininfo.Vendor, coininfo.Currency, timestampFrom)
	if err != nil {
		t.Fatal(err)
	}

	indiFuncs := func(name string, args ...interface{}) techan.Indicator {
		generator := indicatorMap[name]
		f, err := generator(ana.timeSeries, args...)
		if err != nil {
			t.Fatal(name, err)
		}
		return f
	}

	// MACD
	f0 := indiFuncs("macd", 12.0, 26.0)
	f1 := indiFuncs("macdhist", 12.0, 26.0, 9.0)

	f2 := indiFuncs("extrema", f1, 1.0, 5.0)
	zeroLag := 1.0
	zeroSamples := 31.0
	f3 := indiFuncs("zero", f1, zeroLag, zeroSamples)
	f4 := indiFuncs("mflow", 28.0)

	macdValues := plotter.XYs{}
	macdHistValues := plotter.XYs{}
	extremas := plotter.XYs{}
	zeros := plotter.XYs{}
	mflows := plotter.XYs{}
	for i := range prices {
		ana.AppendPastPrice(prices[i])

		v0 := f0.Calculate(ana.timeSeries.LastIndex()).Float()
		v1 := f1.Calculate(ana.timeSeries.LastIndex()).Float()
		v2 := f2.Calculate(ana.timeSeries.LastIndex()).Float()
		v3 := f3.Calculate(ana.timeSeries.LastIndex()).Float()
		v4 := f4.Calculate(ana.timeSeries.LastIndex()).Float()

		macdValues = append(macdValues, plotter.XY{X: float64(prices[i].Timestamp), Y: v0})
		macdHistValues = append(macdHistValues, plotter.XY{X: float64(prices[i].Timestamp), Y: v1})
		if v2 == 5.0 {
			extremas = append(extremas, plotter.XY{X: float64(prices[i].Timestamp), Y: v1})
		}
		if v0 > 0 && v3 == 1.0 {
			zeros = append(zeros, plotter.XY{X: float64(prices[i].Timestamp), Y: v1})
		}
		mflows = append(mflows, plotter.XY{X: float64(prices[i].Timestamp), Y: v4})
	}

	// Plot MACD
	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	p.Title.Text = fmt.Sprintf("MACD Hist(Zero: %s, (%d, %d))", "Spline", int(zeroLag), int(zeroSamples))
	p.X.Label.Text = "X"
	p.Y.Label.Text = "Y"

	var timestampX []string
	for i := range macdHistValues {
		timestampX = append(timestampX, commons.Unix(int64(macdHistValues[i].X)).Format("01/02 15:04"))
	}

	macdL, _ := macdHistValues.XY(0)
	horLineL := plotter.XY{X: macdL, Y: 0.0}
	macdR, _ := macdHistValues.XY(len(macdHistValues) - 1)
	horLineR := plotter.XY{X: macdR, Y: 0.0}
	horLineValues := plotter.XYs{horLineL, horLineR}
	err = plotutil.AddLinePoints(p, horLineValues)

	macdHistPlotter, err := plotter.NewLine(macdHistValues)
	macdHistPlotter.Color = color.RGBA{R: 255, G: 10, B: 46, A: 255}
	macdHistPlotter.LineStyle.Color = macdHistPlotter.Color
	macdHistPlotter.StepStyle = plotter.MidStep
	macdHistPlotter.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}
	macdHistPlotter.LineStyle.Dashes = macdHistPlotter.Dashes

	macdValuesPlotter, err := plotter.NewLine(macdValues)
	macdValuesPlotter.Color = color.RGBA{R: 10, G: 253, B: 10, A: 255}
	macdValuesPlotter.LineStyle.Color = macdValuesPlotter.Color
	macdValuesPlotter.Dashes = []vg.Length{vg.Points(1), vg.Points(1)}
	macdValuesPlotter.LineStyle.Dashes = macdValuesPlotter.Dashes

	err = plotutil.AddLines(p, macdHistPlotter, macdValuesPlotter)

	s1, err := plotter.NewScatter(extremas)
	s1.Color = color.RGBA{129, 23, 255, 255}
	s1.Shape = draw.PyramidGlyph{}
	s1.GlyphStyle.Color = s1.Color
	s1.GlyphStyle.Shape = s1.Shape
	s1.GlyphStyle.Radius = vg.Centimeter * 0.2

	s2, err := plotter.NewScatter(zeros)
	s2.Color = color.RGBA{35, 200, 10, 255}
	s2.Shape = draw.CircleGlyph{}
	s2.Radius = vg.Millimeter * 1
	s2.GlyphStyle.Color = s2.Color
	s2.GlyphStyle.Shape = s2.Shape
	s2.GlyphStyle.Radius = s2.Radius

	labels := plotter.XYLabels{Labels: timestampX}
	l1, err := plotter.NewLabels(labels)
	p.Add(s2, l1)

	p.Legend.Add("MACD Hist", macdHistPlotter)
	p.Legend.Add("MACD Values", macdValuesPlotter)
	p.Legend.Add("Extremas", s1)
	p.Legend.Add("Zeros", s2)

	if err != nil {
		panic(err)
	}
	if err := p.Save(15*vg.Inch, 5*vg.Inch, "/Users/shp/Documents/projects/github.com/helloworldpark/tickle-stock-watcher/zero5.png"); err != nil {
		panic(err)
	}

}

func TestRuleGeneration(t *testing.T) {
	handleErr := func(err error) {
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("------------------")
	}
	analyser := newTestAnalyser()
	tokens, err := parseTokens("(macd(12,26) > 0) && (zero(macdhist(12,26,9),1,15)==1)")
	handleErr(err)
	tokens, err = tidyTokens(tokens)
	handleErr(err)
	fcns, err := reorderTokenByPostfix(tokens)
	handleErr(err)
	for _, f := range fcns {
		fmt.Println(f.t.Kind, f.t.Value, f.argc)
	}
	event, err := analyser.createEvent(fcns, techan.BUY, func(trader *Trader, price structs.CoinPrice, orderSide int, userid int64, repeat bool) {
		fmt.Println("Event Callback: ", price.Close, orderSide, userid, repeat)
	})
	handleErr(err)

	for i := 0; i < 100; i++ {
		fmt.Println(event.IsTriggered(i, nil))
	}
}
