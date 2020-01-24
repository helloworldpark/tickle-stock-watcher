package orders

import (
	"github.com/helloworldpark/tickle-stock-watcher/analyser"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

type watchOrders struct {
	action Action
	name   string
}

func (o *watchOrders) Name() string {
	return o.name
}

func (o *watchOrders) IsValid(args []string) error {
	return nil
}

func (o *watchOrders) SetAction(a Action) {
	o.action = a
}

func (o *watchOrders) OnAction(user structs.User, args []string) error {
	return o.action(user, args)
}

func (o *watchOrders) IsAsync() bool {
	return false
}

func (o *watchOrders) IsPublic() bool {
	return false
}

func (o *watchOrders) ShouldLowerArgs() bool {
	return false
}

// NewWatcherDescriptionOrder order 'watcher'
func NewWatcherDescriptionOrder() Order {
	return &watchOrders{name: "watcher"}
}

// WatcherDescription implements order 'watcher'
func WatcherDescription(watcher watcher.WatcherAccess, onSuccess func(user structs.User, desc string)) Action {
	f := func(user structs.User, args []string) error {
		if !user.Superuser {
			return newError("Only superuser can order this")
		}
		description := watcher.AccessWatcher().Description()
		onSuccess(user, description)
		return nil
	}
	return f
}

func NewBrokerDescriptionOrder() Order {
	return &watchOrders{name: "analyser"}
}

func BrokerDescription(broker analyser.BrokerAccess, onSuccess func(user structs.User, desc string)) Action {
	f := func(user structs.User, args []string) error {
		if !user.Superuser {
			return newError("Only superuser can order this")
		}
		description := broker.AccessBroker().Description()
		onSuccess(user, description)
		return nil
	}
	return f
}

func NewDateCheckerDescriptionOrder() Order {
	return &watchOrders{name: "holiday"}
}

func DateCheckerDescription(dateChecker *watcher.DateChecker, onSuccess func(user structs.User, desc string)) Action {
	f := func(user structs.User, args []string) error {
		if !user.Superuser {
			return newError("Only superuser can order this")
		}
		description := dateChecker.Description()
		onSuccess(user, description)
		return nil
	}
	return f
}
