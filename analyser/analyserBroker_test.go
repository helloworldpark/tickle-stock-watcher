package analyser_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/analyser"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

func TestAnalyserBroker(t *testing.T) {
	credential := database.LoadCredential("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	client := database.CreateClient()
	client.Init(credential)
	client.Open()

	defer client.Close()

	client.RegisterStructFromRegisterables([]database.DBRegisterable{
		structs.Stock{},
		structs.StockPrice{},
		structs.User{},
		structs.UserStock{},
		structs.WatchingStock{},
	})

	g := mockGeneral{
		priceWatcher: watcher.New(client, time.Millisecond*500),
		broker:       analyser.NewBroker(client),
	}

	userIndex := make(map[int64]structs.User)
	for _, u := range structs.AllUsers(client) {
		userIndex[u.UserID] = u
	}
	stocks := make(map[string]bool)
	for _, v := range structs.AllStrategies(client) {
		stock := structs.Stock{StockID: v.StockID}
		g.priceWatcher.Register(stock)
		g.broker.AddStrategy(v, g.onStrategyEvent, false)

		stocks[v.StockID] = true
	}

	for k := range stocks {
		provider := g.priceWatcher.StartWatchingStock(k)
		g.broker.FeedPrice(k, provider)
	}

	timer := time.NewTimer(3 * time.Second)
	<-timer.C
	g.priceWatcher.StopWatching()
	timer = time.NewTimer(3 * time.Second)
	<-timer.C
	fmt.Println("Test Finished")
}

type mockGeneral struct {
	priceWatcher *watcher.Watcher
	broker       *analyser.Broker
}

func (g *mockGeneral) onStrategyEvent(currentTime time.Time, price float64, stockid string, orderSide int, userid int64, repeat bool) {
	// Notify to user
	msgFormat := "[%s] %4d년 %d월 %d일 %02d시 %02d분 %02d초\n%s의 가격이 등록하신 조건에 충족되었습니다: 현재가 %d원"
	buyOrSell := "매수"
	if orderSide == commons.SELL {
		buyOrSell = "매도"
	}
	msg := fmt.Sprintf(msgFormat,
		buyOrSell,
		currentTime.Year(), currentTime.Month(), currentTime.Day(), currentTime.Hour(), currentTime.Minute(), currentTime.Second(),
		stockid, int(price))

	fmt.Println(msg)

	// Delete Strategy
	g.broker.DeleteStrategy(structs.User{UserID: userid}, stockid, orderSide)
	// Withdraw Watcher
	g.priceWatcher.Withdraw(structs.Stock{StockID: stockid})
}
