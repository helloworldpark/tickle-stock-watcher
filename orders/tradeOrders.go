package orders

import (
	"bytes"
	"fmt"

	"github.com/helloworldpark/tickle-stock-watcher/analyser"
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

type tradeOrders struct {
	action    Action
	name      string
	orderSide int
}

func (o *tradeOrders) Name() string {
	return o.name
}

func (o *tradeOrders) IsValid(args []string) error {
	if len(args) < 2 {
		return newError(fmt.Sprintf("Invalid number of arguments: need more than %d, got %d", 1, len(args)))
	}
	return nil
}

func (o *tradeOrders) SetAction(a Action) {
	o.action = a
}

func (o *tradeOrders) OnAction(user structs.User, args []string) error {
	err := o.IsValid(args)
	if err != nil {
		return err
	}
	return o.action(user, args)
}

func (o *tradeOrders) IsAsync() bool {
	return true
}

func (o *tradeOrders) IsPublic() bool {
	return false
}

// NewBuyOrder order for 'buy'
func NewBuyOrder() Order {
	return &tradeOrders{name: "buy", orderSide: commons.BUY}
}

// NewSellOrder order for 'sell'
func NewSellOrder() Order {
	return &tradeOrders{name: "sell", orderSide: commons.SELL}
}

func concat(s []string) string {
	if len(s) == 0 {
		return ""
	}
	buffer := bytes.Buffer{}
	for i := range s {
		buffer.WriteString(s[i])
	}
	return buffer.String()
}

// Trade implements order 'buy' 'sell'
func Trade(
	orderSide int,
	broker analyser.BrokerAccess,
	stockinfo watcher.StockAccess,
	price watcher.WatcherAccess,
	callback analyser.EventCallback,
	onSuccess func(user structs.User, orderside int, stockname, stockid, strategy string)) Action {
	f := func(user structs.User, args []string) error {
		stockid := args[0]
		stock, ok := stockinfo.AccessStockItem(stockid)
		if !ok {
			return newError(fmt.Sprintf("Invalid stock id: %s", stockid))
		}
		strategy := concat(args[1:])

		userStrategy := structs.UserStock{
			UserID:    user.UserID,
			StockID:   stockid,
			Strategy:  strategy,
			OrderSide: orderSide,
			Repeat:    false,
		}
		// Add to analyser
		shouldRetainWatcher, err := broker.AccessBroker().AddStrategy(userStrategy, callback, true)
		if err != nil {
			return newError(err.Error())
		}
		// Add to watcher
		if shouldRetainWatcher {
			ok = price.AccessWatcher().Register(stock)
			if !ok {
				return newError(fmt.Sprintf("Failed to add %s(%s) to PriceWatcher", stock.Name, stock.StockID))
			}
		}
		nowHour := commons.Now().Hour()
		if 9 <= nowHour && float64(nowHour) < 15.5 {
			if broker.AccessBroker().CanFeedPrice(stockid) {
				broker.AccessBroker().FeedPrice(stockid, price.AccessWatcher().StartWatchingStock(stock.StockID))
			}
		}
		onSuccess(user, orderSide, stock.Name, stockid, strategy)
		return nil
	}
	return f
}
