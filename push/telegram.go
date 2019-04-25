package push

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

var telegramToken = ""
var telegramClient = &http.Client{Timeout: time.Second * 30}

type WebhookHandler interface {
	OnWebhook(id int64, msg, messenger string) error
}

type TelegramUser struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

type TelegramChat struct {
	ID       int64  `json:"id"`
	ChatType string `json:"type"`
	Title    string `json:"title"`
	Username string `json:"username"`
}

type TelegramMessage struct {
	MessageID int64        `json:"message_id"`
	From      TelegramUser `json:"from"`
	Date      int64        `json:"date"`
	Chat      TelegramChat `json:"chat"`
	Text      string       `json:"text"`
}

type TelegramUpdate struct {
	UpdateID int64           `json:"update_id"`
	Message  TelegramMessage `json:"message"`
}

type telegramError struct {
	msg string
}

func newError(msg string) telegramError {
	return telegramError{msg: msg}
}

func (e telegramError) Error() string {
	return fmt.Sprintf("[Push] Error at Telegram API Client: %s", e.msg)
}

func GetTelegramToken() string {
	return telegramToken
}

func GetTelegramTokenForURL() string {
	return strings.Split(telegramToken, ":")[0]
}

type tokenStruct struct {
	Token string `json:"token"`
}

func InitTelegram(filePath string) {
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		logger.Panic("%v", err)
	}

	var token tokenStruct
	if err := json.Unmarshal(raw, &token); err != nil {
		logger.Panic("%v", err)
	}

	telegramToken = token.Token
}

func telegramAPI(method string) string {
	if telegramToken == "" {
		logger.Panic("[Push] Telegram client not initialized")
	}
	baseURL := "https://api.telegram.org/bot%s/%s"
	return fmt.Sprintf(baseURL, telegramToken, method)
}

func requestTelegram(method string, body map[string]interface{}, onSuccess func(map[string]interface{}), onFailure func(error)) {
	url := telegramAPI(method)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		if onFailure != nil {
			onFailure(err)
		}
		return
	}
	bodyBuffer := bytes.NewBuffer(bodyBytes)
	resp, err := telegramClient.Post(url, "application/json", bodyBuffer)
	if err != nil {
		if onFailure != nil {
			onFailure(err)
		}
		return
	}

	if onSuccess == nil {
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode/100 == 2 {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			if onFailure != nil {
				onFailure(err)
			}
			return
		}
		var tmp interface{}
		json.Unmarshal(respBody, &tmp)
		result := tmp.(map[string]interface{})
		if result["ok"].(bool) {
			onSuccess(result)
		} else {
			if onFailure != nil {
				onFailure(newError(result["description"].(string)))
			}
		}
	} else {
		if onFailure != nil {
			onFailure(newError(fmt.Sprintf("%d", resp.StatusCode)))
		}
	}
}

func SetWebhookTelegram() {
	body := map[string]interface{}{
		"url":             "https://stock.ticklemeta.kr/api/telegram/" + GetTelegramTokenForURL(),
		"allowed_updates": []string{"message"},
	}
	onSuccess := func(result map[string]interface{}) {
		logger.Info("[Push] Telegram setWebhook Success")
	}
	onFailure := func(err error) {
		logger.Error(newError(err.Error()).Error())
	}
	requestTelegram("setWebhook", body, onSuccess, onFailure)
}

func SendMessageTelegram(id int64, msg string) {
	body := map[string]interface{}{
		"chat_id": id,
		"text":    msg,
	}
	onSuccess := func(result map[string]interface{}) {
		logger.Info("%v", result)
		user := result["from"].(map[string]interface{})
		username := user["username"].(string)
		logger.Info("[Push] Sent message to: %s(%d) \n message: %s", username, id, msg)
	}
	onFailure := func(err error) {
		logger.Error(newError(err.Error()).Error())
	}
	requestTelegram("sendMessage", body, onSuccess, onFailure)
}

func URLTelegramUpdate() string {
	return fmt.Sprintf("/api/telegram/%s", GetTelegramTokenForURL())
}

func OnTelegramUpdate(wh WebhookHandler) func(c *gin.Context) {
	f := func(c *gin.Context) {
		var u TelegramUpdate
		err := c.BindJSON(&u)
		if err != nil {
			logger.Error("[Main] Telegram Update Error: %s", err.Error())
			c.String(400, err.Error())
			return
		}
		go func() {
			err = wh.OnWebhook(u.Message.From.ID, u.Message.Text, "Telegram")
			if err != nil {
				// Push message to user
			}
		}()
		c.String(200, "")
	}
	return f
}
