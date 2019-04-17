package main

import (
	"github.com/gin-gonic/gin"
	"github.com/helloworldpark/tickle-stock-watcher/analyser"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

type RequestHandler interface {
	HandleCommand(string)
}

type General struct {
	priceWatcher *watcher.Watcher
	dateChecker  *watcher.DateChecker
	itemChecker  *watcher.StockItemChecker
	broker       *analyser.AnalyserBroker
}

func NewGeneral(dbClient *database.DBClient) *General {

	g := General{
		priceWatcher: watcher.New(dbClient),
		dateChecker:  watcher.NewDateChecker(),
		itemChecker:  watcher.NewStockItemChecker(dbClient),
		broker:       analyser.NewAnalyserBroker(dbClient),
	}
	return &g
}

func main() {

	// DB Client 생성

	// General 생성

	// DateChecker 초기화

	// ItemChecker 초기화

	// 유저 정보와 등록된 전략들을 바탕으로 PriceWatcher, AnalyserBroker 초기화

	// PriceWatcher는 주중, 장이 열리는 날이면 09시부터 감시 시작
	// PriceWatcher는 주중, 18시가 되면 감시 중단
	// PriceWatcher는 주중, 장이 열리는 날이면 06시부터 오늘로부터 이전 날까지의 가격 정보 수집

	// DateChecker는 매해 12월 29일 07시, 다음 해의 공휴일 정보를 갱신

	// ItemChecker는 매일 05시, 현재 거래 가능한 주식들을 업데이트

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

	router.Run("127.0.0.1:5003")
}
