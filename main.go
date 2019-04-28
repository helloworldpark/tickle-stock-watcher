package main

import (
	"flag"

	"github.com/gin-gonic/gin"
	"github.com/helloworldpark/tickle-stock-watcher/controller"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/push"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

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

	// TelegramClient 초기화
	push.InitTelegram(*telegramPath)

	// General 생성
	general := controller.NewGeneral(client)
	general.Initialize()

	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.String(200, "Hello World!")
	})

	router.POST(push.URLTelegramUpdate(), push.OnTelegramUpdate(general))

	router.Run("127.0.0.1:5003")
}
