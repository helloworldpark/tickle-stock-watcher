package database

import (
	"context"
	"fmt"
	"io/ioutil"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/sheets/v4"
)

var scope = []string{drive.DriveScope, drive.DriveFileScope, sheets.SpreadsheetsScope}

func CreateJWTToken(jsonPath string) *oauth2.Token {
	cred, err := google.FindDefaultCredentials(context.Background(), scope...)
	if err != nil {
		fmt.Errorf("%+v", err.Error())
	}
	jsonKey, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		fmt.Errorf("%+v", err.Error())
	}

	var tokenSource oauth2.TokenSource

	if cred != nil && false {
		tokenSource = cred.TokenSource
		fmt.Println("Credential is not nil")
	} else if len(jsonKey) > 0 {
		cfg, err := google.JWTConfigFromJSON(jsonKey, drive.DriveScope, drive.DriveFileScope)
		if err != nil {
			fmt.Errorf("%+v", err.Error())
		}
		tokenSource = cfg.TokenSource(context.Background())
		a, _ := tokenSource.Token()
		fmt.Println("Token Source From JWTConfigFromJSON", a)
	} else {
		panic("No way")
	}

	token, err := tokenSource.Token()
	if err != nil {
		fmt.Errorf("%+v", err.Error())
	}
	return token
}
