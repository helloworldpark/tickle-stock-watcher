package analyser

import (
	"image/color"
	"math"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

type candler interface {
	Len() int
	Candle(int) (float64, float64, float64, float64, float64)
}

type Candle struct {
	Timestamp, Open, Close, High, Low float64
}

type Candles []Candle

func (c Candles) Len() int { return len(c) }

func (c Candles) Candle(i int) (float64, float64, float64, float64, float64) {
	return c[i].Timestamp, c[i].Open, c[i].Close, c[i].High, c[i].Low
}

func CopyCandles(data candler) Candles {
	cp := make(Candles, data.Len())
	for i := range cp {
		cp[i].Timestamp, cp[i].Open, cp[i].Close, cp[i].High, cp[i].Low = data.Candle(i)
	}
	return cp
}

type CandleSticks struct {
	Candles
	UpColor, DownColor   color.Color
	MinHeight, MaxHeight vg.Length
}

func NewCandleSticks(cs Candles, up, down color.Color, min, max vg.Length) *CandleSticks {
	cp := CopyCandles(cs)
	return &CandleSticks{
		Candles:   cp,
		UpColor:   up,
		DownColor: down,
		MinHeight: min,
		MaxHeight: max,
	}
}

func (cs *CandleSticks) Plot(c draw.Canvas, plt *plot.Plot) {
	trX, trY := plt.Transforms(&c)

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
}

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
