package commons

import (
	"fmt"
	"testing"
)

func TestCrypto(t *testing.T) {
	plaintext := "여기 한국인 있나요?"
	ciphertext := Encrypt(plaintext)
	fmt.Printf("Plaintext: %s\n", plaintext)
	decrypted := Decrypt(ciphertext)
	fmt.Printf("Decrypted: %s\n", decrypted)
	fmt.Printf("Did success: %v\n", validateEncryption(plaintext, ciphertext))
}
