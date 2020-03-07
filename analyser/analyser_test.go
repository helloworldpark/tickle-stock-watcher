package analyser

import (
	"fmt"
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

func prepareDBClient() *database.DBClient {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()
	client.RegisterStructFromRegisterables([]database.DBRegisterable{
		// structs.CoinInfo{},
		structs.Stock{},
		structs.StockPrice{},
		// structs.Invitation{},
		// structs.TradeCredential{},
		// structs.User{},
		// structs.UserStrategy{},
	})
	return client
}

func TestAppendPastPrice(t *testing.T) {
	dbClient := prepareDBClient()
	defer dbClient.Close()

	info := structs.Stock{Name: "Korean Air", StockID: "003490", MarketType: structs.KOSPI}

	analyser := NewAnalyser(info.StockID)
	analyser.NeedPriceFrom()

	timestampFrom := analyser.NeedPriceFrom()

	var prices []structs.StockPrice
	_, err := dbClient.Select(&prices,
		"where StockID=? and Timestamp>=? order by Timestamp",
		info.StockID, timestampFrom)
	if err != nil {
		t.Fatal(err)
	}
	for i := range prices {
		analyser.AppendPastPrice(prices[i])
		fmt.Printf("Price[%d]:%v\n%v\n", i, commons.Unix(prices[i].Timestamp), analyser.timeSeries.LastCandle())
	}
}
