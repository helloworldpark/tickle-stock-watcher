package analyser

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/sdcoffey/techan"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

func TestTechanValidity(t *testing.T) {
	dbClient := prepareDBClient()
	defer func() {
		dbClient.Close()
	}()

	const savePath = "images/zero1.png"

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
	zeroSamples := 7.0
	f3 := indiFuncs("zero", f1, zeroLag, zeroSamples)
	f4 := indiFuncs("mflow", 28.0)

	smoothSpline := newSmoothSplineCalculator(f1, int(zeroLag), int(zeroSamples))

	macdValues := plotter.XYs{}
	macdHistValues := plotter.XYs{}
	extremas := plotter.XYs{}
	zeros := plotter.XYs{}
	mflows := plotter.XYs{}
	smooths := []plotter.XYs{}
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
			newSmooth := smoothSpline.Graph(ana.timeSeries.LastIndex())
			var smoothxy plotter.XYs
			for j := 0; j < len(newSmooth); j++ {
				smoothxy = append(smoothxy, plotter.XY{
					X: float64(prices[i+1-(len(newSmooth)-j)].Timestamp),
					Y: newSmooth[j],
				})
			}
			smooths = append(smooths, smoothxy)
		}
		mflows = append(mflows, plotter.XY{X: float64(prices[i].Timestamp), Y: v4})
	}

	// Plot MACD
	p, err := plot.New()
	if err != nil {
		panic(err)
	}

	macdL, _ := macdHistValues.XY(0)
	horLineL := plotter.XY{X: macdL, Y: 0.0}
	macdR, _ := macdHistValues.XY(len(macdHistValues) - 1)
	horLineR := plotter.XY{X: macdR, Y: 0.0}
	horLineValues := plotter.XYs{horLineL, horLineR}
	horLine, err := plotter.NewLine(horLineValues)
	horLine.LineStyle.Color = color.RGBA{A: 255}
	p.Add(horLine)

	if err != nil {
		t.Error(err)
	}
	macdHistPlotter, err := plotter.NewLine(macdHistValues)
	if err != nil {
		t.Error(err)
	}
	macdHistPlotter.LineStyle.Width = vg.Points(1)
	macdHistPlotter.LineStyle.Color = color.RGBA{R: 0, G: 10, B: 46, A: 255}
	macdHistPlotter.LineStyle.Dashes = []vg.Length{vg.Points(3), vg.Points(3)}

	macdValuesPlotter, err := plotter.NewLine(macdValues)
	macdValuesPlotter.LineStyle.Color = color.RGBA{R: 10, G: 253, B: 10, A: 255}
	macdValuesPlotter.LineStyle.Dashes = []vg.Length{vg.Points(1), vg.Points(1)}

	p.Add(macdHistPlotter)
	p.Add(macdValuesPlotter)
	for _, smoothxy := range smooths {
		smoothLine, _ := plotter.NewLine(smoothxy)
		smoothLine.LineStyle.Color = color.RGBA{R: 255, G: 0, B: 196, A: 255}
		p.Add(smoothLine)

		var smoothEnds plotter.XYs
		smoothEnds = append(smoothEnds, smoothxy[0])
		smoothEnds = append(smoothEnds, smoothxy[len(smoothxy)-1])
		smoothPoint, _ := plotter.NewScatter(smoothEnds)
		smoothPoint.Color = smoothLine.LineStyle.Color
		smoothPoint.Shape = draw.CrossGlyph{}
		p.Add(smoothPoint)

		fmt.Printf("From: %v -> To: %v\n",
			commons.Unix(int64(smoothEnds[0].X)),
			commons.Unix(int64(smoothEnds[1].X)),
		)
	}

	s1, err := plotter.NewScatter(extremas)
	s1.Color = color.RGBA{129, 23, 255, 255}
	s1.Shape = draw.PyramidGlyph{}
	s1.Radius = vg.Millimeter * 1
	s1.GlyphStyle.Color = s1.Color
	s1.GlyphStyle.Shape = s1.Shape
	s1.GlyphStyle.Radius = s1.Radius
	p.Add(s1)

	s2, err := plotter.NewScatter(zeros)
	s2.Color = color.RGBA{35, 200, 10, 255}
	s2.Shape = draw.CircleGlyph{}
	s2.Radius = vg.Millimeter * 1
	s2.GlyphStyle.Color = s2.Color
	s2.GlyphStyle.Shape = s2.Shape
	s2.GlyphStyle.Radius = s2.Radius
	p.Add(s2)

	p.Legend.Add("MACD Hist", macdHistPlotter)
	p.Legend.Add("MACD Values", macdValuesPlotter)
	p.Legend.Add("Extremas", s1)
	p.Legend.Add("Zeros", s2)

	p.Title.Text = fmt.Sprintf("MACD Hist(Zero: %s, (%d, %d))", "Spline", int(zeroLag), int(zeroSamples))
	p.X.Label.Text = "X"
	p.Y.Label.Text = "Y"
	p.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04"}

	if err != nil {
		panic(err)
	}
	if err := p.Save(15*vg.Inch, 5*vg.Inch, savePath); err != nil {
		panic(err)
	}

}
