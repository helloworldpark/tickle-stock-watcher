package orders

import "fmt"
import "github.com/helloworldpark/tickle-stock-watcher/structs"

type Action func(structs.User, []string) error

type orderError struct {
	msg string
}

func (err orderError) Error() string {
	return fmt.Sprintf("[Order] %s", err.msg)
}

type Order interface {
	Name() string
	IsValid([]string) error
	SetAction(Action)
	OnAction(structs.User, []string) error
}
