package commons

import (
	"strconv"
	"strings"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

// GetInt parses string into int
// s: string, comma allowed
// if parsing fails: panics
func GetInt(s string) int {
	val, err := strconv.ParseInt(strings.ReplaceAll(s, ",", ""), 10, 32)
	if err != nil {
		logger.Panic("[Helper] %s", err.Error())
	}
	return int(val)
}

// GetDouble parses string into float64
// s: string, comma allowed
// if parsing fails: panics
func GetDouble(s string) float64 {
	val, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	if err != nil {
		logger.Panic("[Helper] %s", err.Error())
	}
	return val
}

// GetTimestamp returns timestamp from string value given layout.
func GetTimestamp(layout, value string) int64 {
	seoul, _ := time.LoadLocation("Asia/Seoul")
	t, err := time.ParseInLocation(layout, value, seoul)
	if err != nil {
		logger.Panic("[Helper] %s", err.Error())
	}
	return t.Unix()
}
