package controller

import (
	"fmt"
	"github.com/helloworldpark/tickle-stock-watcher/analyser"
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/orders"
	"github.com/helloworldpark/tickle-stock-watcher/push"
	"github.com/helloworldpark/tickle-stock-watcher/scheduler"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
	"strings"
	"time"
)

type conError struct {
	msg string
}

func (e conError) Error() string {
	return fmt.Sprintf("[General] %s", e.msg)
}

var botOrders = map[string]orders.Order{
	"invite": orders.NewInviteOrder(),
	"join":   orders.NewJoinOrder(),
	"buy":    orders.NewBuyOrder(),
	"sell":   orders.NewSellOrder(),
}

func runOrder(user structs.User, orders []string) error {
	if len(orders) == 0 {
		return conError{msg: "Invalid order"}
	}
	lowerOrders := make([]string, len(orders))
	for i := range orders {
		lowerOrders[i] = strings.ToLower(orders[i])
	}
	action, ok := botOrders[lowerOrders[0]]
	if !ok {
		return conError{msg: fmt.Sprintf("Cannot perform %s: don't know how to do", orders[0])}
	}
	return action.OnAction(user, lowerOrders[1:])
}

// General is the main controller of this whole project
type General struct {
	priceWatcher *watcher.Watcher
	dateChecker  *watcher.DateChecker
	itemChecker  *watcher.StockItemChecker
	broker       *analyser.Broker
	pushManager  *push.Manager
	dbClient     *database.DBClient
}

// NewGeneral returns a new pointer to General, uninitialized
func NewGeneral(dbClient *database.DBClient) *General {
	g := General{
		priceWatcher: watcher.New(dbClient, time.Millisecond*500),
		dateChecker:  watcher.NewDateChecker(),
		itemChecker:  watcher.NewStockItemChecker(dbClient),
		broker:       analyser.NewBroker(dbClient),
		pushManager:  push.NewManager(),
		dbClient:     dbClient,
	}
	return &g
}

// OnWebhook interface push.WebhookHandler
func (g *General) OnWebhook(token int64, msg string) {
	user, err := structs.UserFromID(g.dbClient, token)
	emptyUser := structs.User{}
	if user == emptyUser {
		user.UserID = token
	}
	orders := strings.Split(msg, " ")
	err = runOrder(user, orders)
	if err != nil {
		g.onError(user, err)
	}
}

func (g *General) onError(user structs.User, err error) {
	g.pushManager.PushMessage(err.Error(), user.UserID)
}

// Initialize initializes General
func (g *General) Initialize() {
	// DateChecker 초기화
	g.dateChecker.UpdateHolidays(commons.Now().Year())

	// ItemChecker 초기화
	g.itemChecker.UpdateStocks()

	// 유저 정보와 등록된 전략들을 바탕으로 PriceWatcher, Broker, User 현황 초기화
	userIndex := make(map[int64]structs.User)
	for _, u := range structs.AllUsers(g.dbClient) {
		userIndex[u.UserID] = u
	}
	for _, v := range structs.AllStrategies(g.dbClient) {
		stock, ok := g.itemChecker.StockFromID(v.StockID)
		if !ok {
			continue
		}
		g.priceWatcher.Register(stock)
		g.broker.AddStrategy(v, g.onStrategyEvent)
	}

	// 명령어들 초기화
	botOrders["join"].SetAction(orders.Join(g, func(user structs.User) {
		g.pushManager.PushMessage("Congratulations! Press `help` and send to know how to use this bot.", user.UserID)
	}))
	botOrders["invite"].SetAction(orders.Invite(g, func(user structs.User, signature string) {
		pushMessage := fmt.Sprintf("[Invite] Send this signature: \n%s", signature)
		g.pushManager.PushMessage(pushMessage, user.UserID)
	}))
	tradeOnSuccess := func(user structs.User, orderside int, stockname, stockid, strategy string) {
		side := "BUY"
		if orderside == commons.SELL {
			side = "SELL"
		}
		msgFormat := "[%s] Strategy for %s(%s) registered: %s"
		msg := fmt.Sprintf(msgFormat, side, stockname, stockid, strategy)
		g.pushManager.PushMessage(msg, user.UserID)
	}
	botOrders["buy"].SetAction(orders.Trade(commons.BUY, g, g, g, g.onStrategyEvent, tradeOnSuccess))
	botOrders["sell"].SetAction(orders.Trade(commons.SELL, g, g, g, g.onStrategyEvent, tradeOnSuccess))

	// PriceWatcher는 주중, 장이 열리는 날이면 09시부터 감시 시작
	// PriceWatcher는 주중, 18시가 되면 감시 중단
	// PriceWatcher는 주중, 장이 열리는 날이면 06시부터 오늘로부터 이전 날까지의 가격 정보 수집
	// AnalyserBroker는 주중, 장이 열리는 날이면 08시에 과거 가격 정보를 업데이트받는다
	scheduler.ScheduleWeekdays("WatchPrice", watcher.OpeningTime(time.Time{}), func() {
		// 오늘 장날인지 확인
		isMarketOpen := g.dateChecker.IsHoliday(commons.Now())
		if !isMarketOpen {
			return
		}

		stocks := make(map[string]bool)
		for _, v := range structs.AllStrategies(g.dbClient) {
			stocks[v.StockID] = true
		}
		for k := range stocks {
			provider := g.priceWatcher.StartWatchingStock(k)
			g.broker.FeedPrice(k, provider)
		}
	})
	scheduler.ScheduleWeekdays("StopWatchPrice", watcher.ClosingTime(time.Time{}), func() {
		g.priceWatcher.StopWatching()
	})
	scheduler.ScheduleWeekdays("CollectPrice", 6, func() {
		g.priceWatcher.Collect()
	})
	scheduler.ScheduleWeekdays("UpdatePriceBroker", 8, func() {
		g.broker.UpdatePastPrice()
	})

	// DateChecker는 매해 12월 29일 07시, 다음 해의 공휴일 정보를 갱신
	now := commons.Now()
	dec29 := time.Date(now.Year(), time.December, 29, 7, 0, 0, 0, commons.AsiaSeoul)
	ttl := dec29.Sub(now)
	scheduler.SchedulePeriodic("HolidayCheck", time.Hour*24*365, ttl, func() {
		year := commons.Now().Year()
		g.dateChecker.UpdateHolidays(year)
	})

	// ItemChecker는 매일 05시, 현재 거래 가능한 주식들을 업데이트
	scheduler.ScheduleEveryday("StockItemUpdate", 5, func() {
		g.itemChecker.UpdateStocks()
	})

	// 주기적으로 유저들에게 메세지를 보내고(현재 봇에 등록한 주식 종목들), 응답이 없으면 그 유저는 봇을 탈퇴한 것으로 간주하고 유저를 삭제한다

	// General은 다음와 같은 일들을 수행
	// 1. 파싱된 유저의 메세지를 처리
	// 2. 유저가 전략을 추가하면, 전략을 수정하거나, 삭제한다
	// 3. 종목번호에 따른 가격, 종목 정보를 보여준다
	// 4. 유저의 가입 처리를 한다
}

//
func (g *General) onStrategyEvent(price float64, stockid string, orderSide int, userid int64, repeat bool) {
	// Notify to user
	msgFormat := "[%s] 주식종목 %s의 가격이 등록하신 조건에 충족되었습니다: 현재가 %d원"
	buyOrSell := "매수"
	if orderSide == commons.SELL {
		buyOrSell = "매도"
	}
	stock, _ := g.itemChecker.StockFromID(stockid)
	msg := fmt.Sprintf(msgFormat, buyOrSell, stock.Name, int(price))
	g.pushManager.PushMessage(msg, userid)

	// Handle Repeat
	if repeat {
		return
	}
	// Delete Strategy
	g.broker.DeleteStrategy(structs.User{UserID: userid}, stockid, orderSide)
	// Withdraw Watcher
	g.priceWatcher.Withdraw(stock)
}

// AccessDB interface database.DBAccess
func (g *General) AccessDB() *database.DBClient {
	return g.dbClient
}

// AccessStockItem interface watcher.StockAccess
func (g *General) AccessStockItem(stockid string) (structs.Stock, bool) {
	return g.itemChecker.StockFromID(stockid)
}

// AccessBroker interface analyser.BrokerAccess
func (g *General) AccessBroker() *analyser.Broker {
	return g.broker
}

// AccessWatcher interface analyser.WatcherAccess
func (g *General) AccessWatcher() *watcher.Watcher {
	return g.priceWatcher
}
