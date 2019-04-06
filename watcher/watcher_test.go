package watcher_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

func TestWatcher(t *testing.T) {
	w := watcher.New()
	w.Register(commons.Stock{Name: "Samsung Electronics", StockID: "005930", MarketType: commons.KOSPI})
	w.Register(commons.Stock{Name: "Korean Air", StockID: "003490", MarketType: commons.KOSPI})
	w.Register(commons.Stock{Name: "Hanwha Chemicals", StockID: "009830", MarketType: commons.KOSPI})
	timer := time.NewTimer(time.Duration(1) * time.Second)
	go func() {
		<-timer.C
		w.StopWatching()
	}()
	handle := w.StartWatching(time.Duration(500) * time.Millisecond)
	for v := range handle {
		fmt.Println(v)
	}
	fmt.Println("Finished!!!!")
}
