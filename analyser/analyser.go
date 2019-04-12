package analyser

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/helloworldpark/govaluate"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

type token = govaluate.ExpressionToken
type expression = govaluate.EvaluableExpression
type indicatorGen = func(*techan.TimeSeries, ...interface{}) (techan.Indicator, error)
type ruleGen = func(...interface{}) (techan.Rule, error)

type userStockSide struct {
	userid    int64
	stockid   string
	orderside techan.OrderSide
}

type Analyser struct {
	indicatorMap    map[string]indicatorGen // Function Name: Indicator Generator Function
	ruleMap         map[string]ruleGen      // Function Name: Rule Generator Function
	userStrategy    map[userStockSide]Event
	timeSeriesCache map[string]*techan.TimeSeries // StockID: Time Series
}

var opPrecedence = map[string]int{
	"*": 1, "/": 1, "**": 1,
	"+": 2, "-": 2,
	"<": 3, "<=": 3, ">": 3, ">=": 3, "==": 3,
	"(": 4, ")": 4,
	"&&": 5, "||": 5,
}

type AnalyserError struct {
	msg string
}

func (this AnalyserError) Error() string {
	return "[Analyser] " + this.msg
}

func NewAnalyser() *Analyser {
	newAnalyser := Analyser{}
	newAnalyser.indicatorMap = make(map[string]indicatorGen)
	newAnalyser.userStrategy = make(map[userStockSide]Event)
	newAnalyser.timeSeriesCache = make(map[string]*techan.TimeSeries)
	newAnalyser.ruleMap = make(map[string]ruleGen)
	newAnalyser.cacheFunctions()
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

func (this *Analyser) cacheFunctions() {
	this.cacheIndicators()
	this.cacheRules()
}

func (this *Analyser) cacheIndicators() {
	// +-*/
	modifierAppender := func(operator string, ctor func(lhs, rhs techan.Indicator) techan.Indicator) {
		f := func(series *techan.TimeSeries, args ...interface{}) (techan.Indicator, error) {
			if len(args) != 2 {
				return nil, AnalyserError{msg: fmt.Sprintf("Not enough parameters: got %d, need more or equal to 2", len(args))}
			}
			lhs, ok := args[0].(techan.Indicator)
			if !ok {
				return nil, AnalyserError{msg: fmt.Sprintf("First argument must be of type techan.Indicator, you are %v", args[0])}
			}
			rhs, ok := args[1].(techan.Indicator)
			if !ok {
				return nil, AnalyserError{msg: fmt.Sprintf("Second argument must be of type techan.Indicator, you are %v", args[1])}
			}
			return ctor(lhs, rhs), nil
		}
		this.indicatorMap[operator] = f
	}
	modifierAppender("+", NewPlusIndicator)
	modifierAppender("-", NewMinusIndicator)
	modifierAppender("*", NewMultiplyIndicator)
	modifierAppender("/", NewDivideIndicator)

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
	this.indicatorMap["macd"] = funcMacd

	// RSI
	funcRsi := func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) != 1 {
			return nil, AnalyserError{msg: fmt.Sprintf("Not enough parameters: got %d, need more or equal to 1", len(a))}
		}
		timeframe := int(a[0].(float64))
		return newRSI(series, timeframe), nil
	}
	this.indicatorMap["rsi"] = funcRsi

	// Close Price
	funcClose := func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		return techan.NewClosePriceIndicator(series), nil
	}
	this.indicatorMap["close"] = funcClose
	this.indicatorMap["price"] = funcClose
	this.indicatorMap["closeprice"] = funcClose
}

func (this *Analyser) cacheRules() {
	appendRuleComparer := func(op string, ctor func(lhs, rhs techan.Rule) techan.Rule) {
		f := func(args ...interface{}) (techan.Rule, error) {
			if len(args) != 2 {
				return nil, AnalyserError{msg: fmt.Sprintf("Arguments for rule '%s' must be 2, you are %d", op, len(args))}
			}
			r1, ok := args[0].(techan.Rule)
			if !ok {
				return nil, AnalyserError{msg: fmt.Sprintf("First argument must be of type techan.Rule, you are %v", args[0])}
			}
			r2, ok := args[1].(techan.Rule)
			if !ok {
				return nil, AnalyserError{msg: fmt.Sprintf("Second argument must be of type techan.Rule, you are %v", args[1])}
			}
			return ctor(r1, r2), nil
		}
		this.ruleMap[op] = f
	}
	appendRuleComparer("&&", techan.And)
	appendRuleComparer("||", techan.Or)

	appendIndicatorComparer := func(op string, ctor func(lhs, rhs techan.Indicator) techan.Rule) {
		f := func(args ...interface{}) (techan.Rule, error) {
			if len(args) != 2 {
				return nil, AnalyserError{msg: fmt.Sprintf("Arguments for rule '%s' must be 2, you are %d", op, len(args))}
			}
			r1, ok := args[0].(techan.Indicator)
			if !ok {
				return nil, AnalyserError{msg: fmt.Sprintf("First argument must be of type techan.Rule, you are %v", args[0])}
			}
			r2, ok := args[1].(techan.Indicator)
			if !ok {
				return nil, AnalyserError{msg: fmt.Sprintf("Second argument must be of type techan.Rule, you are %v", args[1])}
			}
			return ctor(r1, r2), nil
		}
		this.ruleMap[op] = f
	}
	appendIndicatorComparer(">=", techan.NewCrossUpIndicatorRule)
	appendIndicatorComparer(">", techan.NewCrossUpIndicatorRule)
	appendIndicatorComparer("<=", techan.NewCrossDownIndicatorRule)
	appendIndicatorComparer("<", techan.NewCrossDownIndicatorRule)
	appendIndicatorComparer("==", NewCrossEqualIndicatorRule)
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

type quad struct {
	name  string
	body  techan.Indicator
	start int
	end   int
}

func (this *Analyser) ParseAndCacheStrategy(userid int64, stockid string, orderSide techan.OrderSide, strategyStatement string) (bool, error) {
	// First, parse tokens
	tmpTokens, err := this.parseTokens(strategyStatement)
	if err != nil {
		return false, err
	}

	newTokens, err := this.searchAndReplaceToFunctionTokens(tmpTokens, stockid)
	if err != nil {
		return false, err
	}

	postfixToken, err := this.reorderTokenByPostfix(newTokens)
	if err != nil {
		return false, err
	}

	// Create strategy using postfix tokens
	event, err := this.createEvent(postfixToken, orderSide)
	if err != nil {
		return false, err
	}

	// Cache into map
	userKey := userStockSide{
		userid:    userid,
		stockid:   stockid,
		orderside: orderSide,
	}
	this.userStrategy[userKey] = event
	return true, nil
}

func (this *Analyser) searchAndReplaceToFunctionTokens(tokens []token, stockid string) ([]token, error) {
	// Search for token to switch to pre-cached function
	isFuncFound := false
	funcIdxStart := -1
	funcName := ""
	var funcBody indicatorGen
	var funcParam []interface{}
	tokenToReplace := make([]quad, 0)
	for i, t := range tokens {
		// Find function
		// If found, check if we have
		// If we have, start collecting params
		// If found ')', append a pair of (name, function, startIdx, endIdx)
		if t.Kind == govaluate.VARIABLE {
			// Change function name to lower case
			(&t).Value = strings.ToLower(t.Value.(string))
			funcName = t.Value.(string)
			v, ok := this.indicatorMap[funcName]
			if !ok {
				return nil, AnalyserError{msg: fmt.Sprintf("Unsupported function used: %s", funcName)}
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
				return nil, err
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
		shouldPop := func(t token) bool {
			return t.Kind == govaluate.VARIABLE
		}
		replaceStart := func(mQuad quad, idx int) bool {
			return (mQuad).start == idx
		}
		replaceGoing := func(mQuad quad, idx int) bool {
			return (mQuad).start < idx && idx < (mQuad).end
		}
		replaceEnded := func(mQuad quad, idx int) bool {
			return (mQuad).end == idx
		}
		quadToToken := func(mQuad quad) []token {
			ret := make([]token, 0)
			expFunc := mQuad.body
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
		for i, t := range tokens {
			if len(tokenToReplace) == 0 {
				newTokens = append(newTokens, t)
			} else {
				if shouldPop(t) {
					lastQuad = &tokenToReplace[0]
				}
				if lastQuad == nil {
					newTokens = append(newTokens, t)
					continue
				}
				if replaceStart(*lastQuad, i) {
					replacements := quadToToken(*lastQuad)
					for _, r := range replacements {
						newTokens = append(newTokens, r)
					}
				} else if replaceGoing(*lastQuad, i) {
					continue
				} else if replaceEnded(*lastQuad, i) {
					lastQuad = nil
					tokenToReplace = tokenToReplace[1:]
				}
			}
		}
	} else {
		newTokens = tokens
	}
	return newTokens, nil
}

func (this *Analyser) parseTokens(statement string) ([]token, error) {
	return govaluate.ParseTokens(statement, nil)
}

func (this *Analyser) reorderTokenByPostfix(tokens []token) ([]token, error) {
	// Convert tokens into techan strategy
	// Tokens are reordered by postfix notation
	// operators:
	//             -: 0(Negation)
	//           * /: 1
	//           + -: 2
	//  < <= == >= >: 3
	//         && ||: 4
	//           ( ): 5

	postfixToken := make([]token, 0)
	operatorStack := make([]token, 0)
	functionStarted := false
	for _, t := range tokens {
		if functionStarted {
			if t.Kind == govaluate.CLAUSE_CLOSE {
				functionStarted = false
			}
			continue
		}
		if t.Kind == govaluate.FUNCTION {
			functionStarted = true
			postfixToken = append(postfixToken, t)
		} else if t.Kind == govaluate.NUMERIC {
			postfixToken = append(postfixToken, t)
		} else if t.Kind == govaluate.COMPARATOR ||
			t.Kind == govaluate.LOGICALOP ||
			t.Kind == govaluate.MODIFIER ||
			t.Kind == govaluate.CLAUSE ||
			t.Kind == govaluate.CLAUSE_CLOSE {

			op, ok := t.Value.(string)
			if !ok {
				clause, _ := t.Value.(int32)
				if clause == '(' {
					op = "("
				} else if clause == ')' {
					op = ")"
				} else {
					return nil, AnalyserError{msg: fmt.Sprintf("Invalid token: %v", t)}
				}
				(&t).Value = op
			}
			p := opPrecedence[op]
			for j := len(operatorStack) - 1; j >= 0; j-- {
				o := operatorStack[j]
				// 내 연산자 순위가 스택보다 높으면
				// 내가 들어간다
				// 아니면
				// 내가 스택보다 순위가 높을 때까지 애들을 다 postfixToken에 옮긴다
				if opPrecedence[o.Value.(string)] > p {
					break
				} else {
					if o.Kind != govaluate.CLAUSE && o.Kind != govaluate.CLAUSE_CLOSE {
						postfixToken = append(postfixToken, o)
					}
					operatorStack = operatorStack[:j]
				}
			}
			operatorStack = append(operatorStack, t)
		} else if t.Kind == govaluate.PREFIX {
			// 연산자 순위가 스택보다 무조건 높으므로
			// 내가 들어간다
			operatorStack = append(operatorStack, t)
		} else {
			return nil, AnalyserError{msg: fmt.Sprintf("Invalid token: %v", t)}
		}
	}
	for j := len(operatorStack) - 1; j >= 0; j-- {
		if operatorStack[j].Kind != govaluate.CLAUSE && operatorStack[j].Kind != govaluate.CLAUSE_CLOSE {
			postfixToken = append(postfixToken, operatorStack[j])
		}
		operatorStack = operatorStack[:j]
	}
	return postfixToken, nil
}

func (this *Analyser) createEvent(tokens []token, orderSide techan.OrderSide) (Event, error) {
	rule, err := this.createRule(tokens)
	if err != nil {
		return nil, err
	}
	event := NewEvent(orderSide, rule)
	return event, nil
}

func (this *Analyser) createRule(tokens []token) (techan.Rule, error) {
	indicators := make([]techan.Indicator, 0)
	rules := make([]techan.Rule, 0)
	for len(tokens) > 0 {
		t := tokens[0]
		tokens = tokens[1:]

		if t.Kind == govaluate.FUNCTION {
			indicators = append(indicators, t.Value.(techan.Indicator))
		} else if t.Kind == govaluate.NUMERIC {
			indicators = append(indicators, techan.NewConstantIndicator(t.Value.(float64)))
		} else if t.Kind == govaluate.PREFIX {
			v := indicators[len(indicators)-1]
			indicators = indicators[:(len(indicators) - 1)]
			indicators = append(indicators, NewNegateIndicator(v))
		} else if t.Kind == govaluate.COMPARATOR {
			rhs := indicators[len(indicators)-1]
			lhs := indicators[len(indicators)-2]
			indicators = indicators[:(len(indicators) - 2)]
			ruleMaker := this.ruleMap[t.Value.(string)]
			rule, err := ruleMaker(lhs, rhs)
			if err != nil {
				return nil, err
			}
			rules = append(rules, rule)
		} else if t.Kind == govaluate.LOGICALOP {
			rhs := rules[len(rules)-1]
			lhs := rules[len(rules)-2]
			rules = rules[:(len(rules) - 2)]
			ruleMaker := this.ruleMap[t.Value.(string)]
			rule, err := ruleMaker(lhs, rhs)
			if err != nil {
				return nil, err
			}
			rules = append(rules, rule)
		} else if t.Kind == govaluate.MODIFIER {
			rhs := indicators[len(indicators)-1]
			lhs := indicators[len(indicators)-2]
			indicators = indicators[:(len(indicators) - 2)]
			operated, err := this.indicatorMap[t.Value.(string)](nil, lhs, rhs)
			if err != nil {
				return nil, err
			}
			indicators = append(indicators, operated)
		}
	}

	if len(rules) != 1 {
		// Something wrong
		logger.Panic("[Analyser] Something is wrong: rule must be generated unique.")
	}

	return rules[0], nil
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
