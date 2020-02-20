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
func TestCreateTable(t *testing.T) {
	manager := NewSheetManager(jsonPath)
	sheet := manager.FindDatabase("testdb")
	if sheet != nil {
		fmt.Println("------This database------")
		fmt.Println("Database ID:       ", sheet.SpreadsheetId)
		fmt.Println("Database Name:     ", sheet.Properties.Title)
		fmt.Println("Database Timezone: ", sheet.Properties.TimeZone)
		fmt.Println("------Creating table------")
		manager.CreateTable(sheet, "MAMA")
		manager.CreateTable(sheet, "NENE")
		tables := manager.GetTableList(sheet)
		for i := range tables {
			fmt.Printf("Table[%d] Name: %s\n", i, tables[i].Properties.Title)
		}
	}
}

// View table
func TestViewTable(t *testing.T) {
	manager := NewSheetManager(jsonPath)
	sheet := manager.FindDatabase("testdb")
	if sheet != nil {
		fmt.Println("------This database------")
		fmt.Println("Database ID:       ", sheet.SpreadsheetId)
		fmt.Println("Database Name:     ", sheet.Properties.Title)
		fmt.Println("Database Timezone: ", sheet.Properties.TimeZone)
		fmt.Println("------Viewing table------")
		tables := manager.GetTableList(sheet)
		for i := range tables {
			fmt.Printf("Table[%d] Name: %s\n", i, tables[i].Properties.Title)
		}
		table := manager.GetTable(sheet, "NENE")
		fmt.Printf("Table = %s\n", table.Properties.Title)
	}
}

// Delete table
func TestDeleteTable(t *testing.T) {
	manager := NewSheetManager(jsonPath)
	sheet := manager.FindDatabase("testdb")
	if sheet != nil {
		fmt.Println("------This database------")
		fmt.Println("Database ID:       ", sheet.SpreadsheetId)
		fmt.Println("Database Name:     ", sheet.Properties.Title)
		fmt.Println("Database Timezone: ", sheet.Properties.TimeZone)
		fmt.Println("------Viewing table------")

		deleted := manager.DeleteTable(sheet, "Sheet1")
		if deleted {
			fmt.Println("Deleted Table = Sheet1")
		} else {
			fmt.Println("Failed delete Table = Sheet1")
		}

		tables := manager.GetTableList(sheet)
		for i := range tables {
			fmt.Printf("Table[%d] Name: %s\n", i, tables[i].Properties.Title)
		}
	}
}
