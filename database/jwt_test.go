package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
)

func TestJWT(t *testing.T) {
	// envKey := "GOOGLE_APPLICATION_CREDENTIALS"
	// jsonPath := os.Getenv(envKey)
	jsonPath := "/Users/shp/Documents/projects/ticklemeta-203110-709122f3e3af.json"
	token, requestMore := CreateJWTToken(jsonPath)
	fmt.Printf("access token: %s\n", token.AccessToken)
	fmt.Printf("expires at: %v\n", token.Expiry)

	if requestMore {
		values := make(url.Values)
		values["grant_type"] = []string{"urn:ietf:params:oauth:grant-type:jwt-bearer"}
		values["assertion"] = []string{token.AccessToken}
		resp, err := http.PostForm("https://oauth2.googleapis.com/token", values)
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

		fmt.Println("RequestMore")
		fmt.Println(readJSON)
	}

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
	mapJSON, ok := readJSON.(map[string]interface{})
	if !ok {
		return
	}
	if itemsRaw, ok := mapJSON["items"]; ok {
		if items, ok := itemsRaw.([]interface{}); ok {
			for _, item := range items {
				fmt.Println(item)
			}
		}
	}

}
