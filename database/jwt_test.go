package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

const jsonPath = "/Users/shp/Documents/projects/ticklemeta-203110-709122f3e3af.json"

func TestJWT(t *testing.T) {
	// envKey := "GOOGLE_APPLICATION_CREDENTIALS"
	// jsonPath := os.Getenv(envKey)
	token := CreateJWTToken(jsonPath)
	fmt.Printf("access token: %s\n", token.AccessToken)
	fmt.Printf("expires at: %v\n", token.Expiry)

	// testDriveFileAPI(token.AccessToken)
	testDriveAboutAPI(token.AccessToken)
}

func testDriveFileAPI(accessToken string) {
	fmt.Println("Test Drive File API")

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

	mapJSON, ok := readJSON.(map[string]interface{})
	if !ok {
		return
	}
	if itemsRaw, ok := mapJSON["items"]; ok {
		if items, ok := itemsRaw.([]interface{}); ok {
			fmt.Println("Items: ", len(items))
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					for k, v := range itemMap {
						fmt.Printf("%s:    %v\n", k, v)
					}
					fmt.Println("------------------------")
				}
			}
		}
	}
}

func testDriveAboutAPI(accessToken string) {
	fmt.Println("Test Drive About API")

	req, err := http.NewRequest("GET", fmt.Sprintf("https://www.googleapis.com/drive/v3/about?access_token=%s&fields=*", accessToken), nil)
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

	mapJSON, ok := readJSON.(map[string]interface{})
	if !ok {
		return
	}
	for k, v := range mapJSON {
		fmt.Printf("%s:    %v\n", k, v)
	}
	fmt.Println("------Storage Quota------")
	if storageRaw, ok := mapJSON["storageQuota"]; ok {
		if storage, ok := storageRaw.(map[string]interface{}); ok {
			fmt.Println("Storage: ", len(storage))
			for k, v := range storage {
				fmt.Printf("%s:    %v\n", k, v)
			}
			fmt.Println("------------------------")
		}
	}
}
