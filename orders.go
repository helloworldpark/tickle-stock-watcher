package main

import "fmt"

type action func([]string) error

type order interface {
	name() string
	isValid([]string) error
	setAction(action)
	onAction([]string) error
}

type inviteOrder struct {
	action action
}

func newInviteOrder() *inviteOrder {
	return &inviteOrder{}
}

func (o *inviteOrder) name() string {
	return "invite"
}

func (o *inviteOrder) setAction(a action) {
	o.action = a
}

func (o *inviteOrder) onAction(args []string) error {
	err := o.isValid(args)
	if err != nil {
		return err
	}
	return o.action(args)
}

func (o *inviteOrder) isValid(args []string) error {
	if len(args) != 1 {
		return mainError{msg: fmt.Sprintf("Invalid number of arguments: %d", len(args))}
	}
	return nil
}
