package watcher

import (
	"sort"
	"sync"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/logger"

	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

// StockPrice is just a simple type alias
type StockPrice = structs.StockPrice

// Stock is just a simple type alias
type Stock = structs.Stock

// WatchingStock is just a simple type alias
type WatchingStock = structs.WatchingStock
type workerFunc = func() <-chan StockPrice
type internalCrawler struct {
	lastTimestamp int64
	sentinel      chan struct{}
}

// Watcher is a struct for watching the market
type Watcher struct {
	crawlers map[string]internalCrawler // key: Stock, value: last timestamp of the price info and sentinel
	dbClient *database.DBClient
}

// New creates a new Watcher struct
func New(dbClient *database.DBClient) Watcher {
	watcher := Watcher{
		crawlers: make(map[string]internalCrawler),
		dbClient: dbClient,
	}
	return watcher
}

// Register use it to register a new stock of interest.
// Internally, it investigates if the stock had been registered before
// If registered, it updates the last timestamp of the price.
// Else, it will collect price data from the beginning.
func (w *Watcher) Register(stock Stock) {
	_, ok := w.crawlers[stock.StockID]
	if ok {
		return
	}
	var watchingStock []WatchingStock
	_, err := w.dbClient.Select(watchingStock, "where StockID=?", stock.StockID)
	if err != nil {
		logger.Error("[Watcher] Error while querying WatcherStock from DB: %s", err.Error())
		return
	}
	var newWatchingStock WatchingStock
	if len(watchingStock) == 0 {
		newWatchingStock.StockID = stock.StockID
		newWatchingStock.IsWatching = true
		newWatchingStock.LastPriceTimestamp = 0
	} else {
		newWatchingStock = watchingStock[0]
		newWatchingStock.IsWatching = true
	}
	w.crawlers[stock.StockID] = internalCrawler{
		lastTimestamp: newWatchingStock.LastPriceTimestamp,
		sentinel:      make(chan struct{}),
	}
	ok, _ = w.dbClient.Insert(&newWatchingStock)
	if ok {
		return
	}
	_, err = w.dbClient.Update(&newWatchingStock)
	if err != nil {
		logger.Error("[Watcher] %s", err.Error())
	}
}

// Withdraw withdraws a stock which was of interest.
func (w *Watcher) Withdraw(stock Stock) {
	crawler, ok := w.crawlers[stock.StockID]
	if !ok {
		return
	}
	watchingStock := WatchingStock{
		StockID:            stock.StockID,
		IsWatching:         false,
		LastPriceTimestamp: crawler.lastTimestamp,
	}
	_, err := w.dbClient.Update(&watchingStock)
	if err != nil {
		logger.Error("[Watcher] Error while deleting WatchingStock: %s", err.Error())
	}
	delete(w.crawlers, stock.StockID)
}

// StartWatching use it to start watching the market.
// sleepTime : This is for making the crawler to sleep for a while. Necessary not to be blacklisted by the data providers.
// returns : <-chan StockPrice, which will give stock price until StopWatching is called.
func (w *Watcher) StartWatching(sleepTime time.Duration) <-chan StockPrice {
	// Prepare new sentinel
	for key := range w.crawlers {
		old := w.crawlers[key]
		w.crawlers[key] = internalCrawler{lastTimestamp: old.lastTimestamp, sentinel: make(chan struct{})}
	}
	// Construct function
	workerFuncGenerator := func(stockID string, sentinel <-chan struct{}) workerFunc {
		f := func() <-chan StockPrice {
			out := make(chan StockPrice)
			go func() {
				defer close(out)
				for {
					select {
					case out <- CrawlNow(stockID, 0):
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
	output := func(c <-chan StockPrice) {
		defer wg.Done()
		for v := range c {
			out <- v
		}
	}
	for stockID, crawler := range w.crawlers {
		if crawler.lastTimestamp < 0 {
			continue
		}
		worker := workerFuncGenerator(stockID, w.crawlers[stockID].sentinel)
		go output(worker())
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
	for k := range w.crawlers {
		w.StopWatchingStock(k)
	}
}

// StopWatchingStock call it when to stop watching the specific stock.
func (w *Watcher) StopWatchingStock(stockID string) {
	// Send signal to sentinel
	if c, ok := w.crawlers[stockID]; ok {
		close(c.sentinel)
	}
}

// Collect collects the past price data of the market.
func (w *Watcher) Collect(sleepTime, collectTimedelta time.Duration) {
	// 수집하기 전에 마지막으로 수집한 데가 어딘지 업데이트해둔다
	var watching []WatchingStock
	_, errWatching := w.dbClient.Select(watching, "where IsWatching=?", true)
	if errWatching != nil {
		logger.Error("[Watcher] Error while querying WatchingStock: %s", errWatching.Error())
		return
	}
	for _, v := range watching {
		sentinel := w.crawlers[v.StockID].sentinel
		w.crawlers[v.StockID] = internalCrawler{
			lastTimestamp: v.LastPriceTimestamp,
			sentinel:      sentinel,
		}
	}

	timestampTwoYears := getCollectionStartingDate(2017).Unix()

	// Construct function
	workerFuncGenerator := func(stockID string) workerFunc {
		f := func() <-chan StockPrice {
			outResult := make(chan StockPrice)
			var pivotValue int64
			if w.crawlers[stockID].lastTimestamp < timestampTwoYears {
				pivotValue = timestampTwoYears
			} else {
				pivotValue = w.crawlers[stockID].lastTimestamp
			}
			go func() {
				defer close(outResult)

				shouldCollectMore := func(collected []StockPrice) (bool, int) {
					if len(collected) == 0 {
						return false, 0
					}

					k := sort.Search(len(collected), func(i int) bool {
						return collected[i].Timestamp <= pivotValue
					})
					shouldStop := k < len(collected)
					return !shouldStop, k
				}

				var page = 1
				for {
					// 열심히 긁어온 값에서 같은 시간의 데이터를 발견하면 중지한다
					// 그렇지 않으면 페이지를 늘린다
					// 그리고 잠시 쉰다
					collected := CrawlPast(stockID, page)
					shouldGo, k := shouldCollectMore(collected)
					for i := 0; i < k; i++ {
						outResult <- collected[i]
					}
					if shouldGo {
						page++
						time.Sleep(sleepTime)
					} else {
						break
					}
				}
			}()
			return outResult
		}
		return f
	}
	// Fan In
	var wg sync.WaitGroup
	outCollect := make(chan StockPrice)
	outWatchingStock := make(chan WatchingStock)
	output := func(stockID string, c <-chan StockPrice) {
		defer wg.Done()
		var lastTimestamp int64
		for v := range c {
			if v.Timestamp > lastTimestamp {
				lastTimestamp = v.Timestamp
			}
			outCollect <- v
		}
		outWatchingStock <- WatchingStock{
			StockID:            stockID,
			LastPriceTimestamp: lastTimestamp,
			IsWatching:         true,
		}
	}
	for stockID := range w.crawlers {
		worker := workerFuncGenerator(stockID)
		go output(stockID, worker())
		wg.Add(1)
		time.Sleep(collectTimedelta)
	}
	go func() {
		wg.Wait()
		close(outCollect)
		close(outWatchingStock)
	}()

	var wg2 sync.WaitGroup
	go func() {
		defer wg2.Done()
		for v := range outWatchingStock {
			ok, _ := w.dbClient.Insert(&v)
			if ok {
				return
			}
			_, err := w.dbClient.Update(&v)
			if err != nil {
				logger.Error("[Watcher] %s", err.Error())
			}
		}
	}()
	wg2.Add(1)

	for v := range outCollect {
		_, err := w.dbClient.Insert(&v)
		if err != nil {
			logger.Error("[Watcher] %s", err.Error())
		}
	}
	wg2.Wait()
}

func getCollectionStartingDate(year int) time.Time {
	timezone, _ := time.LoadLocation("Asia/Seoul")
	start := time.Date(year, 1, 2, 0, 0, 0, 0, timezone)

	if start.Weekday() == time.Sunday {
		start = time.Date(start.Year(), start.Month(), 3, 0, 0, 0, 0, timezone)
	} else if start.Weekday() == time.Saturday {
		start = time.Date(start.Year(), start.Month(), 4, 0, 0, 0, 0, timezone)
	}
	return start
}
