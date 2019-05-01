package orders

import "fmt"
import "github.com/helloworldpark/tickle-stock-watcher/structs"

// Action is a function type used by Order
type Action func(structs.User, []string) error

type orderError struct {
	msg string
}

func (err orderError) Error() string {
	return fmt.Sprintf("[Order] %s", err.msg)
}

// Order is an interface for user's orders to the bot
type Order interface {
	Name() string
	IsValid([]string) error
	SetAction(Action)
	OnAction(structs.User, []string) error
	IsAsync() bool
}

type simpleOrder struct {
	name   string
	action Action
}

func (o *simpleOrder) Name() string {
	return o.name
}

func (o *simpleOrder) IsValid(s []string) error {
	return nil
}

func (o *simpleOrder) SetAction(a Action) {
	o.action = a
}

func (o *simpleOrder) OnAction(user structs.User, s []string) error {
	return o.action(user, s)
}

func (o *simpleOrder) IsAsync() bool {
	return false
}

// NewHelpOrder order help
func NewHelpOrder() Order {
	return &simpleOrder{name: "help"}
}
