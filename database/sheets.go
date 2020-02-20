package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/oauth2"
	"google.golang.org/api/sheets/v4"
)

type SheetManager struct {
	client  *http.Client
	service *sheets.Service
	token   *oauth2.Token
}

func NewSheetManager(jsonPath string) *SheetManager {
	client := http.DefaultClient
	service, _ := sheets.New(client)
	token := CreateJWTToken(jsonPath)

	m := &SheetManager{
		client:  client,
		token:   token,
		service: service,
	}
	return m
}

func (m *SheetManager) CreateSpreadsheet(title string) *sheets.Spreadsheet {
	rb := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title:      title,
			TimeZone:   "GMT+09:00",
			AutoRecalc: "ON_CHANGE",
		},
	}
	req := m.service.Spreadsheets.Create(rb)
	req.Header().Add("Authorization", "Bearer "+m.token.AccessToken)
	resp, err := req.Do()
	if err != nil {
		panic(err)
	}
	fmt.Println("Spreadsheed ID: ", resp.SpreadsheetId)

	return resp
}

func (m *SheetManager) GetSpreadsheet(spreadsheetId string) *sheets.Spreadsheet {
	req := m.service.Spreadsheets.Get(spreadsheetId).IncludeGridData(true)
	req.Header().Add("Authorization", "Bearer "+m.token.AccessToken)
	resp, err := req.Do()
	if err != nil {
		panic(err)
	}
	if resp == nil {
		fmt.Println("Not Found: Spreadsheed ID: ", resp.SpreadsheetId)
	} else {
		fmt.Println("    Found: Spreadsheed ID: ", resp.SpreadsheetId)
	}
	return resp
}

func (m *SheetManager) DeleteSpreadsheet(spreadsheetId string) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s", spreadsheetId), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", "Bearer "+m.token.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Delete %s: %d\n", spreadsheetId, resp.StatusCode)
	read, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		panic(err)
	}

	var readJSON interface{}
	json.Unmarshal(read, &readJSON)

	mapJSON, ok := readJSON.(map[string]interface{})
	if ok {
		for k, v := range mapJSON {
			fmt.Printf("%s    %v\n", k, v)
			fmt.Println("------------")
		}
	} else {
		fmt.Println(readJSON)
	}
}
