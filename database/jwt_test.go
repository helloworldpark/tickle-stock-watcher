package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func TestJWT(t *testing.T) {
	envKey := "GOOGLE_APPLICATION_CREDENTIALS"
	jsonPath := os.Getenv(envKey)
	token := CreateJWTToken(jsonPath)
	fmt.Printf("access token: %s\n", token.AccessToken)
	fmt.Printf("expires at: %v\n", token.Expiry)

	testDriveAPI(token.AccessToken)
}

func testDriveAPI(accessToken string) {
	fmt.Println("Test Drive API")

	req, err := http.NewRequest("GET", fmt.Sprintf("https://www.googleapis.com/drive/v2/files?access_token=%s", accessToken), nil)
	if err != nil {
		panic(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	read, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		panic(err)
	}

	var readJSON interface{}
	json.Unmarshal(read, &readJSON)

	fmt.Println("FILES")
	fmt.Println(readJSON)
}
