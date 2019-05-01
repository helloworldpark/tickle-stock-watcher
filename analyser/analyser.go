package analyser

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/helloworldpark/govaluate"
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

type token = govaluate.ExpressionToken
type expression = govaluate.EvaluableExpression
type indicatorGen = func(*techan.TimeSeries, ...interface{}) (techan.Indicator, error)
type ruleGen = func(...interface{}) (techan.Rule, error)
type uid = int64

type eventWrapper struct {
	repeat bool
	event  EventTrigger
}

const (
	// MaxCandles for analysers: only hold price of last MaxCandles days
	MaxCandles = 100
)

// Operator Precedence
var opPrecedence = map[string]int{
	"*": 6, "/": 6, "**": 6,
	"+": 5, "-": 5,
	"<": 4, "<=": 4, ">": 4, ">=": 4, "==": 4,
	"(": 3, ")": 3,
	"&&": 2, "||": 2,
}
var newError = commons.NewTaggedError("Analyser")

// Analyser is a struct for signalling to users by condition they have set.
type Analyser struct {
	indicatorMap map[string]indicatorGen // Function Name: Indicator Generator Function
	ruleMap      map[string]ruleGen      // Function Name: Rule Generator Function
	userStrategy map[uid]map[techan.OrderSide]eventWrapper
	timeSeries   *techan.TimeSeries
	counter      *commons.Ref
	stockID      string
	isWatching   bool
}

// newAnalyser creates and returns a pointer of a new prepared Analyser struct
func newAnalyser(stockID string) *Analyser {
	newAnalyser := Analyser{}
	newAnalyser.indicatorMap = make(map[string]indicatorGen)
	newAnalyser.userStrategy = make(map[uid]map[techan.OrderSide]eventWrapper)
	newAnalyser.timeSeries = techan.NewTimeSeries()
	newAnalyser.ruleMap = make(map[string]ruleGen)
	newAnalyser.counter = &commons.Ref{}
	newAnalyser.stockID = stockID
	newAnalyser.isWatching = false
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
			return nil, newError(fmt.Sprintf("Not enough parameters: got %d, need 1", len(a)))
		}
		timeframe := int(a[0].(float64))
		return newRSI(series, timeframe), nil
	}
	a.indicatorMap["rsi"] = funcRsi

	// Close Price
	funcClose := func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) != 0 {
			return nil, newError(fmt.Sprintf("Too many parameters: got %d, need 0", len(a)))
		}
		return techan.NewClosePriceIndicator(series), nil
	}
	a.indicatorMap["close"] = funcClose
	a.indicatorMap["price"] = funcClose
	a.indicatorMap["closeprice"] = funcClose

	// Increase
	funcIncrease := func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) != 2 {
			return nil, newError(fmt.Sprintf("Number of parameters incorrect: got %d, need 2", len(a)))
		}
		indicator := a[0].(techan.Indicator)
		lag := int(a[1].(float64))
		return newIncreaseIndicator(indicator, lag), nil
	}
	a.indicatorMap["increase"] = funcIncrease

	// Local Extrema
	funcExtrema := func(series *techan.TimeSeries, a ...interface{}) (techan.Indicator, error) {
		if len(a) != 3 {
			return nil, newError(fmt.Sprintf("Number of parameters incorrect: got %d, need 3", len(a)))
		}
		indicator := a[0].(techan.Indicator)
		lag := int(a[1].(float64))
		samples := int(a[2].(float64))
		return newLocalExtremaIndicator(indicator, lag, samples), nil
	}
	a.indicatorMap["extrema"] = funcExtrema
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
	appendIndicatorComparer("<=", NewCrossLTEIndicatorRule)
	appendIndicatorComparer("<", NewCrossLTIndicatorRule)
	appendIndicatorComparer(">=", NewCrossGTEIndicatorRule)
	appendIndicatorComparer(">", NewCrossGTIndicatorRule)
	appendIndicatorComparer("==", NewCrossEqualIndicatorRule)
}

func (a *Analyser) parseAndCacheStrategy(strategy structs.UserStock, callback EventCallback) (bool, error) {
	// First, parse tokens
	tmpTokens, err := a.parseTokens(strategy.Strategy)
	if err != nil {
		return false, err
	}

	newTokens, err := a.tidyTokens(tmpTokens)
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
	userStrategy := eventWrapper{repeat: strategy.Repeat, event: event}
	strategies, ok := a.userStrategy[strategy.UserID]
	if !ok {
		a.userStrategy[strategy.UserID] = make(map[techan.OrderSide]eventWrapper)
		strategies = a.userStrategy[strategy.UserID]
	}
	strategies[techan.OrderSide(strategy.OrderSide)] = userStrategy
	a.userStrategy[strategy.UserID] = strategies
	return true, nil
}

func (a *Analyser) parseTokens(statement string) ([]token, error) {
	return govaluate.ParseTokens(statement, nil)
}

func (a *Analyser) tidyTokens(tokens []token) ([]token, error) {
	for i := range tokens {
		t := &(tokens[i])
		if t.Kind == govaluate.VARIABLE {
			// Change function name to lower case
			t.Value = strings.ToLower(t.Value.(string))
			_, ok := a.indicatorMap[t.Value.(string)]
			if !ok {
				return nil, newError(fmt.Sprintf("Unsupported function used: %s", t.Value.(string)))
			}
		} else if t.Kind == govaluate.CLAUSE {
			t.Value = "("
		} else if t.Kind == govaluate.CLAUSE_CLOSE {
			t.Value = ")"
		}
	}
	return tokens, nil
}

type function struct {
	t    token
	argc int
}

func newFunction(t token, argc int) *function {
	f := function{t: t}
	switch t.Kind {
	case govaluate.NUMERIC, govaluate.CLAUSE, govaluate.CLAUSE_CLOSE:
		f.argc = 0
	case govaluate.PREFIX:
		f.argc = 1
	case govaluate.VARIABLE:
		f.argc = argc
	default:
		f.argc = 2
	}
	return &f
}

func (a *Analyser) reorderTokenByPostfix(tokens []token) ([]function, error) {
	// Convert tokens into techan strategy
	// Tokens are reordered by postfix notation
	// operators:
	//     functions: 8
	//             -: 7(Negation)
	//           * /: 6
	//           + -: 5
	//  < <= == >= >: 4
	//         && ||: 3
	//           ( ): 2

	postfixToken := make([]function, 0)
	operatorStack := make([]*function, 0)
	realFcnStack := make([]*function, 0)
	clauseIdxStack := make([]int, 0)
	for i := range tokens {
		t := tokens[i]
		switch t.Kind {
		case govaluate.NUMERIC:
			postfixToken = append(postfixToken, *newFunction(t, 0))
		case govaluate.COMPARATOR, govaluate.LOGICALOP, govaluate.VARIABLE, govaluate.PREFIX, govaluate.MODIFIER:
			p := precedenceOf(t)
			for j := len(operatorStack) - 1; j >= 0; j-- {
				o := operatorStack[j]
				// 내 연산자 순위가 스택보다 높으면(즉, 숫자가 크면)
				// 내가 들어간다
				// 아니면
				// 내가 스택보다 순위가 높을 때까지 애들을 다 postfixToken에 옮긴다
				op := precedenceOf(o.t)
				if p > op {
					break
				} else {
					postfixToken = append(postfixToken, *o)
					operatorStack = operatorStack[:j]
				}
			}
			operatorStack = append(operatorStack, newFunction(t, 0))
			if t.Kind == govaluate.VARIABLE {
				realFcnStack = append(realFcnStack, operatorStack[len(operatorStack)-1])
			}
		case govaluate.CLAUSE:
			operatorStack = append(operatorStack, newFunction(t, 0))
			// 함수의 인자의 갯수를 파악하기 위해
			// 스택을 사용하여 함수들의 인자를 순서대로 파악한다
			clauseIdxStack = append(clauseIdxStack, i)
		case govaluate.CLAUSE_CLOSE:
			for {
				o := operatorStack[len(operatorStack)-1]
				operatorStack = operatorStack[:len(operatorStack)-1]
				if o.t.Kind == govaluate.CLAUSE {
					break
				} else {
					postfixToken = append(postfixToken, *o)
				}
			}
			lastClauseIdx := clauseIdxStack[len(clauseIdxStack)-1]
			// 함수의 괄호였으므로, realFcnStack에서 함수포인터를 pop한다
			// 함수도 operator stack에서 pop하고 postfix stack으로 옮긴다
			if lastClauseIdx-1 >= 0 && tokens[lastClauseIdx-1].Kind == govaluate.VARIABLE {
				if realFcnStack[len(realFcnStack)-1].argc > 0 {
					realFcnStack[len(realFcnStack)-1].argc++
				}
				realFcnStack = realFcnStack[:len(realFcnStack)-1]
				o := operatorStack[len(operatorStack)-1]
				operatorStack = operatorStack[:len(operatorStack)-1]
				postfixToken = append(postfixToken, *o)
			}
			clauseIdxStack = clauseIdxStack[:len(clauseIdxStack)-1]
		case govaluate.SEPARATOR:
			if len(realFcnStack) > 0 {
				realFcnStack[len(realFcnStack)-1].argc++
			}
		default:
			return nil, newError(fmt.Sprintf("Invalid token: %v", t))
		}
	}
	for j := len(operatorStack) - 1; j >= 0; j-- {
		if operatorStack[j].t.Kind != govaluate.CLAUSE && operatorStack[j].t.Kind != govaluate.CLAUSE_CLOSE {
			postfixToken = append(postfixToken, *operatorStack[j])
		}
		operatorStack = operatorStack[:j]
	}
	return postfixToken, nil
}

func precedenceOf(t token) int {
	if t.Kind == govaluate.VARIABLE {
		return 8
	}
	if t.Kind == govaluate.PREFIX {
		return 7
	}
	return opPrecedence[t.Value.(string)]
}

func (a *Analyser) createEvent(tokens []function, orderSide techan.OrderSide, callback EventCallback) (EventTrigger, error) {
	rule, err := a.createRule(tokens)
	if err != nil {
		return nil, err
	}
	eventTrigger := NewEventTrigger(orderSide, rule, callback)
	return eventTrigger, nil
}

func (a *Analyser) createRule(fcns []function) (techan.Rule, error) {
	indicators := make([]interface{}, 0)
	rules := make([]techan.Rule, 0)
	for len(fcns) > 0 {
		f := fcns[0]
		fcns = fcns[1:]

		switch f.t.Kind {
		case govaluate.NUMERIC:
			// indicators = append(indicators, techan.NewConstantIndicator(t.Value.(float64)))
			indicators = append(indicators, f.t.Value.(float64))
		case govaluate.VARIABLE:
			// 함수를 구성한다
			// 인자를 슬라이스에 담고
			// indicator를 만든다
			args := indicators[len(indicators)-f.argc:]
			indicators = indicators[:len(indicators)-f.argc]
			gen, ok := a.indicatorMap[f.t.Value.(string)]
			if !ok {
				return nil, newError("Not implemented function")
			}
			indicator, err := gen(a.timeSeries, args...)
			if err != nil {
				return nil, err
			}
			indicators = append(indicators, indicator)
		case govaluate.PREFIX:
			v := indicators[len(indicators)-1]
			indicators = indicators[:(len(indicators) - 1)]
			indi, ok := v.(techan.Indicator)
			if ok {
				indicators = append(indicators, newNegateIndicator(indi))
			} else {
				indicators = append(indicators, newNegateIndicatorFromFloat(v.(float64)))
			}
		case govaluate.COMPARATOR:
			if len(indicators) < 2 {
				return nil, newError(fmt.Sprintf("Cannot compose a comparing rule with %d indicators", len(indicators)))
			}
			rhs := indicators[len(indicators)-1]
			lhs := indicators[len(indicators)-2]
			indicators = indicators[:(len(indicators) - 2)]
			ruleMaker := a.ruleMap[f.t.Value.(string)]

			rhsIndicator, ok := rhs.(techan.Indicator)
			if !ok {
				rhsIndicator = techan.NewConstantIndicator(rhs.(float64))
			}
			lhsIndicator, ok := lhs.(techan.Indicator)
			if !ok {
				lhsIndicator = techan.NewConstantIndicator(lhs.(float64))
			}
			rule, err := ruleMaker(lhsIndicator, rhsIndicator)
			if err != nil {
				return nil, err
			}
			rules = append(rules, rule)
		case govaluate.LOGICALOP:
			rhs := rules[len(rules)-1]
			lhs := rules[len(rules)-2]
			rules = rules[:(len(rules) - 2)]
			ruleMaker := a.ruleMap[f.t.Value.(string)]
			rule, err := ruleMaker(lhs, rhs)
			if err != nil {
				return nil, err
			}
			rules = append(rules, rule)
		case govaluate.MODIFIER:
			rhs := indicators[len(indicators)-1]
			lhs := indicators[len(indicators)-2]
			indicators = indicators[:(len(indicators) - 2)]

			rhsIndicator, ok := rhs.(techan.Indicator)
			if !ok {
				rhsIndicator = techan.NewConstantIndicator(rhs.(float64))
			}
			lhsIndicator, ok := lhs.(techan.Indicator)
			if !ok {
				lhsIndicator = techan.NewConstantIndicator(lhs.(float64))
			}
			operated, err := a.indicatorMap[f.t.Value.(string)](nil, lhsIndicator, rhsIndicator)
			if err != nil {
				return nil, err
			}
			indicators = append(indicators, operated)
		}
	}

	if len(rules) != 1 {
		// Something wrong
		return nil, newError(fmt.Sprintf("Rule must exist and be unique: %d rules generated", len(rules)))
	}

	return rules[0], nil
}

func (a *Analyser) appendStrategy(userStrategy structs.UserStock, callback EventCallback) (bool, error) {
	return a.parseAndCacheStrategy(userStrategy, callback)
}

func (a *Analyser) deleteStrategy(userid int64, orderside techan.OrderSide) {
	delete(a.userStrategy[userid], orderside)
	if len(a.userStrategy[userid]) == 0 {
		delete(a.userStrategy, userid)
	}
}

func (a *Analyser) prepareWatching() {
	newCandle := techan.NewCandle(techan.NewTimePeriod(commons.Today(), time.Hour*24))
	a.timeSeries.AddCandle(newCandle)
	if len(a.timeSeries.Candles) > MaxCandles {
		a.timeSeries.Candles = a.timeSeries.Candles[len(a.timeSeries.Candles)-MaxCandles:]
	}
}

func (a *Analyser) isWatchingPrice() bool {
	return a.isWatching
}

func (a *Analyser) watchStockPrice(stockPrice structs.StockPrice) {
	a.isWatching = true
	lastCandle := a.timeSeries.LastCandle()
	lastCandle.ClosePrice = big.NewDecimal(float64(stockPrice.Close))
	lastCandle.Period.End = commons.Unix(stockPrice.Timestamp)
}

func (a *Analyser) stopWatchingPrice() {
	a.isWatching = false
}

func (a *Analyser) appendPastStockPrice(stockPrice structs.StockPrice) {
	var lastTimestamp int64
	if len(a.timeSeries.Candles) > 0 {
		lastTimestamp = a.timeSeries.LastCandle().Period.Start.Unix()
	}
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

func (a *Analyser) needPriceFrom() int64 {
	var start int64
	if len(a.timeSeries.Candles) > 0 {
		start = a.timeSeries.LastCandle().Period.End.Unix()
	} else {
		start = commons.Today().Unix()
	}
	before := int64((MaxCandles/5)+2) * 7 * 24 * 60 * 60 // 100일/5일 -> 20주 + 2주 -> 대충 22주 전까지 데이터 긁어옴
	return start - before
}

func (a *Analyser) calculateStrategies() {
	closePrice := a.timeSeries.LastCandle().ClosePrice.Float()
	currentTime := a.timeSeries.LastCandle().Period.End
	for userid, events := range a.userStrategy {
		for orderside, event := range events {
			if event.event.IsTriggered(a.timeSeries.LastIndex(), nil) {
				event.event.OnEvent(currentTime, closePrice, a.stockID, int(orderside), userid, event.repeat)
			}
		}
	}
}

func (a *Analyser) hasStrategyOfOrderSide(userid uid, orderside int) bool {
	events, ok := a.userStrategy[userid]
	if !ok {
		return false
	}
	_, ok = events[techan.OrderSide(orderside)]
	return ok
}
