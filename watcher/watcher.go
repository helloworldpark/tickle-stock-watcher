package watcher

import (
	"sort"
	"sync"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
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
	ref           *commons.Ref
}

// WatcherAccess provides access to Watcher
type WatcherAccess interface {
	AccessWatcher() *Watcher
}

// Watcher is a struct for watching the market
type Watcher struct {
	crawlers  map[string]internalCrawler // key: Stock ID, value: last timestamp of the price info and sentinel
	dbClient  *database.DBClient
	sleepTime time.Duration
}

// New creates a new Watcher struct
func New(dbClient *database.DBClient, sleepingTime time.Duration) *Watcher {
	watcher := Watcher{
		crawlers:  make(map[string]internalCrawler),
		dbClient:  dbClient,
		sleepTime: sleepingTime,
	}
	return &watcher
}

// Register use it to register a new stock of interest.
// Internally, it investigates if the stock had been registered before
// If registered, it updates the last timestamp of the price.
// Else, it will collect price data from the beginning.
func (w *Watcher) Register(stock Stock) bool {
	old, ok := w.crawlers[stock.StockID]
	if ok {
		old.ref.Retain()
		return true
	}
	var watchingStock []WatchingStock
	_, err := w.dbClient.Select(&watchingStock, "where StockID=?", stock.StockID)
	if err != nil {
		logger.Error("[Watcher] Error while querying WatcherStock from DB: %s", err.Error())
		return false
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
	ref := &commons.Ref{}
	ref.Retain()
	w.crawlers[stock.StockID] = internalCrawler{
		lastTimestamp: newWatchingStock.LastPriceTimestamp,
		sentinel:      make(chan struct{}),
		ref:           ref,
	}
	_, err = w.dbClient.Upsert(&newWatchingStock)
	if err != nil {
		logger.Error("[Watcher] %s", err.Error())
		return false
	}
	return true
}

// Withdraw withdraws a stock which was of interest.
func (w *Watcher) Withdraw(stock Stock) bool {
	crawler, ok := w.crawlers[stock.StockID]
	if !ok {
		return true
	}
	crawler.ref.Release()
	if crawler.ref.Count() > 0 {
		return true
	}
	watchingStock := WatchingStock{
		StockID:            stock.StockID,
		IsWatching:         false,
		LastPriceTimestamp: crawler.lastTimestamp,
	}
	_, err := w.dbClient.Update(&watchingStock)
	if err != nil {
		logger.Error("[Watcher] Error while deleting WatchingStock: %s", err.Error())
		return false
	}
	close(crawler.sentinel)
	delete(w.crawlers, stock.StockID)
	return true
}

// StartWatchingStock use it to start watching the market.
// A channel of StockPrice is returned to get the price info for the given stock id.
// The channel is valid only for one day, since the channel will be closed after the market closing time.
// returns : <-chan StockPrice, which will give stock price until StopWatching is called.
func (w *Watcher) StartWatchingStock(stockID string) <-chan StockPrice {
	old := w.crawlers[stockID]
	if old.lastTimestamp < 0 {
		return nil
	}
	// Prepare new sentinel
	w.crawlers[stockID] = internalCrawler{lastTimestamp: old.lastTimestamp, sentinel: make(chan struct{})}
	// Construct function
	out := make(chan StockPrice)
	go func() {
		defer close(out)
		for {
			select {
			case out <- CrawlNow(stockID, 0):
				time.Sleep(w.sleepTime)
			case <-w.crawlers[stockID].sentinel:
				return
			}
		}
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
func (w *Watcher) Collect() {
	// 수집하기 전에 마지막으로 수집한 데가 어딘지 업데이트해둔다
	var watching []WatchingStock
	_, errWatching := w.dbClient.Select(&watching, "where IsWatching=?", true)
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

	timestampTwoYears := getCollectionStartingDate(2019).Unix()

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
					if (k + 1) < len(collected) {
						k++
					} else {
						if shouldGo == false {
							shouldGo = true
						}
					}
					for i := 0; i < k; i++ {
						outResult <- collected[i]
					}
					if shouldGo {
						page++
						time.Sleep(w.sleepTime)
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
		time.Sleep(w.sleepTime)
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
			_, err := w.dbClient.Upsert(&v)
			if err != nil {
				logger.Error("[Watcher] %s", err.Error())
			}
		}
	}()
	wg2.Add(1)

	// Write to DB by bucket
	bucketSize := 20
	buckets := make([]*[]interface{}, 2)
	bucket1 := make([]interface{}, bucketSize)
	buckets[0] = &bucket1
	bucket2 := make([]interface{}, bucketSize)
	buckets[1] = &bucket2
	activeBucket := 0
	insertToDb := func(b *[]interface{}) {
		_, err := w.dbClient.BulkInsert((*b)...)
		if err != nil {
			logger.Error("[Watcher] %s", err.Error())
		}
	}
	counter := 0
	for v := range outCollect {
		price := v
		(*buckets[activeBucket])[counter] = &price
		counter++
		if counter < bucketSize {
			continue
		}
		go insertToDb(buckets[activeBucket])
		activeBucket = (activeBucket + 1) % 2
		counter = 0
	}
	if counter > 0 {
		*buckets[activeBucket] = (*buckets[activeBucket])[:counter]
		insertToDb(buckets[activeBucket])
	}
	wg2.Wait()
}

func getCollectionStartingDate(year int) time.Time {
	start := time.Date(year, 1, 2, 0, 0, 0, 0, commons.AsiaSeoul)

	if start.Weekday() == time.Sunday {
		start = time.Date(start.Year(), start.Month(), 3, 0, 0, 0, 0, commons.AsiaSeoul)
	} else if start.Weekday() == time.Saturday {
		start = time.Date(start.Year(), start.Month(), 4, 0, 0, 0, 0, commons.AsiaSeoul)
	}
	return start
}
