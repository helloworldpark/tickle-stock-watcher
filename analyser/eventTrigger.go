package analyser

import (
	"github.com/sdcoffey/techan"
)

// EventTrigger is an interface for triggering events.
type EventTrigger interface {
	OrderSide() techan.OrderSide
	HasHappened(index int, record *techan.TradingRecord) bool
	Message(msg string) string
}

type eventTrigger struct {
	orderSide techan.OrderSide
	rule      techan.Rule
	prefix    string
}

// NewEventTrigger will create an EventTrigger for notifiying price changes
func NewEventTrigger(orderSide techan.OrderSide, rule techan.Rule) EventTrigger {
	prefix := "[BUY] "
	if orderSide == techan.SELL {
		prefix = "[SELL] "
	}
	return eventTrigger{
		orderSide: orderSide,
		rule:      rule,
		prefix:    prefix,
	}
}

// Event
func (e eventTrigger) OrderSide() techan.OrderSide {
	return e.orderSide
}

func (e eventTrigger) HasHappened(index int, record *techan.TradingRecord) bool {
	return e.rule.IsSatisfied(index, record)
}

func (e eventTrigger) Message(msg string) string {
	return e.prefix + msg
}
