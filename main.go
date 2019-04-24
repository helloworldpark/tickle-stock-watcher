package main

import (
	"flag"
	"fmt"
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

type RequestHandler interface {
	HandleCommand(string)
}

type General struct {
	priceWatcher *watcher.Watcher
	dateChecker  *watcher.DateChecker
	itemChecker  *watcher.StockItemChecker
	broker       *analyser.Broker
}

func NewGeneral(dbClient *database.DBClient) *General {

	g := General{
		priceWatcher: watcher.New(dbClient, time.Millisecond*500),
		dateChecker:  watcher.NewDateChecker(),
		itemChecker:  watcher.NewStockItemChecker(dbClient),
		broker:       analyser.NewBroker(dbClient),
	}
	return &g
}

func allStrategies(client *database.DBClient) []structs.UserStock {
	var userStrategyList []structs.UserStock
	_, err := client.Select(&userStrategyList, "where true")
	if err != nil {
		logger.Panic("Error while selecting user strategies: %s", err.Error())
	}
	return userStrategyList
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
		structs.Stock{}, structs.StockPrice{}, structs.User{}, structs.UserStock{}, structs.WatchingStock{},
	})

	// General 생성
	general := NewGeneral(client)

	// DateChecker 초기화
	general.dateChecker.UpdateHolidays(commons.Now().Year())

	// ItemChecker 초기화
	general.itemChecker.UpdateStocks()

	// TelegramClient 초기화
	push.InitTelegram(*telegramPath)

	// 유저 정보와 등록된 전략들을 바탕으로 PriceWatcher, Broker 초기화
	for _, v := range allStrategies(client) {
		stock, ok := general.itemChecker.StockFromID(v.StockID)
		if !ok {
			continue
		}
		general.priceWatcher.Register(stock)
		// TODO: implement callback
		// general.broker.AddStrategy(v, callback)
	}

	// PriceWatcher는 주중, 장이 열리는 날이면 09시부터 감시 시작
	// PriceWatcher는 주중, 18시가 되면 감시 중단
	// PriceWatcher는 주중, 장이 열리는 날이면 06시부터 오늘로부터 이전 날까지의 가격 정보 수집
	scheduler.ScheduleWeekdays("WatchPrice", watcher.OpeningTime(time.Time{}), func() {
		// 오늘 장날인지 확인
		isMarketOpen := general.dateChecker.IsHoliday(commons.Now())
		if !isMarketOpen {
			return
		}
		for _, v := range allStrategies(client) {
			provider := general.priceWatcher.StartWatchingStock(v.StockID)
			general.broker.FeedPrice(v.StockID, provider)
		}
	})
	scheduler.ScheduleWeekdays("StopWatchPrice", watcher.ClosingTime(time.Time{}), func() {
		general.priceWatcher.StopWatching()
	})
	scheduler.ScheduleEveryday("CollectPrice", 6, func() {
		general.priceWatcher.Collect()
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

	router.POST(fmt.Sprintf("/api/telegram/%s", push.GetTelegramTokenForURL()), func(c *gin.Context) {
		var v interface{}
		err := c.BindJSON(&v)
		if err == nil {
			logger.Info("[Main] Telegram Update: %v", v)
		} else {
			logger.Error("[Main] Telegram Update Error: %s", err.Error())
		}
		c.String(200, "")
	})

	router.Run("127.0.0.1:5003")
}
