package watcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

// DateChecker is a struct holding holidays as a map
type DateChecker struct {
	holidays map[int64]bool
}

// NewDateChecker returns a new DateChecker with holidays unfilled.
// Holidays are updated
func NewDateChecker() *DateChecker {
	checker := DateChecker{
		holidays: make(map[int64]bool),
	}
	checker.UpdateHolidays(commons.Now().Year())
	return &checker
}

// Year returns the current year.
func (c *DateChecker) Year() int {
	return commons.Now().Year()
}

// IsHoliday checks if the day is holiday or not.
func (c *DateChecker) IsHoliday(day time.Time) bool {
	// 토, 일요일은 주식거래가 없지롱
	if day.Weekday() == time.Sunday || day.Weekday() == time.Saturday {
		return true
	}
	// 공휴일 체크
	y, m, d := day.Date()
	zeroDay := time.Date(y, m, d, 0, 0, 0, 0, commons.AsiaSeoul)
	_, ok := c.holidays[zeroDay.Unix()]
	return ok
}

// UpdateHolidays updates the holidays of the given year.
func (c *DateChecker) UpdateHolidays(year int) {
	holidays := downloadHolidays(year)
	if len(holidays) == 0 {
		return
	}
	for _, v := range holidays {
		c.holidays[v] = true
	}
	logger.Info("[Watcher] Updated holidays")
}

func downloadHolidays(year int) []int64 {
	u := "http://marketdata.krx.co.kr/contents/COM/GenerateOTP.jspx?bld=MKD%2F01%2F0110%2F01100305%2Fmkd01100305_01&name=form&_="
	u += strconv.FormatInt(commons.Now().UnixNano()/1000000, 10)

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

func (d *DateChecker) Description() string {
	now := commons.Now()
	var buf bytes.Buffer

	addLine := func(str string, args ...interface{}) {
		if len(args) > 0 {
			str = fmt.Sprintf(str, args...)
		}
		buf.WriteString(str)
		buf.WriteString("\n")
	}

	now.Weekday()
	weekdayKorean := [7]string{"일", "월", "화", "수", "목", "금", "토"}
	currentYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, commons.AsiaSeoul).Unix()
	addLine("[Holiday]")
	addLine("    Today: %v", now)
	if d.IsHoliday(now) {
		addLine("    Today is holiday")
	} else {
		addLine("    Today is not holiday")
	}
	addLine("List")
	i := 1
	var holidayTimestamp []int64
	for timestamp := range d.holidays {
		if timestamp < currentYear {
			continue
		}
		holidayTimestamp = append(holidayTimestamp, timestamp)
	}
	sort.Slice(holidayTimestamp, func(i, j int) bool {
		return holidayTimestamp[i] < holidayTimestamp[j]
	})
	for _, timestamp := range holidayTimestamp {
		holiday := commons.Unix(timestamp)
		y, m, d := holiday.Date()
		weekday := weekdayKorean[int(holiday.Weekday())]
		addLine("    %2d. %4d년 %2d월 %2d일 %v요일", i, y, int(m), d, weekday)
		i++
	}
	return buf.String()
}
