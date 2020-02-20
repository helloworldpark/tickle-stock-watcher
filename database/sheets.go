package database

import (
	"fmt"
	"net/http"
	"strconv"

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
	resp, err := m.service.Spreadsheets.Create(rb).Do()
	if err != nil {
		panic(err)
	}
	fmt.Println("Spreadsheed ID: ", resp.SpreadsheetId)

	return resp
}

func (m *SheetManager) GetSpreadsheet(spreadsheetId string) *sheets.Spreadsheet {
	resp, err := m.service.Spreadsheets.Get(spreadsheetId).IncludeGridData(true).Do()
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
	sheetId, err := strconv.ParseInt(spreadsheetId, 10, 64)
	if err != nil {
		panic(err)
	}

	var deleteSheet sheets.DeleteSheetRequest
	deleteSheet.SheetId = sheetId
	var request []*sheets.Request
	request = append(request, &(sheets.Request{}))
	request[0] = &sheets.Request{}
	request[0].DeleteSheet = &deleteSheet
	m.service.Spreadsheets.BatchUpdate(spreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{Requests: request})
}
