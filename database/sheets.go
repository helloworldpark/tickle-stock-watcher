package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/oauth2"
	"google.golang.org/api/sheets/v4"
)

const dbFileStart = "database_file_"

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

// Database
func (m *SheetManager) CreateSpreadsheet(title string) *sheets.Spreadsheet {
	rb := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title:      title,
			TimeZone:   "Asia/Seoul",
			AutoRecalc: "ON_CHANGE",
		},
	}
	req := m.service.Spreadsheets.Create(rb)
	req.Header().Add("Authorization", "Bearer "+m.token.AccessToken)
	resp, err := req.Do()
	if err != nil {
		panic(err)
	}

	return resp
}

func (m *SheetManager) GetSpreadsheet(spreadsheetId string) *sheets.Spreadsheet {
	req := m.service.Spreadsheets.Get(spreadsheetId).IncludeGridData(true)
	req.Header().Add("Authorization", "Bearer "+m.token.AccessToken)
	resp, err := req.Do()
	if err != nil {
		panic(err)
	}
	return resp
}

// https://stackoverflow.com/questions/46836393/how-do-i-delete-a-spreadsheet-file-using-google-spreadsheets-api
// https://stackoverflow.com/questions/46310113/consume-a-delete-endpoint-from-golang
func (m *SheetManager) DeleteSpreadsheet(spreadsheetId string) bool {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s", spreadsheetId), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", "Bearer "+m.token.AccessToken)
	resp, err := m.client.Do(req)
	if err != nil {
		panic(err)
	}
	return resp.StatusCode/200 == 1
}

func (m *SheetManager) ListSpreadsheets() []string {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/drive/v3/files", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", "Bearer "+m.token.AccessToken)
	// https://stackoverflow.com/questions/30652577/go-doing-a-get-request-and-building-the-querystring
	// https://developers.google.com/drive/api/v3/mime-types
	shouldBe := "mimeType='application/vnd.google-apps.spreadsheet'"
	values := req.URL.Query()
	values.Add("q", shouldBe)
	req.URL.RawQuery = values.Encode()

	resp, err := m.client.Do(req)
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
		return nil
	}

	files, ok := mapJSON["files"]
	if !ok {
		return nil
	}
	fileArr, ok := files.([]interface{})
	if !ok {
		return nil
	}

	var sheetArr []string
	for _, f := range fileArr {
		fmap, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		trashed, ok := fmap["trashed"].(bool)
		if ok && trashed {
			continue
		}
		fileID, ok := fmap["id"].(string)
		if !ok {
			continue
		}
		sheetArr = append(sheetArr, fileID)

	}
	return sheetArr
}

func (m *SheetManager) FindSpreadsheet(title string) *sheets.Spreadsheet {
	sheetIDs := m.ListSpreadsheets()
	for _, sheetID := range sheetIDs {
		s := m.GetSpreadsheet(sheetID)
		if s.Properties.Title == title {
			return s
		}
	}
	return nil
}

// Database alias api
func (m *SheetManager) CreateDatabase(title string) *sheets.Spreadsheet {
	return m.CreateSpreadsheet(title)
}

func (m *SheetManager) FindDatabase(title string) *sheets.Spreadsheet {
	return m.FindSpreadsheet(title)
}

func (m *SheetManager) DeleteDatabase(title string) bool {
	db := m.FindDatabase(title)
	if db == nil {
		return true
	}
	return m.DeleteSpreadsheet(db.SpreadsheetId)
}

// Table api
func (m *SheetManager) CreateTable(database *sheets.Spreadsheet, tableName string) *sheets.Sheet {

}

func (m *SheetManager) GetTable(database *sheets.Spreadsheet, tableName string) *sheets.Sheet {

}

func (m *SheetManager) DeleteTable(database *sheets.Spreadsheet, tableName string) bool {

}
