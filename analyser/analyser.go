package analyser

import (
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/helloworldpark/govaluate"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

type token = govaluate.ExpressionToken
type expression = govaluate.EvaluableExpression
type indicatorGen = func(*techan.TimeSeries, ...interface{}) (techan.Indicator, error)

type userStockPosition struct {
	userid   int64
	stockid  string
	position int
}

type Analyser struct {
	techFuncMap     map[string]indicatorGen // Function Name: Function Generator
	userStrategy    map[userStockPosition]*expression
	timeSeriesCache map[string]*techan.TimeSeries // StockID: Time Series
}

type AnalyserError struct {
	msg string
}

func (this AnalyserError) Error() string {
	return this.msg
}

func NewAnalyser() *Analyser {
	newAnalyser := Analyser{}
	newAnalyser.techFuncMap = make(map[string]indicatorGen)
	newAnalyser.userStrategy = make(map[userStockPosition]*expression)
	newAnalyser.timeSeriesCache = make(map[string]*techan.TimeSeries)
	return &newAnalyser
}

func NewTestAnalyser() *Analyser {
	analyser := NewAnalyser()
	analyser.RegisterStock("123456")
	series := analyser.timeSeriesCache["123456"]
	for i := 0; i < 100; i++ {
		start := time.Date(0, 0, i, 0, 0, 0, 0, time.UTC)
		candle := techan.NewCandle(techan.NewTimePeriod(start, time.Hour*6))
		candle.ClosePrice = big.NewDecimal(math.Sin(float64(i)))
		series.AddCandle(candle)
	}
	return analyser
}

func (this *Analyser) CacheFunctions() {
	// MACD
	funcMacd := func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) < 2 {
			return nil, AnalyserError{msg: fmt.Sprintf("Not enough parameters: got %d, need more or equal to 2", len(a))}
		}
		shortWindow := int(a[0].(float64))
		longWindow := int(a[1].(float64))
		if len(a) == 2 {
			return newMACD(series, shortWindow, longWindow), nil
		} else if len(a) == 3 {
			signalWindow := int(a[2].(float64))
			return newMACDHist(series, shortWindow, longWindow, signalWindow), nil
		}
		return nil, AnalyserError{msg: fmt.Sprintf("Too much parameters: got %d, need less or equal to 3", len(a))}
	}
	this.techFuncMap["macd"] = funcMacd
	// RSI
}

func newMACD(series *techan.TimeSeries, shortWindow, longWindow int) techan.Indicator {
	return techan.NewMACDIndicator(techan.NewClosePriceIndicator(series), shortWindow, longWindow)
}

func newMACDHist(series *techan.TimeSeries, shortWindow, longWindow, signalWindow int) techan.Indicator {
	macd := newMACD(series, shortWindow, longWindow)
	return techan.NewMACDHistogramIndicator(macd, signalWindow)
}

type quad struct {
	name  string
	body  techan.Indicator
	start int
	end   int
}

func (this *Analyser) ParseAndCacheStrategy(userid int64, stockid string, position int, strategy string) (bool, error) {
	// First, parse tokens
	tmpTokens, err := govaluate.ParseTokens(strategy, nil)
	if err != nil {
		return false, err
	}

	// Search for token to switch to pre-cached function
	isFuncFound := false
	funcIdxStart := -1
	funcName := ""
	var funcBody indicatorGen
	var funcParam []interface{}
	tokenToReplace := make([]quad, 0)
	for i, t := range tmpTokens {
		// Find function
		// If found, check if we have
		// If we have, start collecting params
		// If found ')', append a pair of (name, function, startIdx, endIdx)
		if t.Kind == govaluate.VARIABLE {
			funcName = t.Value.(string)
			v, ok := this.techFuncMap[funcName]
			if !ok {
				return false, AnalyserError{msg: fmt.Sprintf("Unsupported function used: %s", funcName)}
			}
			isFuncFound = ok
			funcIdxStart = i
			funcBody = v
			funcParam = make([]interface{}, 0)
		} else if isFuncFound && t.Kind == govaluate.NUMERIC {
			funcParam = append(funcParam, t.Value.(float64))
		} else if isFuncFound && t.Kind == govaluate.CLAUSE_CLOSE {
			generatedIndicator, err := funcBody(this.timeSeriesCache[stockid], funcParam...)
			if err != nil {
				return false, err
			}

			tokenToReplace = append(tokenToReplace, quad{
				name:  funcName,
				body:  generatedIndicator,
				start: funcIdxStart,
				end:   i,
			})
			isFuncFound = false
			funcName = ""
			funcBody = nil
			funcParam = nil
		}
	}

	// Switch found ones
	var newTokens []token
	if len(tokenToReplace) > 0 {
		shouldReplace := func(mQuad quad, idx int) bool {
			return mQuad.start == idx
		}
		shouldSkip := func(mQuad quad, idx int) bool {
			return mQuad.start < idx && idx <= mQuad.end
		}
		quadToToken := func(mQuad quad) []token {
			ret := make([]token, 0)
			var expFunc govaluate.ExpressionFunction
			expFunc = func(a ...interface{}) (interface{}, error) {
				if len(a) != 1 {
					logger.Panic("[Analyser] Too may parameter for techan function: %v", a)
				}
				idx, ok := a[0].(float64)
				if !ok {
					logger.Panic("[Analyser] Something weird value has come into the techan function as a parameter: %v", a)
				}
				return mQuad.body.Calculate(int(idx)), nil
			}
			ret = append(ret, token{
				Kind:  govaluate.FUNCTION,
				Value: expFunc,
			})
			ret = append(ret, token{
				Kind:  govaluate.CLAUSE,
				Value: '(',
			})
			ret = append(ret, token{
				Kind:  govaluate.VARIABLE,
				Value: "x",
			})
			ret = append(ret, token{
				Kind:  govaluate.CLAUSE_CLOSE,
				Value: ')',
			})
			return ret
		}

		newTokens = make([]token, 0)
		var lastQuad *quad
		for i, t := range tmpTokens {
			if len(tokenToReplace) == 0 {
				if lastQuad == nil {
					newTokens = append(newTokens, t)
				} else {
					if shouldSkip(*lastQuad, i) {
						continue
					}
					newTokens = append(newTokens, t)
				}
			} else {
				lastQuad = &tokenToReplace[0]
				if shouldReplace(*lastQuad, i) {
					replacements := quadToToken(*lastQuad)
					for _, r := range replacements {
						newTokens = append(newTokens, r)
					}
					tokenToReplace = tokenToReplace[1:]
				} else if shouldSkip(*lastQuad, i) {
					continue
				} else {
					newTokens = append(newTokens, t)
				}
			}
		}
	} else {
		newTokens = tmpTokens
	}

	// Cache into map
	userKey := userStockPosition{
		userid:   userid,
		stockid:  stockid,
		position: position,
	}
	parsedExpression, err := govaluate.NewEvaluableExpressionFromTokens(newTokens)
	if err != nil {
		return false, err
	}
	this.userStrategy[userKey] = parsedExpression
	return true, nil
}

func (this *Analyser) PrintAllStrategy() {
	for k, v := range this.userStrategy {
		fmt.Printf("Key: %v, Value: %v", k, v.Vars())
	}
}

func Test() {
	aa, _ := govaluate.NewEvaluableExpression("true == false")
	tokens := aa.Tokens()
	v, err := aa.Evaluate(nil)
	fmt.Println(v, err)
	fmt.Println(tokens)

	function := map[string]govaluate.ExpressionFunction{
		"identity": func(a ...interface{}) (interface{}, error) {
			if len(a) == 2 {
				fmt.Printf("A[1] = %v\n", a[1])
			}
			return a[0], nil
		},
	}
	expString := "identity(x)"
	identityExpression, err := govaluate.NewEvaluableExpressionWithFunctions(expString, function)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(identityExpression.String())
	fmt.Println(identityExpression.Tokens())

	testExpression := "identity(1.0, 2.0) == 1.0"
	bb, err := govaluate.ParseTokens(testExpression, nil)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("----")
	for _, v := range bb {
		fmt.Printf("Kind: %v, Value: %v, ValueType: %v\n",
			v.Kind, v.Value, reflect.TypeOf(v.Value),
		)
	}
	fmt.Println("---------")

	bb[0] = identityExpression.Tokens()[0]
	newExpression, err := govaluate.NewEvaluableExpressionFromTokens(bb)
	result, err := newExpression.Evaluate(nil)
	fmt.Println(result)
	fmt.Println(err)
}

// Analyser의 상태 관리 관련한 함수들

func (this *Analyser) RegisterStock(stockid string) {
	_, ok := this.timeSeriesCache[stockid]
	if !ok {
		this.timeSeriesCache[stockid] = techan.NewTimeSeries()
	}
}

func (this *Analyser) UnregisterStock(stockid string) {
	delete(this.timeSeriesCache, stockid)
}
