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
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

var telegramToken = ""
var telegramClient = &http.Client{Timeout: time.Second * 30}
var newError = commons.NewTaggedError("Push")

// WebhookHandler handler for webhook
type WebhookHandler interface {
	OnWebhook(token int64, msg string)
}

// TelegramUser TelegramUser
type TelegramUser struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

// TelegramChat TelegramChat
type TelegramChat struct {
	ID       int64  `json:"id"`
	ChatType string `json:"type"`
	Title    string `json:"title"`
	Username string `json:"username"`
}

// TelegramMessage TelegramMessage
type TelegramMessage struct {
	MessageID int64        `json:"message_id"`
	From      TelegramUser `json:"from"`
	Date      int64        `json:"date"`
	Chat      TelegramChat `json:"chat"`
	Text      string       `json:"text"`
}

// TelegramUpdate TelegramUpdate
type TelegramUpdate struct {
	UpdateID int64           `json:"update_id"`
	Message  TelegramMessage `json:"message"`
}

// GetTelegramToken get Telegram Bot token
func GetTelegramToken() string {
	return telegramToken
}

// GetTelegramTokenForURL get telegram bot's id
func GetTelegramTokenForURL() string {
	return strings.Split(telegramToken, ":")[0]
}

type tokenStruct struct {
	Token string `json:"token"`
}

// InitTelegram initialize telegram's bot token
func InitTelegram(filePath string) {
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		logger.Panic("[Push] %v", err)
	}

	var token tokenStruct
	if err := json.Unmarshal(raw, &token); err != nil {
		logger.Panic("[Push] %v", err)
	}

	telegramToken = token.Token
	logger.Info("[Push] Initialized Telegram")
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
			onSuccess(result["result"].(map[string]interface{}))
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

// SendMessageTelegram send message to telegram
func SendMessageTelegram(id int64, msg string) {
	body := map[string]interface{}{
		"chat_id": id,
		"text":    msg,
	}
	onSuccess := func(result map[string]interface{}) {
		_, ok := result["chat"]
		if !ok {
			logger.Error("[Push] Value corresponding to key 'chat' does not exist in response of 'sendMessage': %v", result)
			return
		}
		if result["chat"] == nil {
			logger.Error("[Push] Value corresponding to key 'chat' is nil")
			return
		}
		user, ok := result["chat"].(map[string]interface{})
		if !ok {
			logger.Error("[Push] Cannot convert 'chat' to map[string]interface{}")
			return
		}
		_, ok = user["username"]
		if !ok {
			logger.Error("[Push] Value corresponding to key 'username' does not exist")
			return
		}
		if user["username"] == nil {
			logger.Error("[Push] Value corresponding to key 'username' is nil")
			return
		}
		username, ok := user["username"].(string)
		if !ok {
			logger.Error("[Push] Cannot convert 'username' to string")
			return
		}
		logger.Info("[Push] Sent message to: %s(%d) \n message: %s", username, id, msg)
	}
	onFailure := func(err error) {
		logger.Error(newError(err.Error()).Error())
	}
	requestTelegram("sendMessage", body, onSuccess, onFailure)
}

// SendPhotoTelegram send message to telegram
func SendPhotoTelegram(id int64, caption, picURL string) {
	body := map[string]interface{}{
		"chat_id": id,
		"photo":   picURL,
		"caption": caption,
	}
	onSuccess := func(result map[string]interface{}) {
		_, ok := result["chat"]
		if !ok {
			logger.Error("[Push] Value corresponding to key 'chat' does not exist in response of 'sendMessage': %v", result)
			return
		}
		if result["chat"] == nil {
			logger.Error("[Push] Value corresponding to key 'chat' is nil")
			return
		}
		user, ok := result["chat"].(map[string]interface{})
		if !ok {
			logger.Error("[Push] Cannot convert 'chat' to map[string]interface{}")
			return
		}
		_, ok = user["username"]
		if !ok {
			logger.Error("[Push] Value corresponding to key 'username' does not exist")
			return
		}
		if user["username"] == nil {
			logger.Error("[Push] Value corresponding to key 'username' is nil")
			return
		}
		username, ok := user["username"].(string)
		if !ok {
			logger.Error("[Push] Cannot convert 'username' to string")
			return
		}
		logger.Info("[Push] Sent photo to: %s(%d) \n message: %s \n photo: %s", username, id, caption, picURL)
	}
	onFailure := func(err error) {
		logger.Error(newError(err.Error()).Error())
	}
	requestTelegram("sendPhoto", body, onSuccess, onFailure)
}

// URLTelegramUpdate uri where telegram webhook comes
func URLTelegramUpdate() string {
	return fmt.Sprintf("/api/telegram/%s", GetTelegramTokenForURL())
}

// OnTelegramUpdate handler for webhook
func OnTelegramUpdate(wh WebhookHandler) func(c *gin.Context) {
	f := func(c *gin.Context) {
		var u TelegramUpdate
		err := c.BindJSON(&u)
		if err != nil {
			logger.Error("[Push] Telegram Update: %s", err.Error())
			c.String(400, err.Error())
			return
		}
		commons.InvokeGoroutine("Push_OnTelegramUpdate", func() {
			wh.OnWebhook(u.Message.From.ID, u.Message.Text)
		})
		c.String(200, "")
	}
	return f
}
