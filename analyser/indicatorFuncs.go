package analyser

import (
	"fmt"

	"github.com/sdcoffey/techan"
)

func makeMACD(isHist bool) func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
	if isHist {
		return func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
			if len(a) != 3 {
				return nil, newError(fmt.Sprintf("[MACD] Not enough parameters: got %d, 3(MACD+Histogram)", len(a)))
			}
			shortWindow := int(a[0].(float64))
			longWindow := int(a[1].(float64))
			signalWindow := int(a[2].(float64))
			return newMACDHist(series, shortWindow, longWindow, signalWindow), nil
		}
	}
	return func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) != 2 {
			return nil, newError(fmt.Sprintf("[MACD] Not enough parameters: got %d, need 2(MACD)", len(a)))
		}
		shortWindow := int(a[0].(float64))
		longWindow := int(a[1].(float64))
		return newMACD(series, shortWindow, longWindow), nil
	}
}

func makeRSI() func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
	return func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) != 1 {
			return nil, newError(fmt.Sprintf("[rsi] Not enough parameters: got %d, need 1", len(a)))
		}
		timeframe := int(a[0].(float64))
		if timeframe < 1 {
			return nil, newError(fmt.Sprintf("[rsi] Lag should be longer than 0, not %d", timeframe))
		}
		return newRSI(series, timeframe), nil
	}
}

func makeClosePrice() func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
	return func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) != 0 {
			return nil, newError(fmt.Sprintf("[ClosePrice] Too many parameters: got %d, need 0", len(a)))
		}
		return techan.NewClosePriceIndicator(series), nil
	}
}

func makeIncrease() func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
	return func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) != 2 {
			return nil, newError(fmt.Sprintf("[Increase] Number of parameters incorrect: got %d, need 2", len(a)))
		}
		indicator := a[0].(techan.Indicator)
		lag := int(a[1].(float64))
		if lag < 1 {
			return nil, newError(fmt.Sprintf("[Increase] Lag should be longer than 0, not %d", lag))
		}
		return newIncreaseIndicator(indicator, lag), nil
	}
}

func makeExtrema() func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
	return func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) != 3 {
			return nil, newError(fmt.Sprintf("[LocalExtrema] Number of parameters incorrect: got %d, need 3", len(a)))
		}
		indicator := a[0].(techan.Indicator)
		lag := int(a[1].(float64))
		if lag < 1 {
			return nil, newError(fmt.Sprintf("[LocalExtrema] Lag should be longer than 0, not %d", lag))
		}
		samples := int(a[2].(float64))
		if samples < 4 {
			return nil, newError(fmt.Sprintf("[LocalExtrema] Samples should be more than 4, not %d", lag))
		}
		return newLocalExtremaIndicator(indicator, lag, samples), nil
	}
}

func makeMoneyFlowIndex() func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
	return func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) != 1 {
			return nil, newError(fmt.Sprintf("[MoneyFlowIndex] Not enough parameters: got %d, need 1", len(a)))
		}
		timeframe := int(a[0].(float64))
		if timeframe < 1 {
			return nil, newError(fmt.Sprintf("[MoneyFlowIndex] Lag should be longer than 0, not %d", timeframe))
		}
		return newMoneyFlowIndex(series, timeframe), nil
	}
}

func makeIsZero() func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
	return func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) != 3 {
			return nil, newError(fmt.Sprintf("[Zero] Number of parameters incorrect: got %d, need 3", len(a)))
		}
		indicator := a[0].(techan.Indicator)
		lag := int(a[1].(float64))
		if lag < 1 {
			return nil, newError(fmt.Sprintf("[Zero] Lag should be longer than 0, not %d", lag))
		}
		samples := int(a[2].(float64))
		if samples < 4 {
			return nil, newError(fmt.Sprintf("[Zero] Samples should be more than 4, not %d", lag))
		}
		return newLocalZeroIndicator(indicator, lag, samples), nil
	}
}
