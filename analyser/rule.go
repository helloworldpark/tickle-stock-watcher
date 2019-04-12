package analyser

import "github.com/sdcoffey/techan"
import "github.com/sdcoffey/big"

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
