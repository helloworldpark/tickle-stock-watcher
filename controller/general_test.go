package controller

import (
	"fmt"
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

func TestWatchStock(t *testing.T) {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()

	defer client.Close()

	client.RegisterStructFromRegisterables([]database.DBRegisterable{
		// structs.CoinInfo{},
		structs.StockPrice{},
		// structs.Invitation{},
		// structs.TradeCredential{},
		structs.User{},
		structs.UserStock{},
	})

	stocks := make(map[string]bool)
	for _, v := range structs.AllStrategies(client) {
		stocks[v.StockID] = true
	}
	for k := range stocks {
		fmt.Println("Stock: ", k)
	}
	fmt.Println(len(stocks))
}
