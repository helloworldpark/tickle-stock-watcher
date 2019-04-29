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
}
