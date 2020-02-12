package analyser

import (
	"math"

	"github.com/helloworldpark/gonaturalspline/cubicSpline"
	"github.com/helloworldpark/gonaturalspline/knot"
	"github.com/sajari/regression"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

type dualOperatorIndicator struct {
	lhs techan.Indicator
	rhs techan.Indicator
}

type plusIndicator struct {
	dualOperatorIndicator
}

func newPlusIndicator(lhs, rhs techan.Indicator) techan.Indicator {
	return plusIndicator{dualOperatorIndicator{lhs: lhs, rhs: rhs}}
}

func (id plusIndicator) Calculate(index int) big.Decimal {
	return id.lhs.Calculate(index).Add(id.rhs.Calculate(index))
}

type minusIndicator struct {
	dualOperatorIndicator
}

func newMinusIndicator(lhs, rhs techan.Indicator) techan.Indicator {
	return minusIndicator{dualOperatorIndicator{lhs: lhs, rhs: rhs}}
}

func (id minusIndicator) Calculate(index int) big.Decimal {
	return id.lhs.Calculate(index).Sub(id.rhs.Calculate(index))
}

type multIndicator struct {
	dualOperatorIndicator
}

func newMultiplyIndicator(lhs, rhs techan.Indicator) techan.Indicator {
	return multIndicator{dualOperatorIndicator{lhs: lhs, rhs: rhs}}
}

func (id multIndicator) Calculate(index int) big.Decimal {
	return id.lhs.Calculate(index).Mul(id.rhs.Calculate(index))
}

type divIndicator struct {
	dualOperatorIndicator
}

func newDivideIndicator(lhs, rhs techan.Indicator) techan.Indicator {
	return divIndicator{dualOperatorIndicator{lhs: lhs, rhs: rhs}}
}

func (id divIndicator) Calculate(index int) big.Decimal {
	return id.lhs.Calculate(index).Div(id.rhs.Calculate(index))
}

type negateIndicator struct {
	indicator techan.Indicator
}

func newNegateIndicator(indicator techan.Indicator) techan.Indicator {
	return negateIndicator{indicator: indicator}
}

func newNegateIndicatorFromFloat(c float64) techan.Indicator {
	constIndicator := techan.NewConstantIndicator(c)
	return negateIndicator{indicator: constIndicator}
}

func (ni negateIndicator) Calculate(index int) big.Decimal {
	return ni.indicator.Calculate(index).Neg()
}

func newRSI(series *techan.TimeSeries, timeframe int) techan.Indicator {
	return techan.NewRelativeStrengthIndexIndicator(techan.NewClosePriceIndicator(series), timeframe)
}

func newMACD(series *techan.TimeSeries, shortWindow, longWindow int) techan.Indicator {
	return NewCustomMACDIndicator(techan.NewClosePriceIndicator(series), shortWindow, longWindow)
}

func newMACDHist(series *techan.TimeSeries, shortWindow, longWindow, signalWindow int) techan.Indicator {
	macd := newMACD(series, shortWindow, longWindow)
	return NewCustomMACDHistogramIndicator(macd, signalWindow)
}

type moneyFlowIndexIndicator struct {
	series *techan.TimeSeries
	window int
}

func (id *moneyFlowIndexIndicator) typicalPrice(index int) big.Decimal {
	candle := id.series.Candles[index]
	return candle.MinPrice.Add(candle.MaxPrice).Add(candle.ClosePrice).Div(big.NewDecimal(3))
}

// https://school.stockcharts.com/doku.php?id=technical_indicators:money_flow_index_mfi#calculation
func (id *moneyFlowIndexIndicator) Calculate(index int) big.Decimal {
	if index < id.window+1 {
		return big.NewDecimal(100)
	}
	idx := index - id.window - 1
	lastTypicalPrice := id.typicalPrice(idx)
	positiveMflow := big.ZERO
	negativeMflow := big.ZERO
	for idx < index {
		idx++
		currentTypicalPrice := id.typicalPrice(idx)
		volume := id.series.Candles[idx].Volume
		isPositive := currentTypicalPrice.Cmp(lastTypicalPrice)
		if isPositive == 1 { // current > last
			positiveMflow = positiveMflow.Add(currentTypicalPrice.Mul(volume))
		} else if isPositive == -1 { // current < last
			negativeMflow = negativeMflow.Add(currentTypicalPrice.Mul(volume))
		}
		lastTypicalPrice = currentTypicalPrice
	}
	if negativeMflow.Zero() {
		return big.NewDecimal(100)
	}
	moneyRate := positiveMflow.Div(negativeMflow)
	moneyFlow := moneyRate.Div(big.ONE.Add(moneyRate)).Mul(big.NewDecimal(100))
	return moneyFlow
}

func newMoneyFlowIndex(series *techan.TimeSeries, window int) techan.Indicator {
	return &moneyFlowIndexIndicator{
		series: series,
		window: window,
	}
}

type lagDifferenceIndicator struct {
	indicator techan.Indicator
	lag       int
}

func newIncreaseIndicator(indicator techan.Indicator, lag int) techan.Indicator {
	return lagDifferenceIndicator{indicator: indicator, lag: lag}
}

func (ld lagDifferenceIndicator) Calculate(index int) big.Decimal {
	latest := ld.indicator.Calculate(index)
	before := ld.indicator.Calculate(index - ld.lag)
	return latest.Sub(before)
}

type localExtremaIndicator struct {
	indicator techan.Indicator
	lag       int
	samples   int
}

func newLocalExtremaIndicator(indicator techan.Indicator, lag, samples int) techan.Indicator {
	return localExtremaIndicator{indicator: indicator, lag: lag, samples: samples}
}

type vec4 [4]float64
type extrema struct {
	minima float64
	maxima float64
}

func newVec4(v0, v1, v2, v3 float64) vec4 {
	return [4]float64{v0, v1, v2, v3}
}

// Calculate returns integer values indicating state
// -1: Invalid state
// 0 : Increasing (but its speed is decreasing)
//      /
//     /
// 1 : Increasing but local maximum is expected to come
//      -
//     /
// 2 : Decreasing and local maximum was before
//      -
//       \
// 3 : Decreasing
//        \
//         \
// 4 : Decreasing but local minimum expected
//           \
//            -
// 5 : Increasing and local minimum before
//                 /
//               -
// 6 : Increasing (but its speed is increasing)
func (ld localExtremaIndicator) Calculate(index int) big.Decimal {
	r := new(regression.Regression)
	dataAdded := 0
	for i := 0; i < ld.samples; i++ {
		idx := index - i*ld.lag
		if idx < 0 {
			continue
		}
		t := float64(ld.samples - i - 1)
		p := ld.indicator.Calculate(idx).Float()
		r.Train(regression.DataPoint(p, []float64{t, t * t, t * t * t}))
		dataAdded++
	}
	if dataAdded < ld.samples {
		return big.NewDecimal(-1)
	}
	r.Run()

	c := newVec4(r.Coeff(0), r.Coeff(1), r.Coeff(2), r.Coeff(3))
	if !hasLocalExtrema(c) {
		if c[3] > 0 {
			return big.NewDecimal(6)
		}
		return big.NewDecimal(3)
	}
	extrema := findLocalExtrema(c)
	f1 := derivative(c)
	f2 := curvature(c)
	now := float64(ld.samples - 1)

	increasing := f1(now) > 0
	curvIncreasing := f2(now) > 0
	distMinima := math.Abs(now - extrema.minima)
	distMaxima := math.Abs(now - extrema.maxima)
	inExtrema := distMinima < float64(ld.lag) || distMaxima < float64(ld.lag)

	if !curvIncreasing {
		if increasing {
			if inExtrema {
				return big.NewDecimal(1)
			}
			return big.NewDecimal(0)
		}
		if inExtrema {
			return big.NewDecimal(2)
		}
		return big.NewDecimal(3)
	}
	if increasing {
		if inExtrema {
			return big.NewDecimal(5)
		}
		return big.NewDecimal(6)
	}
	if inExtrema {
		return big.NewDecimal(4)
	}
	return big.NewDecimal(3)
}

func hasLocalExtrema(f vec4) bool {
	return math.Abs(f[3]) > 0.0 && f[2]*f[2]-3.0*f[1]*f[3] > 0.0
}

func findLocalExtrema(f vec4) extrema {
	q := f[2]*f[2] - 3.0*f[1]*f[3]
	d := math.Sqrt(q)
	x1 := f[1] / (-f[2] + d)
	x2 := f[1] / (-f[2] - d)
	var minima, maxima float64
	if f[3] < 0 {
		if x1 < x2 {
			minima = x1
			maxima = x2
		} else {
			minima = x2
			maxima = x1
		}
	} else {
		if x1 < x2 {
			minima = x2
			maxima = x1
		} else {
			minima = x1
			maxima = x2
		}
	}
	return extrema{minima: minima, maxima: maxima}
}

func value(f vec4) func(float64) float64 {
	return func(t float64) float64 {
		return f[0] + t*(f[1]+t*(f[2]+t*f[3]))
	}
}

func derivative(f vec4) func(float64) float64 {
	return func(t float64) float64 {
		return f[1] + t*(2*f[2]+t*3*f[3])
	}
}

func curvature(f vec4) func(float64) float64 {
	return func(t float64) float64 {
		return 2*f[2] + t*6*f[3]
	}
}

type localZeroIndicator struct {
	indicator techan.Indicator
	lag       int
	samples   int
}

func (ld localZeroIndicator) Calculate(index int) big.Decimal {
	// -1: Invalid
	//  0: No zero
	//  1: Zero exists, and increasing
	//  2: Zero exists, and decreasing
	r := new(regression.Regression)
	dataAdded := 0
	for i := 0; i < ld.samples; i++ {
		idx := index - i*ld.lag
		if idx < 0 {
			return big.NewDecimal(-1)
		}
		t := float64(ld.samples - i - 1)
		p := ld.indicator.Calculate(idx).Float()
		r.Train(regression.DataPoint(p, []float64{t, t * t, t * t * t}))
		dataAdded++
	}
	r.Run()

	c := newVec4(r.Coeff(0), r.Coeff(1), r.Coeff(2), r.Coeff(3))
	f0 := value(c)
	f1 := derivative(c)
	now := float64(ld.samples - 1)

	increasing := f1(now) > 0
	hasZero := f0(now) > 0 && f0(now*0.75) < 0
	if hasZero {
		if increasing {
			return big.NewDecimal(1)
		}
		return big.NewDecimal(2)
	}
	return big.NewDecimal(0)
}

// func newLocalZeroIndicator(indicator techan.Indicator, lag, samples int) techan.Indicator {
// 	return localZeroIndicator{indicator: indicator, lag: lag, samples: samples}
// }

// SmoothSplineCalculator Wrapper of cubicSpline.NaturalCubicSplines
type SmoothSplineCalculator struct {
	indicator techan.Indicator
	lag       int
	samples   int
	ncs       *cubicSpline.NaturalCubicSplines
	knots     knot.Knot
}

// Graph Returns local smooth spline values around the index
func (ld SmoothSplineCalculator) Graph(index int) []float64 {
	// -1: Invalid
	//  0: No zero
	//  1: Zero exists, and increasing
	//  2: Zero exists, and decreasing
	var y []float64
	for i := 0; i < ld.knots.Count(); i++ {
		idx := index + int(ld.knots.At(i))
		if idx < 0 {
			return nil
		}
		p := ld.indicator.Calculate(idx).Float()
		y = append(y, p)
	}
	ld.ncs.Interpolate(y)

	var yhat []float64
	for i := 0; i < ld.knots.Count(); i++ {
		yhat = append(yhat, ld.ncs.At(ld.knots.At(i)))
	}

	return yhat
}

func newSmoothSplineCalculator(indicator techan.Indicator, lag, samples int) SmoothSplineCalculator {
	end := 0
	start := -lag * (samples - 1)
	knots := knot.NewUniformKnot(float64(start), float64(end), samples, 1)
	ncs := cubicSpline.NewNaturalCubicSplines(knots, nil)
	const lambda = 0.1
	ncs.Solve(lambda)
	return SmoothSplineCalculator{indicator: indicator, lag: lag, samples: samples, ncs: ncs, knots: knots}
}

type localZeroSplineIndicator struct {
	indicator techan.Indicator
	lag       int
	samples   int
	ncs       *cubicSpline.NaturalCubicSplines
	knots     knot.Knot
}

func (ld localZeroSplineIndicator) Calculate(index int) big.Decimal {
	// -1: Invalid
	//  0: No zero
	//  1: Zero exists, and increasing
	//  2: Zero exists, and decreasing
	var y []float64
	for i := 0; i < ld.knots.Count(); i++ {
		idx := index + int(ld.knots.At(i))
		if idx < 0 {
			return big.NewDecimal(-1)
		}
		p := ld.indicator.Calculate(idx).Float()
		y = append(y, p)
	}
	ld.ncs.Interpolate(y)

	x0 := ld.knots.At(ld.knots.Count() - 2)
	v0 := ld.ncs.At(x0)
	x1 := ld.knots.At(ld.knots.Count() - 1)
	v1 := ld.ncs.At(x1)
	const t = 0.8
	x2 := ld.knots.At(0)*(1.0-t) + x1*t
	v2 := ld.ncs.At(x2)

	increasing := v1 > v0
	hasZero := v1 > 0 && v2 < 0
	if hasZero {
		if increasing {
			return big.NewDecimal(1)
		}
		return big.NewDecimal(2)
	}
	return big.NewDecimal(0)
}

func newLocalZeroIndicator(indicator techan.Indicator, lag, samples int) techan.Indicator {
	end := 0
	start := -lag * (samples - 1)
	knots := knot.NewUniformKnot(float64(start), float64(end), samples, 1)
	ncs := cubicSpline.NewNaturalCubicSplines(knots, nil)
	const lambda = 0.1
	ncs.Solve(lambda)
	return localZeroSplineIndicator{indicator: indicator, lag: lag, samples: samples, ncs: ncs, knots: knots}
}

// CustomEMAIndicator
type customEmaIndicator struct {
	techan.Indicator
	window      int
	resultCache []*big.Decimal
}

// NewCustomEMAIndicator returns a derivative indicator which returns the average of the current and preceding values in
// the given window, with values closer to current index given more weight. A more in-depth explanation can be found here:
// http://www.investopedia.com/terms/e/ema.asp
func NewCustomEMAIndicator(indicator techan.Indicator, window int) techan.Indicator {
	// emaIndiType := reflect.TypeOf(indicator)
	// if emaIndiType != nil {
	// 	logger.Info("EMA Indicator: %v", emaIndiType)
	// }
	return &customEmaIndicator{
		Indicator:   indicator,
		window:      window,
		resultCache: make([]*big.Decimal, 10),
	}
}

func (ema *customEmaIndicator) Calculate(index int) big.Decimal {
	if index == 0 {
		result := ema.Indicator.Calculate(index)
		return result
	} else if index+1 < ema.window {
		result := ema.Indicator.Calculate(index)
		ema.cacheResult(index, result)
		return result
	} else if len(ema.resultCache) > index {
		if len(ema.resultCache)-index == 1 {
			var emaPrevious = big.ZERO
			if ema.resultCache[index-1] != nil {
				emaPrevious = *ema.resultCache[index-1]
			}
			mult := big.NewDecimal(2.0 / float64(ema.window+1))
			result := ema.Indicator.Calculate(index).Sub(emaPrevious).Mul(mult).Add(emaPrevious)
			ema.cacheResult(index, result)
			return result
		}
		if ema.resultCache[index] != nil {
			return *ema.resultCache[index]
		}
	}

	emaPrev := ema.Calculate(index - 1)
	mult := big.NewDecimal(2.0 / float64(ema.window+1))
	result := ema.Indicator.Calculate(index).Sub(emaPrev).Mul(mult).Add(emaPrev)
	ema.cacheResult(index, result)

	return result
}

func (ema *customEmaIndicator) cacheResult(index int, val big.Decimal) {
	if index < len(ema.resultCache) {
		ema.resultCache[index] = &val
	} else {
		ema.resultCache = append(ema.resultCache, &val)
	}
}

// NewCustomMACDIndicator returns a derivative Indicator which returns the difference between two EMAIndicators with long and
// short windows. It's useful for gauging the strength of price movements. A more in-depth explanation can be found here:
// http://www.investopedia.com/terms/m/macd.asp
func NewCustomMACDIndicator(baseIndicator techan.Indicator, shortwindow, longwindow int) techan.Indicator {
	return techan.NewDifferenceIndicator(NewCustomEMAIndicator(baseIndicator, shortwindow), NewCustomEMAIndicator(baseIndicator, longwindow))
}

// NewCustomMACDHistogramIndicator returns a derivative Indicator based on the MACDIndicator, the result of which is
// the macd indicator minus it's signalLinewindow EMA. A more in-depth explanation can be found here:
// http://stockcharts.com/school/doku.php?id=chart_school:technical_indicators:macd-histogram
func NewCustomMACDHistogramIndicator(macdIdicator techan.Indicator, signalLinewindow int) techan.Indicator {
	return techan.NewDifferenceIndicator(macdIdicator, NewCustomEMAIndicator(macdIdicator, signalLinewindow))
}
