package analyser

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/helloworldpark/govaluate"
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

type token = govaluate.ExpressionToken
type expression = govaluate.EvaluableExpression
type indicatorGen = func(*techan.TimeSeries, ...interface{}) (techan.Indicator, error)
type ruleGen = func(...interface{}) (techan.Rule, error)

type userSide struct {
	userid    int
	orderside techan.OrderSide
	repeat    bool
}

// Error is an error struct
type Error struct {
	msg string
}

func (err Error) Error() string {
	return "[Analyser] " + err.msg
}

func newError(msg string) Error {
	return Error{msg: msg}
}

// Operator Precedence
var opPrecedence = map[string]int{
	"*": 1, "/": 1, "**": 1,
	"+": 2, "-": 2,
	"<": 3, "<=": 3, ">": 3, ">=": 3, "==": 3,
	"(": 4, ")": 4,
	"&&": 5, "||": 5,
}

// Analyser is a struct for signalling to users by condition they have set.
type Analyser struct {
	indicatorMap map[string]indicatorGen // Function Name: Indicator Generator Function
	ruleMap      map[string]ruleGen      // Function Name: Rule Generator Function
	userStrategy map[userSide]EventTrigger
	timeSeries   *techan.TimeSeries
	counter      *commons.Ref
	stockID      string
}

// newAnalyser creates and returns a pointer of a new prepared Analyser struct
func newAnalyser(stockID string) *Analyser {
	newAnalyser := Analyser{}
	newAnalyser.indicatorMap = make(map[string]indicatorGen)
	newAnalyser.userStrategy = make(map[userSide]EventTrigger)
	newAnalyser.timeSeries = techan.NewTimeSeries()
	newAnalyser.ruleMap = make(map[string]ruleGen)
	newAnalyser.counter = &commons.Ref{}
	newAnalyser.stockID = stockID
	newAnalyser.cacheFunctions()
	return &newAnalyser
}

func newTestAnalyser() *Analyser {
	analyser := newAnalyser("123456")
	for i := 0; i < 100; i++ {
		start := time.Date(0, 0, i, 0, 0, 0, 0, time.UTC)
		candle := techan.NewCandle(techan.NewTimePeriod(start, time.Hour*6))
		candle.ClosePrice = big.NewDecimal(math.Sin(float64(i)))
		analyser.timeSeries.AddCandle(candle)
	}
	return analyser
}

// Retain implementation of ReferenceCounting
func (a *Analyser) Retain() {
	a.counter.Retain()
}

// Release implementation of ReferenceCounting
func (a *Analyser) Release() {
	a.counter.Release()
}

// Count implementation of ReferenceCounting
func (a *Analyser) Count() int {
	return a.counter.Count()
}

func (a *Analyser) cacheFunctions() {
	a.cacheIndicators()
	a.cacheRules()
}

func (a *Analyser) cacheIndicators() {
	// +-*/
	modifierAppender := func(operator string, ctor func(lhs, rhs techan.Indicator) techan.Indicator) {
		f := func(series *techan.TimeSeries, args ...interface{}) (techan.Indicator, error) {
			if len(args) != 2 {
				return nil, newError(fmt.Sprintf("Not enough parameters: got %d, need more or equal to 2", len(args)))
			}
			lhs, ok := args[0].(techan.Indicator)
			if !ok {
				return nil, newError(fmt.Sprintf("First argument must be of type techan.Indicator, you are %v", args[0]))
			}
			rhs, ok := args[1].(techan.Indicator)
			if !ok {
				return nil, newError(fmt.Sprintf("Second argument must be of type techan.Indicator, you are %v", args[1]))
			}
			return ctor(lhs, rhs), nil
		}
		a.indicatorMap[operator] = f
	}
	modifierAppender("+", newPlusIndicator)
	modifierAppender("-", newMinusIndicator)
	modifierAppender("*", newMultiplyIndicator)
	modifierAppender("/", newDivideIndicator)

	// MACD
	funcMacd := func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) < 2 {
			return nil, newError(fmt.Sprintf("Not enough parameters: got %d, need more or equal to 2", len(a)))
		}
		shortWindow := int(a[0].(float64))
		longWindow := int(a[1].(float64))
		if len(a) == 2 {
			return newMACD(series, shortWindow, longWindow), nil
		} else if len(a) == 3 {
			signalWindow := int(a[2].(float64))
			return newMACDHist(series, shortWindow, longWindow, signalWindow), nil
		}
		return nil, newError(fmt.Sprintf("Too much parameters: got %d, need less or equal to 3", len(a)))
	}
	a.indicatorMap["macd"] = funcMacd

	// RSI
	funcRsi := func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) != 1 {
			return nil, newError(fmt.Sprintf("Not enough parameters: got %d, need more or equal to 1", len(a)))
		}
		timeframe := int(a[0].(float64))
		return newRSI(series, timeframe), nil
	}
	a.indicatorMap["rsi"] = funcRsi

	// Close Price
	funcClose := func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		return techan.NewClosePriceIndicator(series), nil
	}
	a.indicatorMap["close"] = funcClose
	a.indicatorMap["price"] = funcClose
	a.indicatorMap["closeprice"] = funcClose
}

func (a *Analyser) cacheRules() {
	appendRuleComparer := func(op string, ctor func(lhs, rhs techan.Rule) techan.Rule) {
		f := func(args ...interface{}) (techan.Rule, error) {
			if len(args) != 2 {
				return nil, newError(fmt.Sprintf("Arguments for rule '%s' must be 2, you are %d", op, len(args)))
			}
			r1, ok := args[0].(techan.Rule)
			if !ok {
				return nil, newError(fmt.Sprintf("First argument must be of type techan.Rule, you are %v", args[0]))
			}
			r2, ok := args[1].(techan.Rule)
			if !ok {
				return nil, newError(fmt.Sprintf("Second argument must be of type techan.Rule, you are %v", args[1]))
			}
			return ctor(r1, r2), nil
		}
		a.ruleMap[op] = f
	}
	appendRuleComparer("&&", techan.And)
	appendRuleComparer("||", techan.Or)

	appendIndicatorComparer := func(op string, ctor func(lhs, rhs techan.Indicator) techan.Rule) {
		f := func(args ...interface{}) (techan.Rule, error) {
			if len(args) != 2 {
				return nil, newError(fmt.Sprintf("Arguments for rule '%s' must be 2, you are %d", op, len(args)))
			}
			r1, ok := args[0].(techan.Indicator)
			if !ok {
				return nil, newError(fmt.Sprintf("First argument must be of type techan.Rule, you are %v", args[0]))
			}
			r2, ok := args[1].(techan.Indicator)
			if !ok {
				return nil, newError(fmt.Sprintf("Second argument must be of type techan.Rule, you are %v", args[1]))
			}
			return ctor(r1, r2), nil
		}
		a.ruleMap[op] = f
	}
	appendIndicatorComparer(">=", techan.NewCrossUpIndicatorRule)
	appendIndicatorComparer(">", techan.NewCrossUpIndicatorRule)
	appendIndicatorComparer("<=", techan.NewCrossDownIndicatorRule)
	appendIndicatorComparer("<", techan.NewCrossDownIndicatorRule)
	appendIndicatorComparer("==", NewCrossEqualIndicatorRule)
}

type quad struct {
	name  string
	body  techan.Indicator
	start int
	end   int
}

func (a *Analyser) parseAndCacheStrategy(strategy structs.UserStock, callback EventCallback) (bool, error) {
	// First, parse tokens
	tmpTokens, err := a.parseTokens(strategy.Strategy)
	if err != nil {
		return false, err
	}

	newTokens, err := a.searchAndReplaceToFunctionTokens(tmpTokens)
	if err != nil {
		return false, err
	}

	postfixToken, err := a.reorderTokenByPostfix(newTokens)
	if err != nil {
		return false, err
	}

	// Create strategy using postfix tokens
	orderSide := techan.OrderSide(strategy.OrderSide)
	event, err := a.createEvent(postfixToken, orderSide, callback)
	if err != nil {
		return false, err
	}

	// Cache into map
	userKey := userSide{
		userid:    strategy.UserID,
		orderside: orderSide,
		repeat:    strategy.Repeat,
	}
	a.userStrategy[userKey] = event
	return true, nil
}

func (a *Analyser) searchAndReplaceToFunctionTokens(tokens []token) ([]token, error) {
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
			v, ok := a.indicatorMap[funcName]
			if !ok {
				return nil, newError(fmt.Sprintf("Unsupported function used: %s", funcName))
			}
			isFuncFound = ok
			funcIdxStart = i
			funcBody = v
			funcParam = make([]interface{}, 0)
		} else if isFuncFound && t.Kind == govaluate.NUMERIC {
			funcParam = append(funcParam, t.Value.(float64))
		} else if isFuncFound && t.Kind == govaluate.CLAUSE_CLOSE {
			generatedIndicator, err := funcBody(a.timeSeries, funcParam...)
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

func (a *Analyser) parseTokens(statement string) ([]token, error) {
	return govaluate.ParseTokens(statement, nil)
}

func (a *Analyser) reorderTokenByPostfix(tokens []token) ([]token, error) {
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
					return nil, newError(fmt.Sprintf("Invalid token: %v", t))
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
			return nil, newError(fmt.Sprintf("Invalid token: %v", t))
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

func (a *Analyser) createEvent(tokens []token, orderSide techan.OrderSide, callback EventCallback) (EventTrigger, error) {
	rule, err := a.createRule(tokens)
	if err != nil {
		return nil, err
	}
	eventTrigger := NewEventTrigger(orderSide, rule, callback)
	return eventTrigger, nil
}

func (a *Analyser) createRule(tokens []token) (techan.Rule, error) {
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
			indicators = append(indicators, newNegateIndicator(v))
		} else if t.Kind == govaluate.COMPARATOR {
			rhs := indicators[len(indicators)-1]
			lhs := indicators[len(indicators)-2]
			indicators = indicators[:(len(indicators) - 2)]
			ruleMaker := a.ruleMap[t.Value.(string)]
			rule, err := ruleMaker(lhs, rhs)
			if err != nil {
				return nil, err
			}
			rules = append(rules, rule)
		} else if t.Kind == govaluate.LOGICALOP {
			rhs := rules[len(rules)-1]
			lhs := rules[len(rules)-2]
			rules = rules[:(len(rules) - 2)]
			ruleMaker := a.ruleMap[t.Value.(string)]
			rule, err := ruleMaker(lhs, rhs)
			if err != nil {
				return nil, err
			}
			rules = append(rules, rule)
		} else if t.Kind == govaluate.MODIFIER {
			rhs := indicators[len(indicators)-1]
			lhs := indicators[len(indicators)-2]
			indicators = indicators[:(len(indicators) - 2)]
			operated, err := a.indicatorMap[t.Value.(string)](nil, lhs, rhs)
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

func (a *Analyser) appendStrategy(userStrategy structs.UserStock, callback EventCallback) (bool, error) {
	return a.parseAndCacheStrategy(userStrategy, callback)
}

func (a *Analyser) deleteStrategy(userid int, orderside techan.OrderSide) {
	key := userSide{userid: userid, orderside: orderside}
	delete(a.userStrategy, key)
}

func (a *Analyser) prepareWatching() {
	newCandle := techan.NewCandle(techan.NewTimePeriod(commons.Today(), time.Hour*24))
	a.timeSeries.AddCandle(newCandle)
	for len(a.timeSeries.Candles) > 100 {
		a.timeSeries.Candles = a.timeSeries.Candles[1:]
	}
}

func (a *Analyser) watchStockPrice(stockPrice structs.StockPrice) {
	lastCandle := a.timeSeries.LastCandle()
	lastCandle.ClosePrice = big.NewDecimal(float64(stockPrice.Close))
}

func (a *Analyser) appendPastStockPrice(stockPrice structs.StockPrice) {
	lastTimestamp := a.timeSeries.LastCandle().Period.Start.Unix()
	if lastTimestamp > stockPrice.Timestamp {
		return
	}
	var candle *techan.Candle
	if lastTimestamp == stockPrice.Timestamp {
		candle = a.timeSeries.LastCandle()
	} else {
		start := time.Unix(stockPrice.Timestamp, 0)
		candle = techan.NewCandle(techan.NewTimePeriod(start, time.Hour*24))
	}
	candle.OpenPrice = big.NewDecimal(float64(stockPrice.Open))
	candle.ClosePrice = big.NewDecimal(float64(stockPrice.Close))
	candle.MaxPrice = big.NewDecimal(float64(stockPrice.High))
	candle.MinPrice = big.NewDecimal(float64(stockPrice.Low))
	candle.Volume = big.NewDecimal(float64(stockPrice.Volume))
	if lastTimestamp < stockPrice.Timestamp {
		a.timeSeries.AddCandle(candle)
	}
}

func (a *Analyser) calculateStrategies() {
	triggered := make(map[userSide]EventTrigger)
	for k, v := range a.userStrategy {
		if v.IsTriggered(a.timeSeries.LastIndex(), nil) {
			triggered[k] = v
		}
	}
	closePrice := a.timeSeries.LastCandle().ClosePrice.Float()
	for k, v := range triggered {
		v.OnEvent(closePrice, a.stockID, int(k.orderside))
	}
}
