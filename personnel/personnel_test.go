package personnel

import (
	"fmt"
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

func TestInvite(t *testing.T) {
	user := structs.User{Superuser: true}
	signature, invitation, err := Invite(user, "503652742")
	fmt.Println(err)
	// fmt.Printf("Signature: %s\n", signature)
	err = ValidateInvitation(invitation, signature)
	fmt.Println(err)
}
