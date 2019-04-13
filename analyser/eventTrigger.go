package analyser

import (
	"github.com/sdcoffey/techan"
)

// EventCallback is a type of callback when the trigger is triggered.
type EventCallback func(price float64, stockid string, orderSide int)

// EventTrigger is an interface for triggering events.
type EventTrigger interface {
	OrderSide() techan.OrderSide
	IsTriggered(index int, record *techan.TradingRecord) bool
	SetCallback(callback EventCallback)
	OnEvent(price float64, stockid string, orderSide int)
}

type eventTrigger struct {
	orderSide techan.OrderSide
	rule      techan.Rule
	prefix    string
	callback  EventCallback
}

// NewEventTrigger will create an EventTrigger for notifiying price changes
func NewEventTrigger(orderSide techan.OrderSide, rule techan.Rule, callback EventCallback) EventTrigger {
	prefix := "[BUY] "
	if orderSide == techan.SELL {
		prefix = "[SELL] "
	}
	return &eventTrigger{
		orderSide: orderSide,
		rule:      rule,
		prefix:    prefix,
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

func (e *eventTrigger) OnEvent(price float64, stockid string, orderSide int) {
	e.callback(price, stockid, orderSide)
}
