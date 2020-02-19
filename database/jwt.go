package database

import (
	"io/ioutil"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func CreateJWTToken(jsonPath string) *oauth2.Token {
	jsonKey, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		panic(err)
	}
	tokenSource, err := google.JWTAccessTokenSourceFromJSON(jsonKey, "https://stock.ticklemeta.kr")
	if err != nil {
		panic(err)
	}
	token, err := tokenSource.Token()
	if err != nil {
		panic(err)
	}
	return token
}
