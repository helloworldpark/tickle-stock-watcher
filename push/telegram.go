package push

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

var telegramToken = ""
var telegramClient = &http.Client{Timeout: time.Second * 30}

type telegramUser struct {
	id           int64
	isBot        bool   `json:"is_bot"`
	firstName    string `json:"first_name"`
	lastName     string `json:"last_name"`
	username     string
	languageCode string `json:"language_code"`
}

type telegramChat struct {
	id       int64
	chatType string `json:"type"`
	title    string
	username string
}

type telegramMessage struct {
	messageID int64 `json:"message_id"`
	from      telegramUser
	date      int64
	chat      telegramChat
	text      string
}

type telegramUpdate struct {
	updateID int64 `json:"update_id"`
	message  telegramMessage
}

func GetTelegramToken() string {
	return telegramToken
}

func GetTelegramTokenForURL() string {
	return strings.Split(telegramToken, ":")[0]
}

func InitTelegram(filePath string) {
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		logger.Panic("%v", err)
	}

	var token struct{ token string }
	if err := json.Unmarshal(raw, &token); err != nil {
		logger.Panic("%v", err)
	}

	telegramToken = token.token
	log.Print("Telegram Token: ", telegramToken)
}

func SetTelegramWebhook() {
	if telegramToken == "" {
		logger.Panic("[Push] Telegram client not initialized")
	}
	url := "https://api.telegram.org/bot" + telegramToken + "/" + "setWebhook"
	body := map[string]interface{}{
		"url":             "https://stock.ticklemeta.kr/api/telegram/" + GetTelegramTokenForURL(),
		"allowed_updates": []string{"message"},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		logger.Panic("[Push] Your JSON is wrong: %s", err.Error())
	}
	bodyBuffer := bytes.NewBuffer(bodyBytes)
	resp, err := telegramClient.Post(url, "application/json", bodyBuffer)
	if err != nil {
		logger.Error("[Push] Error while sending request to Telegram setWebhook: %s", err.Error())
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode/100 == 2 {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			var tmp interface{}
			json.Unmarshal(respBody, &tmp)
			result := tmp.(map[string]interface{})
			if result["ok"].(bool) {
				logger.Info("[Push] Telegram setWebhook Success")
			} else {
				logger.Info("[Push] Telegram setWebhook Failed: %s", result["description"].(string))
			}
		}
	} else {
		logger.Error("[Push] Telegram setWebhook Status Error: %d", resp.StatusCode)
	}
}
