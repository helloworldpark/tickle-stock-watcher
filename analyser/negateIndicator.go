package analyser

import "github.com/sdcoffey/techan"
import "github.com/sdcoffey/big"

type negateIndicator struct {
	indicator techan.Indicator
}

func newNegateIndicator(indicator techan.Indicator) techan.Indicator {
	return negateIndicator{indicator: indicator}
}

func (ni negateIndicator) Calculate(index int) big.Decimal {
	return ni.indicator.Calculate(index).Neg()
}
