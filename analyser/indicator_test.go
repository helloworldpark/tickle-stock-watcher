package analyser

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

func TestLocalMinMax(t *testing.T) {
	series := techan.NewTimeSeries()
	f := func(x float64) float64 {
		return math.Sin(x)
	}
	for i := 0; i < 1000; i++ {
		t := float64(i) / 10.0
		y := f(t)
		tt := time.Date(0, 0, 0, 0, 0, 0, i, time.UTC)
		tp := techan.NewTimePeriod(tt, 1)
		candle := techan.NewCandle(tp)
		candle.ClosePrice = big.NewDecimal(y)
		series.AddCandle(candle)
	}
	closeIndicator := techan.NewClosePriceIndicator(series)
	localIndicator := newLocalMinMaxIndicator(closeIndicator, 2, 4)
	for i := 0; i < 1000; i++ {
		price := closeIndicator.Calculate(i).Float()
		local := int(localIndicator.Calculate(i).Float())
		fmt.Printf("Price[%d]: %f Local: %d\n", i, price, local)
	}
}
