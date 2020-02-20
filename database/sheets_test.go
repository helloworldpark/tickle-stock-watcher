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
func TestGetSpreadsheet(t *testing.T) {
	manager := NewSheetManager(jsonPath)
	sheetId := "1QMUZpqgBCHWEFQ7YEWBwmWwkrtU8yNOTJD0srqt4aFc"
	sheet := manager.GetSpreadsheet(sheetId)
	fmt.Println("------Get sheet ", sheetId, "------")
	fmt.Println("------    Result ", sheet.SpreadsheetId)
}

// create
func TestCreateSpreadsheet(t *testing.T) {
	manager := NewSheetManager(jsonPath)
	sheet := manager.CreateSpreadsheet("Testing!")
	fmt.Println("------Created sheet------")
	fmt.Println("Sheet ID: ", sheet.SpreadsheetId)
	fmt.Println("Sheet Name: ", sheet.Properties.Title)
	fmt.Println("Sheet Timezone: ", sheet.Properties.TimeZone)
}

// list
func TestListSpreadsheet(t *testing.T) {
	manager := NewSheetManager(jsonPath)
	sheets := manager.ListSpreadsheets()
	for _, s := range sheets {
		sheet := manager.GetSpreadsheet(s)
		fmt.Println("------Listing sheet------")
		fmt.Println("Sheet ID: ", sheet.SpreadsheetId)
		fmt.Println("Sheet Name: ", sheet.Properties.Title)
		fmt.Println("Sheet Timezone: ", sheet.Properties.TimeZone)
	}
}

// Delete
func TestDeleteSpreadsheet(t *testing.T) {
	manager := NewSheetManager(jsonPath)
	sheet := manager.FindSpreadsheet("testdb")
	fmt.Println("------Found sheet------")
	fmt.Println("Sheet ID: ", sheet.SpreadsheetId)
	fmt.Println("Sheet Name: ", sheet.Properties.Title)
	fmt.Println("Sheet Timezone: ", sheet.Properties.TimeZone)
	sheetId := sheet.SpreadsheetId
	if manager.DeleteSpreadsheet(sheet.SpreadsheetId) {
		fmt.Println("------Deleted sheet ", sheetId, "------")
	}
}

// Find
func TestSpreadsheet005(t *testing.T) {
	manager := NewSheetManager(jsonPath)
	sheet := manager.FindSpreadsheet("Test First!")
	if sheet != nil {
		fmt.Println("------Listing sheet------")
		fmt.Println("Sheet ID: ", sheet.SpreadsheetId)
		fmt.Println("Sheet Name: ", sheet.Properties.Title)
		fmt.Println("Sheet Timezone: ", sheet.Properties.TimeZone)
	}
}

// Create DB
func TestSpreadsheet006(t *testing.T) {
	manager := NewSheetManager(jsonPath)
	sheet := manager.CreateDatabase("testdb")
	if sheet != nil {
		fmt.Println("------Listing sheet------")
		fmt.Println("Sheet ID: ", sheet.SpreadsheetId)
		fmt.Println("Sheet Name: ", sheet.Properties.Title)
		fmt.Println("Sheet Timezone: ", sheet.Properties.TimeZone)
	}
}

// Create table
func TestSpreadsheet007(t *testing.T) {
	manager := NewSheetManager(jsonPath)
	sheet := manager.FindDatabase("testdb")
	if sheet != nil {
		fmt.Println("------Listing sheet------")
		fmt.Println("Sheet ID: ", sheet.SpreadsheetId)
		fmt.Println("Sheet Name: ", sheet.Properties.Title)
		fmt.Println("Sheet Timezone: ", sheet.Properties.TimeZone)
	}
}
