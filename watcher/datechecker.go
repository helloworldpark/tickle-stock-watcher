package watcher

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

type DateChecker struct {
	Holidays map[int64]bool
}

func NewDateChecker() DateChecker {
	checker := DateChecker{
		Holidays: make(map[int64]bool),
	}
	return checker
}

func (c *DateChecker) Year() int {
	return time.Now().Year()
}

func (c *DateChecker) IsHoliday(day time.Time) bool {
	zeroDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	_, ok := c.Holidays[zeroDay.Unix()]
	return ok
}

func (c *DateChecker) UpdateHolidays(year int) {
	holidays := downloadHolidays(year)
	if len(holidays) == 0 {
		return
	}
	for _, v := range holidays {
		c.Holidays[v] = true
	}
}

func downloadHolidays(year int) []int64 {
	u := "http://marketdata.krx.co.kr/contents/COM/GenerateOTP.jspx?bld=MKD%2F01%2F0110%2F01100305%2Fmkd01100305_01&name=form&_="
	u += strconv.FormatInt(time.Now().UnixNano()/1000000, 10)

	resOTP, err := http.Get(u)
	if err != nil {
		logger.Error("[Watcher] Error while getting OTP code for holiday checking: %s", err.Error())
		return nil
	}

	defer resOTP.Body.Close()
	byteOTP, err := ioutil.ReadAll(resOTP.Body)
	if err != nil {
		logger.Error("[Watcher] Error while reading OTP code for holiday checking: %s", err.Error())
		return nil
	}

	otp := string(byteOTP)

	formData := url.Values{
		"code":          {otp},
		"search_bas_yy": {strconv.FormatInt(int64(year), 10)},
		"gridTP":        {"KRX"},
		"pagePath":      {"/contents/MKD/01/0110/01100305/MKD01100305.jsp"},
		"pageFirstCall": {"Y"},
	}
	respHoliday, err := http.PostForm("http://marketdata.krx.co.kr/contents/MKD/99/MKD99000001.jspx", formData)
	if err != nil {
		logger.Error("[Watcher] Error while requesting holidays: %s", err.Error())
		return nil
	}

	defer respHoliday.Body.Close()
	byteHoliday, err := ioutil.ReadAll(respHoliday.Body)
	if err != nil {
		logger.Error("[Watcher] Error while reading holiday bytes: %s", err.Error())
		return nil
	}

	var downloaded map[string]interface{}
	json.Unmarshal(byteHoliday, &downloaded)
	holidays := downloaded["block1"].([]interface{})
	result := make([]int64, len(holidays))
	for i, v := range holidays {
		h := v.(map[string]interface{})
		dateString := (h["calnd_dd"]).(string)
		dateTimestamp := commons.GetTimestamp("2006-01-02", dateString)
		result[i] = dateTimestamp
	}
	return result
}
