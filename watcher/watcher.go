package watcher

import (
	"sync"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

// StockPrice is just a simple type alias
type StockPrice = structs.StockPrice

// Stock is just a simple type alias
type Stock = structs.Stock
type workerFunc = func() <-chan StockPrice

// Watcher is a struct for watching the market
type Watcher struct {
	crawlers map[Stock]int
	sentinel chan struct{}
	// watcherCollector chan StockPrice
}

// New creates a new Watcher struct
func New() Watcher {
	watcher := Watcher{
		crawlers: make(map[Stock]int),
		sentinel: make(chan struct{}),
	}
	return watcher
}

// Register use it to register a new stock of interest.
func (w *Watcher) Register(stock Stock) {
	ref, _ := w.crawlers[stock]
	w.crawlers[stock] = ref + 1
}

// Withdraw withdraws a stock which was of interest.
func (w *Watcher) Withdraw(stock Stock) {
	ref, ok := w.crawlers[stock]
	if !ok {
		return
	}

	w.crawlers[stock] = ref - 1
	if ref-1 == 0 {
		delete(w.crawlers, stock)
	}
}

// StartWatching use it to start watching the market.
// sleepTime : This is for making the crawler to sleep for a while. Necessary not to be blacklisted by the data providers.
// returns : <-chan StockPrice, which will give stock price until StopWatching is called.
func (w *Watcher) StartWatching(sleepTime time.Duration) <-chan StockPrice {
	// Prepare new sentinel
	w.sentinel = make(chan struct{})
	// Construct function
	workerFuncGenerator := func(stockID string, sentinel <-chan struct{}) workerFunc {
		f := func() <-chan StockPrice {
			out := make(chan StockPrice)
			go func() {
				defer close(out)
				for {
					select {
					case out <- StockPrice(CrawlNow(stockID, 0)):
						time.Sleep(sleepTime)
					case <-sentinel:
						return
					}
				}
			}()
			return out
		}
		return f
	}

	// Fan In
	var wg sync.WaitGroup
	out := make(chan StockPrice)
	output := func(c <-chan StockPrice, sentinel <-chan struct{}) {
		defer wg.Done()
		for v := range c {
			select {
			case out <- v:
			case <-sentinel:
				return
			}
		}
	}
	for stockID, ref := range w.crawlers {
		if ref <= 0 {
			continue
		}
		worker := workerFuncGenerator(stockID.StockID, w.sentinel)
		go output(worker(), w.sentinel)
		wg.Add(1)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// StopWatching call it when to stop watching the market.
func (w *Watcher) StopWatching() {
	// Send signal to sentinel
	close(w.sentinel)
}

// Collect collects the past price data of the market.
func (w *Watcher) Collect() {
	// Construct function

	// Fan In
}
