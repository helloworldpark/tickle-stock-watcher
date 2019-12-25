package commons

import (
	"strconv"
	"strings"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

// AsiaSeoul is the timezone of Asia/Seoul
var AsiaSeoul *time.Location

func init() {
	AsiaSeoul = time.FixedZone("Asia/Seoul", 9*60*60)
}

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

// GetInt64 parses string into int64
// s: string, comma allowed
// if parsing fails: panics
func GetInt64(s string) int64 {
	val, err := strconv.ParseInt(strings.ReplaceAll(s, ",", ""), 10, 64)
	if err != nil {
		logger.Panic("[Helper] %s", err.Error())
	}
	return val
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
	t, err := time.ParseInLocation(layout, value, AsiaSeoul)
	if err != nil {
		logger.Panic("[Helper] %s", err.Error())
	}
	return t.Unix()
}

// Now returns time.Now() of Asia/Seoul
func Now() time.Time {
	return time.Now().In(AsiaSeoul)
}

// Today returns today's time.Time of Asia/Seoul
func Today() time.Time {
	now := Now()
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, AsiaSeoul)
}

// Unix converts timestamp to time in Seoul
func Unix(timestamp int64) time.Time {
	t := time.Unix(timestamp, 0)
	t = t.In(AsiaSeoul)
	return t
}

// MaxInt Maximum of two numbers
func MaxInt(a, b int) int {
	if a < b {
		return b
	}
	return a
}

// MinInt Minimum of two numbers
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
