package controller

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/analyser"
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/orders"
	"github.com/helloworldpark/tickle-stock-watcher/push"
	"github.com/helloworldpark/tickle-stock-watcher/scheduler"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

var botOrders = map[string]orders.Order{
	"help":           orders.NewHelpOrder(),
	"invite":         orders.NewInviteOrder(),
	"join":           orders.NewJoinOrder(),
	"buy":            orders.NewBuyOrder(),
	"sell":           orders.NewSellOrder(),
	"strategy":       orders.NewStrategyOrder(),
	"stock":          orders.NewStockOrder(),
	"delete":         orders.NewDeleteOrder(),
	"watcher":        orders.NewWatcherDescriptionOrder(),
	"analyser":       orders.NewBrokerDescriptionOrder(),
	"holiday":        orders.NewDateCheckerDescriptionOrder(),
	"terminate":      orders.NewTerminationOrder(),
	"prospect":       orders.NewProspectsOrder(),
	"appendProspect": orders.NewAppendProspectOrder(),
}
var newError = commons.NewTaggedError("Controller")

// General is the main controller of this whole project
// General은 다음와 같은 일들을 수행
// 1. 파싱된 유저의 메세지를 처리
// 2. 유저가 전략을 추가하면, 전략을 수정하거나, 삭제한다
// 3. 종목번호에 따른 가격, 종목 정보를 보여준다
// 4. 유저의 가입 처리를 한다
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
		priceWatcher: watcher.New(dbClient, 30*time.Second),
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
	logger.Info("[Controller] User: %d Message: %s", token, msg)
	user, err := structs.UserFromID(g.dbClient, token)
	emptyUser := structs.User{}
	isGuest := false
	if user == emptyUser {
		user.UserID = token
		isGuest = true
	}
	args := strings.Split(msg, " ")
	preAsync := func() {
		g.pushManager.PushMessage("명령 접수. 대기하라.", token)
	}
	onAsync := func(e error) {
		if e != nil {
			g.onError(user, e)
		}
	}
	err = g.runOrder(user, isGuest, args, preAsync, onAsync)
	if err != nil {
		g.onError(user, err)
	}
}

func (g *General) runOrder(user structs.User, isGuest bool, orders []string, preAsync func(), onAsync func(err error)) error {
	if len(orders) == 0 {
		return newError("공허한 명령")
	}
	lowerOrders := make([]string, len(orders))
	for i := range orders {
		lowerOrders[i] = strings.ToLower(orders[i])
	}
	action, ok := botOrders[lowerOrders[0]]
	if !ok {
		return newError(fmt.Sprintf("수행 불가 %s: 모른다 어떻게 하는지", orders[0]))
	}
	if !action.IsPublic() && isGuest {
		return newError("누구냐 너는 거부한다 너의 명령")
	}
	if action.IsAsync() {
		preAsync()
		commons.InvokeGoroutine("General_runOrder_"+orders[0], func() {
			onAsync(action.OnAction(user, lowerOrders[1:]))
		})
		return nil
	}
	return action.OnAction(user, lowerOrders[1:])
}

func (g *General) onError(user structs.User, err error) {
	logger.Error(err.Error())
	g.pushManager.PushMessage(err.Error(), user.UserID)
}

// Initialize initializes General
func (g *General) Initialize() {
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
		shouldRetainWatcher, err := g.broker.AddStrategy(v, g.onStrategyEvent, false)
		if err == nil {
			logger.Info("[Controller] Added strategy for stock %s", v.StockID)
			if shouldRetainWatcher {
				g.priceWatcher.Register(stock)
			}
		} else {
			logger.Error(err.Error())
		}
	}

	// 명령어들 초기화
	botOrders["help"].SetAction(func(user structs.User, s []string) error {
		msg := "참고해라 닝겐\nhttps://github.com/helloworldpark/tickle-stock-watcher/wiki/BotOrders"
		g.pushManager.PushMessage(msg, user.UserID)
		return nil
	})
	botOrders["/start"] = botOrders["help"]
	botOrders["도움"] = botOrders["help"]
	botOrders["join"].SetAction(orders.Join(g, func(user structs.User) {
		g.pushManager.PushMessage("축하! `help` 또는 `도움` 치고 깨우쳐라 사용법 이 봇", user.UserID)
	}))
	botOrders["가입"] = botOrders["join"]
	botOrders["invite"].SetAction(orders.Invite(g, func(user structs.User, signature string) {
		pushMessage := fmt.Sprintf("[초대] 보내라 이 서명\n%s", signature)
		g.pushManager.PushMessage(pushMessage, user.UserID)
	}))
	botOrders["invite"] = botOrders["초대"]
	tradeOnSuccess := func(user structs.User, orderside int, stockname, stockid, strategy string) {
		side := []string{"사다", "팔다"}[orderside]
		msgFormat := "[%s] 종목 %s(%s)의 거래 전략 등록되다: %s"
		msg := fmt.Sprintf(msgFormat, side, stockname, stockid, strategy)
		g.pushManager.PushMessage(msg, user.UserID)
	}
	botOrders["buy"].SetAction(orders.Trade(commons.BUY, g, g, g, g.onStrategyEvent, tradeOnSuccess))
	botOrders["산다"] = botOrders["buy"]
	botOrders["사다"] = botOrders["buy"]
	botOrders["sell"].SetAction(orders.Trade(commons.SELL, g, g, g, g.onStrategyEvent, tradeOnSuccess))
	botOrders["팔다"] = botOrders["sell"]
	botOrders["판다"] = botOrders["sell"]
	botOrders["strategy"].SetAction(orders.Strategy(g, func(user structs.User, strategies []structs.UserStock) {
		side := []string{"사다", "팔다"}
		buffer := bytes.Buffer{}
		buffer.WriteString("전략: \n")
		sort.Slice(strategies, func(i, j int) bool {
			return strategies[i].OrderSide < strategies[j].OrderSide
		})
		for i := range strategies {
			stock, ok := g.itemChecker.StockFromID(strategies[i].StockID)
			buffer.WriteString("[")
			buffer.WriteString(side[strategies[i].OrderSide])
			buffer.WriteString("] ")
			if ok {
				buffer.WriteString(stock.Name)
				buffer.WriteString("(")
				buffer.WriteString(strategies[i].StockID)
				buffer.WriteString(")")
				if strategies[i].Repeat {
					buffer.WriteString("(반복)")
				}
				buffer.WriteString(": ")
				buffer.WriteString(strategies[i].Strategy)
			} else {
				buffer.WriteString("거래정지")
				buffer.WriteString("(")
				buffer.WriteString(strategies[i].StockID)
				buffer.WriteString(")")
			}
			buffer.WriteString("\n")
		}
		g.pushManager.PushMessage(buffer.String(), user.UserID)
	}))
	botOrders["전략"] = botOrders["strategy"]
	botOrders["stock"].SetAction(orders.QueryStockByName(g, func(user structs.User, stock structs.Stock) {
		buffer := bytes.Buffer{}
		buffer.WriteString("이름: ")
		buffer.WriteString(stock.Name)
		buffer.WriteString("\n")
		buffer.WriteString("거래소: ")
		buffer.WriteString(string(stock.MarketType))
		buffer.WriteString("\n")
		buffer.WriteString("종목번호: ")
		buffer.WriteString(stock.StockID)

		g.pushManager.PushMessage(buffer.String(), user.UserID)
	}))
	botOrders["주식"] = botOrders["stock"]
	botOrders["delete"].SetAction(orders.DeleteOrder(g, g, g, func(user structs.User, stockname, stockid string) {
		msg := fmt.Sprintf("삭제 종목 %s(%s)의 거래 전략", stockname, stockid)
		g.pushManager.PushMessage(msg, user.UserID)
	}))
	botOrders["삭제"] = botOrders["delete"]

	// Watcher 현황
	botOrders["watcher"].SetAction(orders.WatcherDescription(g, func(user structs.User, desc string) {
		g.pushManager.PushMessage(desc, user.UserID)
	}))

	// Analyser 현황
	botOrders["analyser"].SetAction(func(user structs.User, args []string) error {
		if user.Superuser {
			desc := g.AccessBroker().Description()
			g.pushManager.PushMessage(desc, user.UserID)
			return nil
		}
		return newError("Only superuser can order this")
	})
	botOrders["broker"] = botOrders["analyser"]
	botOrders["analyserbroker"] = botOrders["analyser"]

	// Holiday 현황
	botOrders["holiday"].SetAction(orders.DateCheckerDescription(g.dateChecker, func(user structs.User, desc string) {
		g.pushManager.PushMessage(desc, user.UserID)
	}))
	botOrders["holidays"] = botOrders["holiday"]

	// Terminate
	botOrders["terminate"].SetAction(func(user structs.User, args []string) error {
		if !user.Superuser {
			return newError("Only superuser can order this")
		}
		logger.Panic("Reboot")
		return nil
	})
	botOrders["terminator"] = botOrders["terminate"]
	botOrders["reboot"] = botOrders["terminate"]
	botOrders["restart"] = botOrders["terminate"]

	// Prospect
	botOrders["prospect"].SetAction(func(user structs.User, args []string) error {
		if !user.Superuser {
			return newError("Only superuser can order this")
		}
		analyser.FindProspects(g.dbClient, g.itemChecker, func(msg, savePath string) {
			if len(savePath) > 0 {
				g.pushManager.PushPhoto(msg, savePath, user.UserID)
			} else {
				g.pushManager.PushMessage(msg, user.UserID)
			}
		})
		return nil
	})
	botOrders["prospects"] = botOrders["prospect"]
	botOrders["scout"] = botOrders["prospect"]
	botOrders["scouter"] = botOrders["prospect"]
	botOrders["scouters"] = botOrders["prospect"]

	// appendProspect
	botOrders["appendProspect"].SetAction(func(user structs.User, args []string) error {
		prospects, now := analyser.ActiveProspects(g.dbClient, g.itemChecker)
		if len(prospects) == 0 {
			return newError(fmt.Sprintf("No prospects today(%v)", now))
		}
		f := orders.Trade(commons.BUY, g, g, g, g.onStrategyEvent, tradeOnSuccess)
		for stockID := range prospects {
			f(user, []string{stockID, "macd(12,26)>0&&zero(macdhist(12,26,9),1,7)==1&&mflow(14)<80"})
		}
		return nil
	})
	botOrders["appendProspects"] = botOrders["appendProspect"]

	// ItemChecker는 매일 05시, 현재 거래 가능한 주식들을 업데이트
	// AnalyserBroker는 주중, 장이 열리는 날이면 08시에 과거 가격 정보를 업데이트받는다
	// PriceWatcher는 주중, 장이 열리는 날이면 09시부터 감시 시작
	// PriceWatcher는 주중, 15시 30분이 되면 감시 중단
	// PriceWatcher는 주중, 장이 열리는 날이면 18시 30분부터 오늘로부터 이전의 가격 정보 수집
	scheduler.ScheduleEveryday("StockItemUpdate", 5, func() {
		g.itemChecker.UpdateStocks()
	})
	scheduler.ScheduleWeekdays("UpdatePriceBroker", 8, func() {
		g.broker.UpdatePastPrice()
	})
	watchPrice := func() {
		// 오늘 장날인지 확인
		isMarketClosed := g.dateChecker.IsHoliday(commons.Now())
		if isMarketClosed {
			logger.Warn("[Controller] Holiday: %s", commons.Now().String())
			return
		}

		// 중복될 수 있어서 주식들의 집합을 구한 후에 감시하도록 처리
		stocks := make(map[string]bool)
		for _, v := range structs.AllStrategies(g.dbClient) {
			stocks[v.StockID] = true
		}
		logger.Info("[Controller] Stock Set = %+v", stocks)
		randomGen := rand.New(rand.NewSource(time.Now().UnixNano()))
		for k := range stocks {
			if g.broker.CanFeedPrice(k) {
				provider := g.priceWatcher.StartWatchingStock(k)
				g.broker.FeedPrice(k, provider)
				sleepTime := randomGen.Float64()
				duration := time.Duration(sleepTime * float64(time.Second))
				time.Sleep(duration)
			}
		}
	}
	now := commons.Now()
	nowHour := float64(now.Hour()) + float64(now.Minute())/60
	if 9 < nowHour && nowHour < 15.5 {
		commons.InvokeGoroutine("controller_General_Initialize_WatchPrice_daily", watchPrice)
	}
	scheduler.ScheduleWeekdays("WatchPrice", 9, watchPrice)
	scheduler.ScheduleWeekdays("StopWatchPrice", 15.5, func() {
		g.broker.StopFeedingPrice()
		g.priceWatcher.StopWatching()
	})
	scheduler.ScheduleWeekdays("CollectPrice", 18.5, func() {
		g.priceWatcher.Collect()
	})
	findProspect := func() {
		users := structs.AllUsers(g.dbClient)
		analyser.FindProspects(g.dbClient, g.itemChecker, func(msg, savePath string) {
			for _, u := range users {
				if len(savePath) > 0 {
					g.pushManager.PushPhoto(msg, savePath, u.UserID)
				} else {
					g.pushManager.PushMessage(msg, u.UserID)
				}
			}
		})
	}
	scheduler.ScheduleEveryday("FindProspects", 20, findProspect)

	// DateChecker는 매해 12월 29일 07시, 다음 해의 공휴일 정보를 갱신
	now = commons.Now()
	dec29 := time.Date(now.Year(), time.December, 29, 7, 0, 0, 0, commons.AsiaSeoul)
	ttl := dec29.Sub(now)
	scheduler.SchedulePeriodic("HolidayCheck", time.Hour*24*365, ttl, func() {
		g.dateChecker.UpdateHolidays(commons.Now().Year())
	})
	logger.Info("[Controller] Initialized Controller")

	var superuser []structs.User
	g.dbClient.Select(&superuser, "where Superuser=?", true)
	if len(superuser) == 1 {
		g.pushManager.PushMessage("ticklestock 시작", superuser[0].UserID)
	}
}

// onStrategyEvent callback to be called when the users' strategies are fulfilled
func (g *General) onStrategyEvent(price structs.StockPrice, orderSide int, userid int64, repeat bool) {
	// Notify to user
	msgFormat := "[%s] %4d년 %d월 %d일 %02d시 %02d분 %02d초\n%s의 가격, 전략에 부합: 현재가 %d원"
	side := []string{"사다", "팔다"}[orderSide]
	stock, _ := g.itemChecker.StockFromID(price.StockID)
	currentTime := commons.Unix(price.Timestamp)
	y, m, d := currentTime.Date()
	h, i, s := currentTime.Clock()
	msg := fmt.Sprintf(msgFormat,
		side,
		y, m, d, h, i, s,
		stock.Name, int(price.Close))
	g.pushManager.PushMessage(msg, userid)

	// Handle Repeat
	if repeat {
		return
	}
	// Delete Strategy
	err := g.broker.DeleteStrategy(structs.User{UserID: userid}, price.StockID, orderSide)
	if err == nil {
		logger.Info("[Controller] Deleted strategy: %d, %s, %d", userid, stock, orderSide)
	} else {
		logger.Error("[Controller] %s", err.Error())
	}
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

// AccessStockItemByName interface watcher.StockAccess
func (g *General) AccessStockItemByName(stockname string) (structs.Stock, bool) {
	return g.itemChecker.StockFromName(stockname)
}

// AccessBroker interface analyser.BrokerAccess
func (g *General) AccessBroker() *analyser.Broker {
	return g.broker
}

// AccessWatcher interface analyser.WatcherAccess
func (g *General) AccessWatcher() *watcher.Watcher {
	return g.priceWatcher
}
