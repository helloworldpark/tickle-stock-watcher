package structs

import (
	"crypto/rsa"
	"encoding/json"

	"github.com/helloworldpark/tickle-stock-watcher/database"
)

// Invitation is a struct for describing invitation
type Invitation struct {
	Guestname string
	PublicKey string
}

func NewInvitation(guestname string, publicKey *rsa.PublicKey) Invitation {
	marshaled, _ := json.Marshal(publicKey)
	return Invitation{Guestname: guestname, PublicKey: string(marshaled)}
}

func (iv Invitation) GetPublicKey() rsa.PublicKey {
	publicKey := rsa.PublicKey{}
	json.Unmarshal([]byte(iv.PublicKey), &publicKey)
	return publicKey
}

// GetDBRegisterForm is just an implementation
func (s Invitation) GetDBRegisterForm() database.DBRegisterForm {
	form := database.DBRegisterForm{
		BaseStruct:    Invitation{},
		UniqueColumns: []string{"PublicKey"},
	}
	return form
}
