package analyser

import (
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/sdcoffey/techan"
)

// EventCallback is a type of callback when the trigger is triggered.
type EventCallback func(price structs.StockPrice, orderSide int, userid int64, repeat bool)

// EventTrigger is an interface for triggering events.
type EventTrigger interface {
	OrderSide() techan.OrderSide
	IsTriggered(index int, record *techan.TradingRecord) bool
	SetCallback(callback EventCallback)
	OnEvent(price structs.StockPrice, orderSide int, userid int64, repeat bool)
}

type eventTrigger struct {
	orderSide techan.OrderSide
	rule      techan.Rule
	callback  EventCallback
}

// newEventTrigger will create an EventTrigger for notifiying price changes
func newEventTrigger(orderSide techan.OrderSide, rule techan.Rule, callback EventCallback) EventTrigger {
	return &eventTrigger{
		orderSide: orderSide,
		rule:      rule,
		callback:  callback,
	}
}

// Event
func (e *eventTrigger) OrderSide() techan.OrderSide {
	return e.orderSide
}

func (e *eventTrigger) IsTriggered(index int, record *techan.TradingRecord) bool {
	return e.rule.IsSatisfied(index, record)
}

func (e *eventTrigger) SetCallback(callback EventCallback) {
	e.callback = callback
}

func (e *eventTrigger) OnEvent(price structs.StockPrice, orderSide int, userid int64, repeat bool) {
	e.callback(price, orderSide, userid, repeat)
}
