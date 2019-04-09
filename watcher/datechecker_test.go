package watcher

import (
	"fmt"
	"testing"
	"time"
)

func TestDownloadHolidays(t *testing.T) {
	downloadHolidays(2019)
}

func TestDateChecker(t *testing.T) {
	checker := NewDateChecker()
	checker.UpdateHolidays(2019)

	loc, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now()
	fmt.Printf("Is %v Holiday: %v\n", today, checker.IsHoliday(today))

	holiday := time.Date(2019, 4, 5, 0, 0, 0, 0, loc)
	fmt.Printf("Is %v Holiday: %v\n", holiday, checker.IsHoliday(holiday))

	holiday = time.Date(2019, 9, 13, 0, 0, 0, 0, loc)
	fmt.Printf("Is %v Holiday: %v\n", holiday, checker.IsHoliday(holiday))
}
