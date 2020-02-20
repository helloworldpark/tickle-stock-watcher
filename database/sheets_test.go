package database

import (
	"fmt"
	"testing"
)

// create, get, delete
func TestSpreadsheet001(t *testing.T) {
	manager := NewSheetManager(jsonPath)
	sheet := manager.CreateSpreadsheet("Test First!")
	fmt.Println("------Created sheet------")
	fmt.Println("Sheet ID: ", sheet.SpreadsheetId)
	fmt.Println("Sheet Name: ", sheet.Properties.Title)
	fmt.Println("Sheet Timezone: ", sheet.Properties.TimeZone)
	sheetId := sheet.SpreadsheetId
	manager.DeleteSpreadsheet(sheet.SpreadsheetId)
	fmt.Println("------Deleted sheet ", sheetId, "------")
	sheet = manager.GetSpreadsheet(sheetId)
	fmt.Println("------Get sheet ", sheetId, "------")
	fmt.Println("------    Result ", sheet)
}

// get
func TestSpreadsheet002(t *testing.T) {
	manager := NewSheetManager(jsonPath)
	sheetId := "1QMUZpqgBCHWEFQ7YEWBwmWwkrtU8yNOTJD0srqt4aFc"
	sheet := manager.GetSpreadsheet(sheetId)
	fmt.Println("------Get sheet ", sheetId, "------")
	fmt.Println("------    Result ", sheet.SpreadsheetId)
}
