package watcher_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

func TestWatcher(t *testing.T) {
	w := watcher.New()
	w.Register(structs.Stock{Name: "Samsung Electronics", StockID: "005930", MarketType: structs.KOSPI})
	w.Register(structs.Stock{Name: "Korean Air", StockID: "003490", MarketType: structs.KOSPI})
	w.Register(structs.Stock{Name: "Hanwha Chemicals", StockID: "009830", MarketType: structs.KOSPI})
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

	timer = time.NewTimer(time.Duration(1) * time.Second)
	go func() {
		<-timer.C
		w.StopWatching()
	}()
	handle = w.StartWatching(time.Duration(500) * time.Millisecond)
	for v := range handle {
		fmt.Println(v)
	}
	fmt.Println("Finished Again!!!!")
}
