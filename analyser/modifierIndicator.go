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

func NewPlusIndicator(lhs, rhs techan.Indicator) techan.Indicator {
	return plusIndicator{dualOperatorIndicator{lhs: lhs, rhs: rhs}}
}

func (id plusIndicator) Calculate(index int) big.Decimal {
	return id.lhs.Calculate(index).Add(id.rhs.Calculate(index))
}

type minusIndicator struct {
	dualOperatorIndicator
}

func NewMinusIndicator(lhs, rhs techan.Indicator) techan.Indicator {
	return minusIndicator{dualOperatorIndicator{lhs: lhs, rhs: rhs}}
}

func (id minusIndicator) Calculate(index int) big.Decimal {
	return id.lhs.Calculate(index).Sub(id.rhs.Calculate(index))
}

type multIndicator struct {
	dualOperatorIndicator
}

func NewMultiplyIndicator(lhs, rhs techan.Indicator) techan.Indicator {
	return multIndicator{dualOperatorIndicator{lhs: lhs, rhs: rhs}}
}

func (id multIndicator) Calculate(index int) big.Decimal {
	return id.lhs.Calculate(index).Mul(id.rhs.Calculate(index))
}

type divIndicator struct {
	dualOperatorIndicator
}

func NewDivideIndicator(lhs, rhs techan.Indicator) techan.Indicator {
	return divIndicator{dualOperatorIndicator{lhs: lhs, rhs: rhs}}
}

func (id divIndicator) Calculate(index int) big.Decimal {
	return id.lhs.Calculate(index).Div(id.rhs.Calculate(index))
}
