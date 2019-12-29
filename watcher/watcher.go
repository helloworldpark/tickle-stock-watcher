package watcher

import (
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/logger"

	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

// CandleTimeUnitMinute 하나의 캔들은 15분
const CandleTimeUnitMinute = int64(15) // 15분
// CandleTimeUnitSeconds 하나의 캔들은 15분, 이를 초 단위로 표현
const CandleTimeUnitSeconds = CandleTimeUnitMinute * 60

// CandleTimeUnitNanoseconds 하나의 캔들은 15분, 이를 time.Time 구조체로 표현
const CandleTimeUnitNanoseconds = time.Minute * 15 // 15분

// StockPrice is just a simple type alias
type StockPrice = structs.StockPrice

// Stock is just a simple type alias
type Stock = structs.Stock

// WatchingStock is just a simple type alias
type WatchingStock = structs.WatchingStock
type workerFunc = func() <-chan StockPrice

// WatcherAccess provides access to Watcher
type WatcherAccess interface {
	AccessWatcher() *Watcher
}

type internalCrawler struct {
	lastTimestamp int64
	sentinel      chan struct{}
	ref           *commons.Ref
}

// Watcher is a struct for watching the market
type Watcher struct {
	crawlers  map[string]internalCrawler // key: Stock ID, value: last timestamp of the price info and sentinel
	dbClient  *database.DBClient
	sleepTime time.Duration
	mutex     *sync.Mutex
}

// New creates a new Watcher struct
func New(dbClient *database.DBClient, sleepingTime time.Duration) *Watcher {
	watcher := Watcher{
		crawlers:  make(map[string]internalCrawler),
		dbClient:  dbClient,
		sleepTime: sleepingTime,
		mutex:     &sync.Mutex{},
	}
	return &watcher
}

func newInternalCrawler(lastTimestamp int64) internalCrawler {
	ref := &commons.Ref{}
	ref.Retain()
	return internalCrawler{
		lastTimestamp: lastTimestamp,
		sentinel:      make(chan struct{}),
		ref:           ref,
	}
}

// Register use it to register a new stock of interest.
// Internally, it investigates if the stock had been registered before
// If registered, it updates the last timestamp of the price.
// Else, it will collect price data from the beginning.
func (w *Watcher) Register(stock Stock) bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	_, ok := w.crawlers[stock.StockID]
	if ok {
		w.crawlers[stock.StockID].ref.Retain()
		logger.Info("[Watcher] Registered to watch %s(%s): Referred %d times", stock.Name, stock.StockID, w.crawlers[stock.StockID].ref.Count())
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
	w.crawlers[stock.StockID] = newInternalCrawler(newWatchingStock.LastPriceTimestamp)
	_, err = w.dbClient.Upsert(&newWatchingStock)
	if err == nil {
		logger.Info("[Watcher] Registered to watch %s(%s)", stock.Name, stock.StockID)
	} else {
		logger.Error("[Watcher] %s", err.Error())
	}
	return err == nil
}

// Withdraw withdraws a stock which was of interest.
func (w *Watcher) Withdraw(stock Stock) bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	crawler, ok := w.crawlers[stock.StockID]
	if !ok {
		logger.Warn("[Watcher] Attempt to withdraw nonexisting stock ID: %s", stock.StockID)
		return true
	}
	w.crawlers[stock.StockID].ref.Release()
	if w.crawlers[stock.StockID].ref.Count() > 0 {
		logger.Info("[Watcher] Withdrawal success: currently %d needs", w.crawlers[stock.StockID].ref.Count())
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
	logger.Info("[Watcher] Withdrawal success: no more need to watch or collect %s(%s)", stock.Name, stock.StockID)
	return true
}

// StartWatchingStock use it to start watching the market.
// A channel of StockPrice is returned to get the price info for the given stock id.
// The channel is valid only for one day, since the channel will be closed after the market closing time.
// returns : <-chan StockPrice, which will give stock price until StopWatching is called.
func (w *Watcher) StartWatchingStock(stockID string) <-chan StockPrice {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	old := w.crawlers[stockID]
	if old.lastTimestamp < 0 {
		logger.Warn("[Watcher] Negative last timestamp: %d of stock ID: %s", old.lastTimestamp, stockID)
		return nil
	}
	// Prepare new sentinel
	w.crawlers[stockID] = newInternalCrawler(old.lastTimestamp)
	// Construct function
	out := make(chan StockPrice)
	funcWork := func() {
		defer close(out)
		for {
			select {
			case out <- CrawlNow(stockID, 0):
				time.Sleep(w.sleepTime)
			case <-w.crawlers[stockID].sentinel:
				return
			}
		}
	}
	commons.InvokeGoroutine("Watcher_StartWatchingStock_"+stockID, funcWork)
	logger.Info("[Watcher] StartWatchingStock: %s", stockID)
	return out
}

// StopWatching call it when to stop watching the market.
func (w *Watcher) StopWatching() {
	// Send signal to sentinel
	for k := range w.crawlers {
		w.StopWatchingStock(k)
	}
	logger.Info("[Watcher] Stop watching %d stocks", len(w.crawlers))
}

// StopWatchingStock call it when to stop watching the specific stock.
func (w *Watcher) StopWatchingStock(stockID string) {
	// Send signal to sentinel
	if c, ok := w.crawlers[stockID]; ok {
		c.sentinel <- struct{}{}
		close(c.sentinel)
		c.sentinel = nil
		w.crawlers[stockID] = c
	}
	logger.Info("[Watcher] Stop watching stock ID: %s", stockID)
}

// Collect collects the past price data of the market.
func (w *Watcher) Collect() {
	// 수집하기 전에 마지막으로 수집한 데가 어딘지 업데이트해둔다
	var watching []WatchingStock
	_, errWatching := w.dbClient.Select(&watching, "where IsWatching=?", true)
	if errWatching != nil {
		logger.Error("[Watcher] Error while querying WatchingStock for Collect: %s", errWatching.Error())
		return
	}
	registeredWatching := make([]WatchingStock, 0)
	w.mutex.Lock()
	for i := range watching {
		_, ok := w.crawlers[watching[i].StockID]
		if ok {
			registeredWatching = append(registeredWatching, watching[i])
		}
	}

	for _, watch := range registeredWatching {
		sentinel := w.crawlers[watch.StockID].sentinel
		newCrawler := newInternalCrawler(watch.LastPriceTimestamp)
		newCrawler.sentinel = sentinel
		w.crawlers[watch.StockID] = newCrawler
	}
	logger.Info("[Watcher] Start Collect %d stocks", len(registeredWatching))
	w.mutex.Unlock()

	timestampTwoYears := GetCollectionStartingDate(commons.Now().Year() - 2).Unix()

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
			commons.InvokeGoroutine("Watcher_Collect_workerFunc_"+stockID, func() {
				defer close(outResult)

				shouldCollectMore := func(collected []StockPrice) (bool, int) {
					k := sort.Search(len(collected), func(i int) bool {
						return collected[i].Timestamp <= pivotValue
					})
					shouldGo := k == len(collected)
					return shouldGo, k
				}

				var page = 1
				var collectedLastTimestamp = int64(0)
				for {
					// 열심히 긁어온 값에서 같은 시간의 데이터를 발견하면 중지한다
					// 그렇지 않으면 페이지를 늘린다
					// 그리고 잠시 쉰다
					collected := CrawlPast(stockID, page)
					if len(collected) == 0 {
						break
					}
					shouldGo, k := shouldCollectMore(collected)
					if (k + 1) < len(collected) {
						k++
					}
					for i := 0; i < k; i++ {
						outResult <- collected[i]
						if collectedLastTimestamp < collected[i].Timestamp {
							collectedLastTimestamp = collected[i].Timestamp
						} else {
							shouldGo = false
						}
					}
					if shouldGo {
						page++
						time.Sleep(w.sleepTime)
					} else {
						break
					}
				}
			})
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
	randomGen := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, watch := range registeredWatching {
		stockID := watch.StockID
		worker := workerFuncGenerator(stockID)
		go output(stockID, worker())
		wg.Add(1)
		sleepTime := randomGen.Float64()
		sleepTime += 0.001
		duration := time.Duration(sleepTime * float64(time.Second))
		time.Sleep(duration)
	}
	commons.InvokeGoroutine("Watcher_Collect_dbcollect1", func() {
		wg.Wait()
		close(outCollect)
		close(outWatchingStock)
	})

	var wg2 sync.WaitGroup
	commons.InvokeGoroutine("Watcher_Collect_dbcollect2", func() {
		defer wg2.Done()
		for v := range outWatchingStock {
			_, err := w.dbClient.Upsert(&v)
			if err != nil {
				logger.Error("[Watcher] Error while Collect: %s", err.Error())
			}
		}
	})
	wg2.Add(1)

	// Write to DB by bucket
	bucketSize := 200
	buckets := make([]*[]interface{}, 2)
	bucket1 := make([]interface{}, bucketSize)
	buckets[0] = &bucket1
	bucket2 := make([]interface{}, bucketSize)
	buckets[1] = &bucket2
	activeBucket := 0
	insertToDb := func(b *[]interface{}) {
		_, err := w.dbClient.BulkInsert(true, (*b)...)
		if err != nil {
			logger.Error("[Watcher] Error while Collect: %s", err.Error())
		}
	}
	counter := 0
	total := 0
	for v := range outCollect {
		price := v
		(*buckets[activeBucket])[counter] = &price
		counter++
		if counter < bucketSize {
			continue
		}
		go insertToDb(buckets[activeBucket])
		activeBucket = (activeBucket + 1) % 2
		total += counter
		counter = 0
	}
	if counter > 0 {
		*buckets[activeBucket] = (*buckets[activeBucket])[:counter]
		insertToDb(buckets[activeBucket])
		total += counter
	}
	wg2.Wait()
	logger.Info("[Watcher] Finished Collect: %d stocks, %d items", len(registeredWatching), total)
}

// GetCollectionStartingDate gets time until when to crawl
func GetCollectionStartingDate(year int) time.Time {
	start := time.Date(year, 1, 2, 0, 0, 0, 0, commons.AsiaSeoul)

	y, m, _ := start.Date()
	if start.Weekday() == time.Sunday {
		start = time.Date(y, m, 3, 0, 0, 0, 0, commons.AsiaSeoul)
	} else if start.Weekday() == time.Saturday {
		start = time.Date(y, m, 4, 0, 0, 0, 0, commons.AsiaSeoul)
	}
	return start
}
