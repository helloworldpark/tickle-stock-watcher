package commons

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/helloworldpark/tickle-stock-watcher/logger")

var encryptionKey = []byte("[&Hz7s7z9b~$?{VqmX.bb.S994g<hW7'")

// ResetEncryptionKey Use this to change the encryption key
func ResetEncryptionKey(newKey string) {
	encryptionKey = []byte(newKey)
}

// Encrypt Encrypts message.
func Encrypt(msg string) string {
	plaintext := []byte(msg)
	paddingCount := aes.BlockSize - (len(plaintext) % aes.BlockSize)
	padding := make([]byte, paddingCount)
	for idx := range padding {
		padding[idx] = byte(paddingCount)
	}
	plaintext = append(plaintext, padding...)

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		logger.Panic("[Commons][Crypto] Error at Encrypt.NewCipher: %+v", err)
	}
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		logger.Panic("[Commons][Crypto] Error at Encrypt.IVGeneration: %+v", err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	return hex.EncodeToString(ciphertext)
}

// Decrypt Decrypts message. Length of message must be longer than aes.BlockSize
func Decrypt(msg string) string {
	ciphertext, _ := hex.DecodeString(msg)

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		logger.Panic("[Commons][Crypto] Error at Decrypt.NewCipher: %+v", err)
	}
	if len(ciphertext) < aes.BlockSize {
		logger.Panic("[Commons][Crypto] Error at Decrypt: ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	if len(ciphertext)%aes.BlockSize != 0 {
		logger.Panic("[Commons][Crypto] Error at Decrypt: ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)
	paddingCount := int(ciphertext[len(ciphertext)-1])
	ciphertext = ciphertext[:len(ciphertext)-paddingCount]

	return string(ciphertext)
}

func validateEncryption(plaintext, ciphertext string) bool {
	return plaintext == Decrypt(ciphertext)
}
