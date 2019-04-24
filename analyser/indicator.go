package analyser

import (
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
	return techan.NewMACDIndicator(techan.NewClosePriceIndicator(series), shortWindow, longWindow)
}

func newMACDHist(series *techan.TimeSeries, shortWindow, longWindow, signalWindow int) techan.Indicator {
	macd := newMACD(series, shortWindow, longWindow)
	return techan.NewMACDHistogramIndicator(macd, signalWindow)
}
