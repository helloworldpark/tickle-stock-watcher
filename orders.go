package main

import "fmt"
import "github.com/helloworldpark/tickle-stock-watcher/structs"

type action func(structs.User, []string) error

type order interface {
	name() string
	isValid([]string) error
	setAction(action)
	onAction(structs.User, []string) error
}

type inviteOrder struct {
	action action
}

func newInviteOrder() order {
	return &inviteOrder{}
}

func (o *inviteOrder) name() string {
	return "invite"
}

func (o *inviteOrder) setAction(a action) {
	o.action = a
}

func (o *inviteOrder) onAction(user structs.User, args []string) error {
	err := o.isValid(args)
	if err != nil {
		return err
	}
	return o.action(user, args)
}

func (o *inviteOrder) isValid(args []string) error {
	if len(args) != 1 {
		return mainError{msg: fmt.Sprintf("Invalid number of arguments: %d", len(args))}
	}
	return nil
}

type joinOrder struct {
	action action
}

func newJoinOrder() order {
	return &joinOrder{}
}

func (o *joinOrder) name() string {
	return "join"
}

func (o *joinOrder) isValid(args []string) error {
	if len(args) != 2 {
		return mainError{msg: fmt.Sprintf("Invalid number of arguments: %d", len(args))}
	}
	return nil
}

func (o *joinOrder) setAction(a action) {
	o.action = a
}

func (o *joinOrder) onAction(user structs.User, args []string) error {
	err := o.isValid(args)
	if err != nil {
		return err
	}
	return o.action(user, args)
}
