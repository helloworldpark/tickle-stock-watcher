package main

import (
	"flag"
	"fmt"
	"github.com/helloworldpark/tickle-stock-watcher/personnel"
	"strings"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/scheduler"

	"github.com/gin-gonic/gin"
	"github.com/helloworldpark/tickle-stock-watcher/analyser"
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/push"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

type General struct {
	priceWatcher *watcher.Watcher
	dateChecker  *watcher.DateChecker
	itemChecker  *watcher.StockItemChecker
	broker       *analyser.Broker
	users        map[string][]structs.User
	pushManager  *push.Manager
	dbClient     *database.DBClient
}

func NewGeneral(dbClient *database.DBClient) *General {
	g := General{
		priceWatcher: watcher.New(dbClient, time.Millisecond*500),
		dateChecker:  watcher.NewDateChecker(),
		itemChecker:  watcher.NewStockItemChecker(dbClient),
		broker:       analyser.NewBroker(dbClient),
		users:        make(map[string][]structs.User),
		pushManager:  push.NewManager(),
		dbClient:     dbClient,
	}
	return &g
}

type mainError struct {
	msg string
}

func (e mainError) Error() string {
	return fmt.Sprintf("[Main] %s", e.msg)
}

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
	g.pushManager.PushMessage(err.Error(), user)
}

var botOrders = map[string]order{
	"invite": newInviteOrder(),
	"join":   newJoinOrder(),
}

func runOrder(user structs.User, orders []string) error {
	if len(orders) == 0 {
		return mainError{msg: "Invalid order"}
	}
	lowerOrders := make([]string, len(orders))
	for i := range orders {
		lowerOrders[i] = strings.ToLower(orders[i])
	}
	action, ok := botOrders[lowerOrders[0]]
	if !ok {
		return mainError{msg: fmt.Sprintf("Cannot perform %s: don't know how to do", orders[0])}
	}
	return action.onAction(user, lowerOrders[1:])
}

func main() {
	defer logger.Close()

	credPath := flag.String("credential", "", "Credential for DB access")
	telegramPath := flag.String("telegram", "", "Telegram token for webhook")
	flag.Parse()

	if credPath == nil || *credPath == "" {
		logger.Panic("No -credential provided")
	}

	if telegramPath == nil || *telegramPath == "" {
		logger.Panic("No -telegram provided")
	}

	// DB Client 생성
	credential := database.LoadCredential(*credPath)
	client := database.CreateClient()
	client.Init(credential)
	client.Open()
	defer client.Close()

	// DB 테이블 초기화
	client.RegisterStructFromRegisterables([]database.DBRegisterable{
		structs.Stock{},
		structs.StockPrice{},
		structs.User{},
		structs.UserStock{},
		structs.WatchingStock{},
		structs.Invitation{},
	})

	// General 생성
	general := NewGeneral(client)

	// DateChecker 초기화
	general.dateChecker.UpdateHolidays(commons.Now().Year())

	// ItemChecker 초기화
	general.itemChecker.UpdateStocks()

	// TelegramClient 초기화
	push.InitTelegram(*telegramPath)

	// 유저 정보와 등록된 전략들을 바탕으로 PriceWatcher, Broker, User 현황 초기화
	userIndex := make(map[int64]structs.User)
	for _, u := range structs.AllUsers(client) {
		userIndex[u.UserID] = u
	}
	for _, v := range structs.AllStrategies(client) {
		stock, ok := general.itemChecker.StockFromID(v.StockID)
		if !ok {
			continue
		}
		_, ok = general.users[v.StockID]
		if !ok {
			general.users[v.StockID] = make([]structs.User, 0)
		}
		general.users[v.StockID] = append(general.users[v.StockID], userIndex[v.UserID])
		general.priceWatcher.Register(stock)
		// TODO: implement callback
		// general.broker.AddStrategy(v, callback)
	}

	// 명령어들 초기화
	botOrders["invite"].setAction(func(user structs.User, args []string) error {
		guestname := args[0]
		signature, invitation, err := personnel.Invite(user, guestname)
		if err != nil {
			logger.Error("%s", err.Error())
			return err
		}
		_, err = general.dbClient.Insert(invitation)
		if err != nil {
			logger.Error("%s", err.Error())
			return err
		}
		pushMessage := fmt.Sprintf("[Invite] Signature: \n%s", signature)
		general.pushManager.PushMessage(pushMessage, user)
		logger.Info("[Invite] Invitation signature created: %s", signature)
		return nil
	})
	botOrders["join"].setAction(func(user structs.User, args []string) error {
		username := args[0]
		signature := args[1]
		var invitation []structs.Invitation
		_, err := general.dbClient.Select(&invitation, "where Guestname=?", username)
		if err != nil {
			return err
		}
		if len(invitation) == 0 {
			return mainError{msg: fmt.Sprintf("No invitation issued for username %s", username)}
		}
		err = personnel.ValidateInvitation(invitation[0], signature)
		if err != nil {
			return err
		}

		user.Superuser = false

		_, err = general.dbClient.Insert(user)
		if err != nil {
			return err
		}

		general.dbClient.Delete(structs.Invitation{}, "where Guestname=?", username)
		general.pushManager.PushMessage("Congratulations! Press `help` and send to know how to use this bot.", user)

		return nil
	})

	// PriceWatcher는 주중, 장이 열리는 날이면 09시부터 감시 시작
	// PriceWatcher는 주중, 18시가 되면 감시 중단
	// PriceWatcher는 주중, 장이 열리는 날이면 06시부터 오늘로부터 이전 날까지의 가격 정보 수집
	// AnalyserBroker는 주중, 장이 열리는 날이면 08시에 과거 가격 정보를 업데이트받는다
	scheduler.ScheduleWeekdays("WatchPrice", watcher.OpeningTime(time.Time{}), func() {
		// 오늘 장날인지 확인
		isMarketOpen := general.dateChecker.IsHoliday(commons.Now())
		if !isMarketOpen {
			return
		}

		stocks := make(map[string]bool)
		for _, v := range structs.AllStrategies(client) {
			stocks[v.StockID] = true
		}
		for k := range stocks {
			provider := general.priceWatcher.StartWatchingStock(k)
			general.broker.FeedPrice(k, provider)
		}
	})
	scheduler.ScheduleWeekdays("StopWatchPrice", watcher.ClosingTime(time.Time{}), func() {
		general.priceWatcher.StopWatching()
	})
	scheduler.ScheduleWeekdays("CollectPrice", 6, func() {
		general.priceWatcher.Collect()
	})
	scheduler.ScheduleWeekdays("UpdatePriceBroker", 8, func() {
		general.broker.UpdatePastPrice()
	})

	// DateChecker는 매해 12월 29일 07시, 다음 해의 공휴일 정보를 갱신
	now := commons.Now()
	dec29 := time.Date(now.Year(), time.December, 29, 7, 0, 0, 0, commons.AsiaSeoul)
	ttl := dec29.Sub(now)
	scheduler.SchedulePeriodic("HolidayCheck", time.Hour*24*365, ttl, func() {
		year := commons.Now().Year()
		general.dateChecker.UpdateHolidays(year)
	})

	// ItemChecker는 매일 05시, 현재 거래 가능한 주식들을 업데이트
	scheduler.ScheduleEveryday("StockItemUpdate", 5, func() {
		general.itemChecker.UpdateStocks()
	})

	// 주기적으로 유저들에게 메세지를 보내고(현재 봇에 등록한 주식 종목들), 응답이 없으면 그 유저는 봇을 탈퇴한 것으로 간주하고 유저를 삭제한다

	// General은 다음와 같은 일들을 수행
	// 1. 파싱된 유저의 메세지를 처리
	// 2. 유저가 전략을 추가하면, 전략을 수정하거나, 삭제한다
	// 3. 종목번호에 따른 가격, 종목 정보를 보여준다
	// 4. 유저의 가입 처리를 한다

	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.String(200, "Hello World!")
	})

	router.POST(push.URLTelegramUpdate(), push.OnTelegramUpdate(general))

	router.Run("127.0.0.1:5003")
}
