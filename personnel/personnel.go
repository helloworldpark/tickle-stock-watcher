package personnel

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"fmt"
	"strconv"
	"strings"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

var newError = commons.NewTaggedError("Personnel")

// Invite generates an RSA public key and a signature
func Invite(user structs.User, guestname string) (string, structs.Invitation, error) {
	// Superuser가 아니면 이 기능은 못 쓰도록 막는다
	if !user.Superuser {
		return "", structs.Invitation{}, newError("Unauthorized to invite others")
	}
	rng := rand.Reader
	privateKey, err := rsa.GenerateKey(rng, 2048)
	if err != nil {
		return "", structs.Invitation{}, newError(err.Error())
	}
	message := []byte(guestname)
	hashed := sha512.Sum512(message)
	signature, err := rsa.SignPKCS1v15(rng, privateKey, crypto.SHA512, hashed[:])
	if err != nil {
		return "", structs.Invitation{}, newError(err.Error())
	}
	sign := encodeByteArray(signature[:])
	invitation := structs.NewInvitation(guestname, &privateKey.PublicKey)
	return sign, invitation, nil
}

// ValidateInvitation validates the signature with a public key saved before
func ValidateInvitation(invitation structs.Invitation, signature string) error {
	message := []byte(invitation.Guestname)
	hashed := sha512.Sum512(message)
	sign, err := decodeToByteArray(signature)
	if err != nil {
		return err
	}
	publicKey := invitation.GetPublicKey()
	err = rsa.VerifyPKCS1v15(&publicKey, crypto.SHA512, hashed[:], sign)
	if err != nil {
		return newError(err.Error())
	}
	return nil
}

func encodeByteArray(b []byte) string {
	format := "%02x"
	buffer := strings.Builder{}
	for i := range b {
		buffer.WriteString(fmt.Sprintf(format, b[i]))
	}
	return buffer.String()
}

func decodeToByteArray(s string) ([]byte, error) {
	if len(s)%2 == 1 {
		return nil, newError(fmt.Sprintf("Invalid parameter %s: s should have even numbers of characters", s))
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s)/2; i++ {
		v, err := strconv.ParseUint(s[2*i:2*i+2], 16, 8)
		if err != nil {
			return nil, newError(err.Error())
		}
		b[i] = byte(v)
	}
	return b, nil
}
