package analyser

import (
	"github.com/sdcoffey/techan"
)

type Event interface {
	OrderSide() techan.OrderSide
	HasHappened(index int, record *techan.TradingRecord) bool
	Message(msg string) string
}

type event struct {
	orderSide techan.OrderSide
	rule      techan.Rule
	prefix    string
}

type buyEvent struct {
	event
}

type sellEvent struct {
	event
}

func NewEvent(orderSide techan.OrderSide, rule techan.Rule) Event {
	if orderSide == techan.BUY {
		return buyEvent{event{
			orderSide: techan.BUY,
			rule:      rule,
			prefix:    "[BUY] ",
		}}
	}
	return sellEvent{event{
		orderSide: techan.SELL,
		rule:      rule,
		prefix:    "[SELL] ",
	}}
}

// Event
func (e buyEvent) OrderSide() techan.OrderSide {
	return e.orderSide
}

func (e buyEvent) HasHappened(index int, record *techan.TradingRecord) bool {
	return e.rule.IsSatisfied(index, record)
}

func (e buyEvent) Message(msg string) string {
	return e.prefix + msg
}

func (e sellEvent) OrderSide() techan.OrderSide {
	return e.orderSide
}

func (e sellEvent) HasHappened(index int, record *techan.TradingRecord) bool {
	return e.rule.IsSatisfied(index, record)
}

func (e sellEvent) Message(msg string) string {
	return e.prefix + msg
}
