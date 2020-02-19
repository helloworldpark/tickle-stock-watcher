package database

import (
	"context"
	"fmt"
	"io/ioutil"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
)

func CreateJWTToken(jsonPath string) *oauth2.Token {
	cred, err := google.FindDefaultCredentials(context.Background(), drive.DriveScope, drive.DriveFileScope)
	if err != nil {
		fmt.Errorf("%+v", err.Error())
	}
	jsonKey, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		fmt.Errorf("%+v", err.Error())
	}

	var tokenSource oauth2.TokenSource

	if cred != nil {
		tokenSource = cred.TokenSource
		fmt.Println("Credential is not nil")
	} else if len(jsonKey) > 0 {
		tokenSource, err = google.JWTAccessTokenSourceFromJSON(jsonKey, "https://stock.ticklemeta.kr")
		fmt.Println("Token Source From JWTAccessTokenSourceFromJSON")
		if err != nil {
			fmt.Errorf("%+v", err.Error())
		}
	} else {
		panic("No way")
	}

	token, err := tokenSource.Token()
	if err != nil {
		fmt.Errorf("%+v", err.Error())
	}
	return token
}
