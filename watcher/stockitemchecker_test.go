package watcher

import (
	"fmt"
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

func TestDownload(t *testing.T) {
	result := downloadStockSymbols(structs.KOSDAQ)
	for _, v := range result {
		fmt.Println(v)
	}
}

func TestUpdate(t *testing.T) {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()

	defer client.Close()

	client.RegisterStructFromRegisterables([]database.DBRegisterable{structs.Stock{}})

	checker := NewStockItemChecker(client)

	stock, ok := checker.StockFromName("삼성전자")
	fmt.Println(ok, stock)
	stock, ok = checker.StockFromID("005930")
	fmt.Println(ok, stock)
	stock, ok = checker.StockFromName("CJ CGV")
	fmt.Println(ok, stock)
	stock, ok = checker.StockFromName("CJCGV\n")
	fmt.Println(ok, stock)
}
