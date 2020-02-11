package analyser

import (
	"image/color"
	"math"

	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/sdcoffey/techan"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// Candles array of candle
type Candles []candle

// Len length of the array
func (c Candles) Len() int { return len(c) }

// Candle timestamp, open, close, high, low
func (c Candles) Candle(i int) (float64, float64, float64, float64, float64) {
	return c[i].Timestamp, c[i].Open, c[i].Close, c[i].High, c[i].Low
}

type candler interface {
	Len() int
	Candle(int) (float64, float64, float64, float64, float64)
}

type candle struct {
	Timestamp, Open, Close, High, Low float64
}

func copyCandles(data candler) Candles {
	cp := make(Candles, data.Len())
	for i := range cp {
		cp[i].Timestamp, cp[i].Open, cp[i].Close, cp[i].High, cp[i].Low = data.Candle(i)
	}
	return cp
}

// CandleSticks struct of candle sticks
type CandleSticks struct {
	Candles
	timeSeries         *techan.TimeSeries
	UpColor, DownColor color.Color
}

// NewCandleSticks factory method for candle sticks
func NewCandleSticks(cs Candles, timeSeries *techan.TimeSeries, up, down color.Color) *CandleSticks {
	cp := copyCandles(cs)
	return &CandleSticks{
		Candles:    cp,
		timeSeries: timeSeries,
		UpColor:    up,
		DownColor:  down,
	}
}

/* plot.Plot interface methods
 *
 */
// Plot Plot
func (cs *CandleSticks) Plot(c draw.Canvas, plt *plot.Plot) {
	trX, trY := plt.Transforms(&c)

	// Plot candlestick
	for _, d := range cs.Candles {
		x0 := trX(d.Timestamp)
		x1 := trX(d.Timestamp + 24*60*60) // 24시간
		y0 := trY(d.Open)
		y1 := trY(d.Close)

		if y0 <= y1 {
			c.SetColor(cs.UpColor)
		} else {
			c.SetColor(cs.DownColor)
		}

		var p vg.Rectangle
		p.Min = vg.Point{X: x0, Y: vg.Length(math.Min(float64(y0), float64(y1)))}
		p.Max = vg.Point{X: x1, Y: vg.Length(math.Max(float64(y0), float64(y1)))}
		c.Fill(p.Path())

		x0 = trX(d.Timestamp + 8*60*60)
		x1 = trX(d.Timestamp + 16*60*60) // 8시간
		y0 = trY(d.High)
		y1 = trY(d.Low)

		var q vg.Rectangle
		q.Min = vg.Point{X: x0, Y: y0}
		q.Max = vg.Point{X: x1, Y: y1}
		c.Fill(q.Path())
	}

	indiFuncs := func(name string, args ...interface{}) techan.Indicator {
		generator := indicatorMap[name]
		f, err := generator(cs.timeSeries, args...)
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
	for i, d := range cs.Candles {
		v0 := f0.Calculate(i).Float()
		if v0 <= 0 {
			continue
		}

		g := smoothSpline.Graph(i)
		if len(g) < 7 {
			continue
		}

		isIncreasing := g[6] > g[5] && g[5] > g[4]
		if !isIncreasing {
			continue
		}

		x0 := trX(d.Timestamp)
		x1 := trX(d.Timestamp + 24*60*60) // 24시간
		y0 := trY(d.Open)
		y1 := trY(d.Close)
		var q vg.Rectangle
		q.Min = vg.Point{X: x0, Y: vg.Length(math.Min(float64(y0), float64(y1)))}
		q.Max = vg.Point{X: x1, Y: vg.Length(math.Max(float64(y0), float64(y1)))}
		c.SetColor(color.RGBA{R: 255, G: 255, A: 255})
		c.Fill(q.Path())
	}
}

// DataRange DataRange
func (cs *CandleSticks) DataRange() (xmin, xmax, ymin, ymax float64) {
	xmin = cs.Candles[0].Timestamp
	xmax = cs.Candles[len(cs.Candles)-1].Timestamp + 9*60

	ymin = cs.Candles[0].Low
	ymax = cs.Candles[0].High

	for _, d := range cs.Candles {
		ymin = math.Min(ymin, d.Low)
		ymax = math.Max(ymax, d.High)
	}

	return xmin, xmax, ymin, ymax
}
