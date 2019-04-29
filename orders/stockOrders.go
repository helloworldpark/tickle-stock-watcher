package orders

import (
	"fmt"
	"github.com/helloworldpark/tickle-stock-watcher/analyser"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

type stockOrders struct {
	action  Action
	name    string
	minArgc int
}

func (o *stockOrders) Name() string {
	return o.name
}

func (o *stockOrders) IsValid(args []string) error {
	if len(args) < o.minArgc {
		return orderError{msg: fmt.Sprintf("Invalid number of arguments: need more than %d, got %d", o.minArgc, len(args))}
	}
	return nil
}

func (o *stockOrders) SetAction(a Action) {
	o.action = a
}

func (o *stockOrders) OnAction(user structs.User, args []string) error {
	err := o.IsValid(args)
	if err != nil {
		return err
	}
	return o.action(user, args)
}

// NewStrategyOrder order 'strategy'
func NewStrategyOrder() Order {
	return &stockOrders{name: "strategy", minArgc: 0}
}

// NewStockOrder order 'stock'
func NewStockOrder() Order {
	return &stockOrders{name: "stock", minArgc: 1}
}

// Strategy implements order 'strategy'
func Strategy(broker analyser.BrokerAccess, onSuccess func(user structs.User, strategies []structs.UserStock)) Action {
	f := func(user structs.User, args []string) error {
		strategies := broker.AccessBroker().GetStrategy(user)
		if strategies == nil {
			return orderError{msg: "Failed to query your strategies."}
		}
		onSuccess(user, strategies)
		return nil
	}
	return f
}

// QueryStockByName implementation for order 'stock'
func QueryStockByName(stockinfo watcher.StockAccess, onSuccess func(user structs.User, stock structs.Stock)) Action {
	f := func(user structs.User, args []string) error {
		stockname := concat(args)
		stock, ok := stockinfo.AccessStockItemByName(stockname)
		if !ok {
			return orderError{fmt.Sprintf("Failed to find stock info by name %s", stockname)}
		}
		onSuccess(user, stock)
		return nil
	}
	return f
}
