package orders

import (
	"fmt"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/personnel"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

type personnelOrder struct {
	action Action
	name   string
	argc   int
}

func (o *personnelOrder) Name() string {
	return o.name
}

func (o *personnelOrder) IsValid(args []string) error {
	if len(args) != o.argc {
		return orderError{msg: fmt.Sprintf("Invalid number of arguments: need %d, got %d", o.argc, len(args))}
	}
	return nil
}

func (o *personnelOrder) SetAction(a Action) {
	o.action = a
}

func (o *personnelOrder) OnAction(user structs.User, args []string) error {
	err := o.IsValid(args)
	if err != nil {
		return err
	}
	return o.action(user, args)
}

// NewInviteOrder returns a invite order
func NewInviteOrder() Order {
	return &personnelOrder{name: "invite", argc: 1}
}

// NewJoinOrder returns a join order
func NewJoinOrder() Order {
	return &personnelOrder{name: "join", argc: 2}
}

// Invite returns a invite functionality
func Invite(db database.DBAccess, onSuccess func(user structs.User, signature string)) Action {
	f := func(user structs.User, args []string) error {
		guestname := args[0]
		signature, invitation, err := personnel.Invite(user, guestname)
		if err != nil {
			logger.Error("%s", err.Error())
			return orderError{msg: err.Error()}
		}
		_, err = db.AccessDB().Insert(&invitation)
		if err != nil {
			logger.Error("%s", err.Error())
			return orderError{msg: err.Error()}
		}
		logger.Info("[Invite] Invitation signature created: %s", signature)
		onSuccess(user, signature)
		return nil
	}
	return f
}

// Join returns a join functionality
func Join(db database.DBAccess, onSuccess func(user structs.User)) Action {
	f := func(user structs.User, args []string) error {
		username := args[0]
		signature := args[1]
		var invitation []structs.Invitation
		_, err := db.AccessDB().Select(&invitation, "where Guestname=?", username)
		if err != nil {
			return err
		}
		if len(invitation) == 0 {
			return orderError{msg: fmt.Sprintf("No invitation issued for username %s", username)}
		}
		err = personnel.ValidateInvitation(invitation[0], signature)
		if err != nil {
			return err
		}

		user.Superuser = false

		_, err = db.AccessDB().Insert(&user)
		if err != nil {
			return err
		}

		db.AccessDB().Delete(structs.Invitation{}, "where Guestname=?", username)
		onSuccess(user)

		return nil
	}
	return f
}
