package watcher

import (
	"fmt"
	"testing"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
)

func TestDownloadHolidays(t *testing.T) {
	downloadHolidays(2019)
}

func TestDateChecker(t *testing.T) {
	checker := NewDateChecker()
	checker.UpdateHolidays(2019)

	today := commons.Now()
	fmt.Printf("Is %v Holiday: %v\n", today, checker.IsHoliday(today))

	holiday := time.Date(2019, 4, 5, 0, 0, 0, 0, commons.AsiaSeoul)
	fmt.Printf("Is %v Holiday: %v\n", holiday, checker.IsHoliday(holiday))

	holiday = time.Date(2019, 9, 13, 0, 0, 0, 0, commons.AsiaSeoul)
	fmt.Printf("Is %v Holiday: %v\n", holiday, checker.IsHoliday(holiday))

	holiday = time.Date(2019, 5, 1, 0, 0, 0, 0, commons.AsiaSeoul)
	fmt.Printf("Is %v Holiday: %v\n", holiday, checker.IsHoliday(holiday))
}
