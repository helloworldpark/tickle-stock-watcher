package analyser

import (
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

type equalityRule struct {
	lhs techan.Indicator
	rhs techan.Indicator
}

// NewCrossEqualIndicatorRule returns a new Rule checking if two indicators' calculation are equal
func NewCrossEqualIndicatorRule(lhs, rhs techan.Indicator) techan.Rule {
	return equalityRule{lhs: lhs, rhs: rhs}
}

func (e equalityRule) IsSatisfied(index int, record *techan.TradingRecord) bool {
	return e.lhs.Calculate(index).Sub(e.rhs.Calculate(index)).Abs().LT(big.NewDecimal(1e-9))
}

type compareRule struct {
	lhs techan.Indicator
	rhs techan.Indicator
	cmp big.Decimal
	eq  big.Decimal
}

// NewCrossLTIndicatorRule returns a new Rule checking if two indicators' calculation are equal
func NewCrossLTIndicatorRule(lhs, rhs techan.Indicator) techan.Rule {
	return compareRule{lhs: lhs, rhs: rhs, cmp: big.NewDecimal(-1), eq: big.NewDecimal(1.e-9)}
}

// NewCrossGTIndicatorRule returns a new Rule checking if two indicators' calculation are equal
func NewCrossGTIndicatorRule(lhs, rhs techan.Indicator) techan.Rule {
	return compareRule{lhs: lhs, rhs: rhs, cmp: big.NewDecimal(1), eq: big.NewDecimal(1.e-9)}
}

// NewCrossLTEIndicatorRule returns a new Rule checking if two indicators' calculation are equal
func NewCrossLTEIndicatorRule(lhs, rhs techan.Indicator) techan.Rule {
	return compareRule{lhs: lhs, rhs: rhs, cmp: big.NewDecimal(-1), eq: big.NewDecimal(0)}
}

// NewCrossGTEIndicatorRule returns a new Rule checking if two indicators' calculation are equal
func NewCrossGTEIndicatorRule(lhs, rhs techan.Indicator) techan.Rule {
	return compareRule{lhs: lhs, rhs: rhs, cmp: big.NewDecimal(1), eq: big.NewDecimal(0)}
}

func (e compareRule) IsSatisfied(index int, record *techan.TradingRecord) bool {
	return e.lhs.Calculate(index).Sub(e.rhs.Calculate(index)).Mul(e.cmp).GTE(big.NewDecimal(0).Add(e.eq))
}
