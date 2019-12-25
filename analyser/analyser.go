package analyser

import (
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
	// maxCandles for analysers: only hold price of last maxCandles days
	maxCandles = 10001
)

// Analyser is a struct for signalling to users by condition they have set.
type Analyser struct {
	userStrategy map[uid]map[techan.OrderSide]eventWrapper
	timeSeries   *techan.TimeSeries
	counter      *commons.Ref
	stockID      string
	isWatching   bool
}

// NewAnalyser creates and returns a pointer of a new prepared Analyser struct
func NewAnalyser(stockID string) *Analyser {
	newAnalyser := Analyser{}
	newAnalyser.userStrategy = make(map[uid]map[techan.OrderSide]eventWrapper)
	newAnalyser.timeSeries = techan.NewTimeSeries()
	newAnalyser.counter = &commons.Ref{}
	newAnalyser.stockID = stockID
	newAnalyser.isWatching = false
	return &newAnalyser
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

/**
 * Strategy-related
 */

// AppendStrategy Appends strategy with callback
func (a *Analyser) AppendStrategy(strategy structs.UserStock, callback EventCallback) (bool, error) {
	// First, parse tokens
	tmpTokens, err := parseTokens(strategy.Strategy)
	if err != nil {
		return false, err
	}

	newTokens, err := tidyTokens(tmpTokens)
	if err != nil {
		return false, err
	}

	postfixToken, err := reorderTokenByPostfix(newTokens)
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

func (a *Analyser) createEvent(tokens []function, orderSide techan.OrderSide, callback EventCallback) (EventTrigger, error) {
	rule, err := a.createRule(tokens)
	if err != nil {
		return nil, err
	}
	eventTrigger := newEventTrigger(orderSide, rule, callback)
	return eventTrigger, nil
}

// DeleteStrategy Deletes strategy of a user with side
func (a *Analyser) DeleteStrategy(userid int64, orderside techan.OrderSide) {
	delete(a.userStrategy[userid], orderside)
	if len(a.userStrategy[userid]) == 0 {
		delete(a.userStrategy, userid)
	}
}

// CalculateStrategies calculates strategies from the last candle
func (a *Analyser) CalculateStrategies() {
	price := candleToStockPrice(a.stockID, a.timeSeries.LastCandle(), true)
	for userid, events := range a.userStrategy {
		for orderside, event := range events {
			if event.event.IsTriggered(a.timeSeries.LastIndex(), nil) {
				event.event.OnEvent(price, int(orderside), userid, event.repeat)
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

/**
 * Price-watching
 */

func (a *Analyser) prepareWatching() {
	newCandle := techan.NewCandle(techan.NewTimePeriod(commons.Today(), time.Hour*24))
	a.timeSeries.AddCandle(newCandle)
	if len(a.timeSeries.Candles) > maxCandles {
		a.timeSeries.Candles = a.timeSeries.Candles[len(a.timeSeries.Candles)-maxCandles:]
	}
}

func (a *Analyser) isWatchingPrice() bool {
	return a.isWatching
}

func (a *Analyser) watchPrice(stockPrice structs.StockPrice) {
	a.isWatching = true
	lastCandle := a.timeSeries.LastCandle()
	lastCandle.ClosePrice = big.NewDecimal(float64(stockPrice.Close))
	lastCandle.Period.End = commons.Unix(stockPrice.Timestamp)
}

func (a *Analyser) stopWatchingPrice() {
	a.isWatching = false
}

// AppendPastPrice appends price into the time series
func (a *Analyser) AppendPastPrice(stockPrice structs.StockPrice) {
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
		start := commons.Unix(stockPrice.Timestamp)
		candle = techan.NewCandle(techan.NewTimePeriod(start, time.Hour*24))
	}
	candle.OpenPrice = big.NewDecimal(float64(stockPrice.Open))
	candle.ClosePrice = big.NewDecimal(float64(stockPrice.Close))
	candle.MaxPrice = big.NewDecimal(float64(stockPrice.High))
	candle.MinPrice = big.NewDecimal(float64(stockPrice.Low))
	candle.Volume = big.NewDecimal(float64(stockPrice.Volume))
	if a.timeSeries.AddCandle(candle) && len(a.timeSeries.Candles) > maxCandles {
		a.timeSeries.Candles = a.timeSeries.Candles[(len(a.timeSeries.Candles) - maxCandles):]
	}
}

// NeedPriceFrom calculates timestamp from when to fetch coin price data.
func (a *Analyser) NeedPriceFrom() int64 {
	var start int64
	if len(a.timeSeries.Candles) > 0 {
		start = a.timeSeries.LastCandle().Period.Start.Unix()
	} else {
		start = commons.Today().Unix()
	}
	before := int64(24 * 60 * 60 * maxCandles) // maxCandles * 일 만큼(단위: 초)
	return start - before
}
