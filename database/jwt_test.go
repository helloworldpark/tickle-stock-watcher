package database

import (
	"fmt"
	"os"
	"testing"
)

func TestJWT(t *testing.T) {
	envKey := "GOOGLE_APPLICATION_CREDENTIALS"
	jsonPath := os.Getenv(envKey)
	token := CreateJWTToken(jsonPath)
	fmt.Printf("access token: %s\n", token.AccessToken)
	fmt.Printf("expires at: %v\n", token.Expiry)

}
