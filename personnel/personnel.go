package personnel

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"fmt"

	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

type InvitationError struct {
	msg string
}

func (err InvitationError) Error() string {
	return fmt.Sprintf("[Personnel] %s", err.msg)
}

func Invite(user structs.User, guesttoken, messenger string) (string, structs.Invitation, error) {
	// Superuser가 아니면 이 기능은 못 쓰도록 막는다
	if !user.Superuser {
		return "", structs.Invitation{}, InvitationError{"Unauthorized to invite others"}
	}
	rng := rand.Reader
	privateKey, err := rsa.GenerateKey(rng, 2048)
	if err != nil {
		return "", structs.Invitation{}, InvitationError{err.Error()}
	}
	message := []byte(fmt.Sprintf("%s%s", guesttoken, messenger))
	hashed := sha512.Sum512(message)
	signature, err := rsa.SignPKCS1v15(rng, privateKey, crypto.SHA512, hashed[:])
	if err != nil {
		return "", structs.Invitation{}, InvitationError{err.Error()}
	}
	sign := string(signature[:])
	invitation := structs.NewInvitation(string(hashed[:]), &privateKey.PublicKey)
	return sign, invitation, nil
}

func ValidateInvitation(invitation structs.Invitation, guesttoken, messenger, signature string) error {
	message := []byte(fmt.Sprintf("%s%s", guesttoken, messenger))
	hashed := sha512.Sum512(message)
	if string(hashed[:]) != invitation.Hashed {
		return InvitationError{fmt.Sprintf("Validation for invitation failed: %s, %s", guesttoken, messenger)}
	}
	sign := []byte(signature)
	publicKey := invitation.GetPublicKey()
	err := rsa.VerifyPKCS1v15(&publicKey, crypto.SHA512, hashed[:], sign)
	return err
}
