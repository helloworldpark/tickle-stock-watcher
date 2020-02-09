package analyser

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
	"github.com/sdcoffey/techan"
)

// User alias
type User = structs.User

// UserStock alias
type UserStock = structs.UserStock

type analyserHolder struct {
	analyser *Analyser
	sentinel chan struct{}
}

// BrokerAccess give access to broker
type BrokerAccess interface {
	AccessBroker() *Broker
}

// Broker is an Analysis Manager
type Broker struct {
	analysers map[string]*analyserHolder // Key: Stock ID, Value: Analyser Holder
	users     map[int64]map[string]bool  // Key: User ID, Value: Stock ID set
	dbClient  *database.DBClient
	mutex     *sync.Mutex
}

// NewBroker creates a new initialized pointer of Broker
func NewBroker(dbClient *database.DBClient) *Broker {
	newBroker := Broker{}
	newBroker.analysers = make(map[string]*analyserHolder)
	newBroker.dbClient = dbClient
	newBroker.users = make(map[int64]map[string]bool)
	newBroker.mutex = &sync.Mutex{}

	return &newBroker
}

func newHolder(stockID string) *analyserHolder {
	newAnalyser := NewAnalyser(stockID)
	newAnalyser.Retain()
	holder := analyserHolder{
		analyser: newAnalyser,
		sentinel: make(chan struct{}),
	}
	return &holder
}

// AddStrategy adds a user's strategy with a callback which will be for sending push messages.
// Returns
//     didRetainAnalyser   bool
//     error               error
func (b *Broker) AddStrategy(userStrategy UserStock, callback EventCallback, updateDB bool) (bool, error) {
	b.mutex.Lock()
	// Handle analysers
	holder, stockOK := b.analysers[userStrategy.StockID]
	userStockList, userOK := b.users[userStrategy.UserID]
	didRetainAnalyser := false // Retain 하는 경우(원칙): 뭔가 새로울 때
	b.mutex.Unlock()

	if stockOK {
		if userOK {
			// 이 주식은 다른 사람이 전략을 넣은 적이 있고, 이 유저도 넣는 경우이다
			// 만일 이전에 넣은 적이 없는 Order Side라면, Retain한다
			if !holder.analyser.hasStrategyOfOrderSide(userStrategy.UserID, userStrategy.OrderSide) {
				b.mutex.Lock()
				holder.analyser.Retain()
				b.mutex.Unlock()
				didRetainAnalyser = true
			}
		} else {
			// 이 주식은 다른 사람이 전략을 넣은 적이 있는데, 이 유저는 처음
			b.mutex.Lock()
			holder.analyser.Retain()
			b.mutex.Unlock()

			userStockList = make(map[string]bool)
			userStockList[userStrategy.StockID] = true
			b.users[userStrategy.UserID] = userStockList

			didRetainAnalyser = true
		}
	} else {
		if userOK {
			// 유저가 예전에 다른 주식의 전략은 넣은 적 있지만, 이번 주식의 전략은 어쨌건 새로 추가
			userStockList[userStrategy.StockID] = true
			b.users[userStrategy.UserID] = userStockList
		} else {
			// 유저도 뉴비, 주식도 뉴비
			userStockList = make(map[string]bool)
			userStockList[userStrategy.StockID] = true
			b.users[userStrategy.UserID] = userStockList
		}
		// Create analyser
		b.mutex.Lock()
		b.analysers[userStrategy.StockID] = newHolder(userStrategy.StockID)
		b.mutex.Unlock()

		holder = b.analysers[userStrategy.StockID]
		didRetainAnalyser = true
	}

	// Add or update strategy of the analyser
	b.mutex.Lock()
	ok, err := b.analysers[userStrategy.StockID].analyser.AppendStrategy(userStrategy, callback)
	b.mutex.Unlock()
	if !ok {
		if didRetainAnalyser {
			b.mutex.Lock()
			holder.analyser.Release()
			if holder.analyser.Count() <= 0 {
				// Deactivate analyser
				close(holder.sentinel)
				// Delete analyser from list
				delete(b.analysers, userStrategy.StockID)
			}
			b.mutex.Unlock()
		}
		didRetainAnalyser = false
		return didRetainAnalyser, err
	}

	// Handle DB if needed
	if updateDB {
		ok, err = b.dbClient.Upsert(&userStrategy)
	}

	// Update stock price if needed
	if !stockOK {
		b.UpdatePastPriceOfStock(userStrategy.StockID)
	}
	return didRetainAnalyser, err
}

// DeleteStrategy deletes a strategy from the managing list.
// Analyser will be destroyed only if there are no need to manage it.
func (b *Broker) DeleteStrategy(user User, stockID string, orderSide int) error {
	// Handle analysers
	holder, ok := b.analysers[stockID]
	if ok {
		b.mutex.Lock()
		holder.analyser.Release()
		if holder.analyser.Count() <= 0 {
			// Deactivate analyser
			close(holder.sentinel)
			logger.Info("[Analyser] Closed sentinel %s", stockID)
			// Delete analyser from list
			delete(b.analysers, stockID)
		} else {
			holder.analyser.DeleteStrategy(user.UserID, techan.OrderSide(orderSide))
		}
		defer b.mutex.Unlock()
	} else {
		return newError(fmt.Sprintf("Trying to delete an analyser which was not registered for stock ID %s", stockID))
	}

	// Delete from DB
	_, err := b.dbClient.Delete(UserStock{}, "where UserID=? and StockID=? and OrderSide=?", user.UserID, stockID, orderSide)
	return err
}

// GetStrategy gets strategy of a specific user.
func (b *Broker) GetStrategy(user User) []UserStock {
	var result []UserStock
	_, err := b.dbClient.Select(&result, "where UserID=?", user.UserID)
	if err != nil {
		logger.Error("[Analyser] Error while selecting strategy from database: %s", err.Error())
	}
	return result
}

// CanFeedPrice returns if can feed price to analyser
func (b *Broker) CanFeedPrice(stockID string) bool {
	b.mutex.Lock()
	holder, ok := b.analysers[stockID]
	b.mutex.Unlock()
	if !ok {
		logger.Warn("[Analyser] Attempt to feed price of nonexisting stock ID: %s", stockID)
		return false
	}
	b.mutex.Lock()
	isWatching := holder.analyser.isWatchingPrice()
	b.mutex.Unlock()
	return !isWatching
}

// FeedPrice is a function for updating the latest stock price.
func (b *Broker) FeedPrice(stockID string, provider <-chan structs.StockPrice) {
	canFeed := b.CanFeedPrice(stockID)
	if !canFeed {
		logger.Warn("[Analyser] Cannot feed stock ID: %s", stockID)
		return
	}
	// will be reconnected later
	if provider == nil {
		logger.Warn("[Analyser] Provider is nil: %s", stockID)
		return
	}
	b.mutex.Lock()
	holder := b.analysers[stockID]
	holder.analyser.prepareWatching()
	b.mutex.Unlock()
	funcWork := func() {
		defer func() {
			holder.analyser.stopWatchingPrice()
			logger.Info("[Analyser] Stop watching price: %s", stockID)
		}()
		for {
			select {
			case price := <-provider:
				holder.analyser.watchPrice(price)
				holder.analyser.CalculateStrategies()
			case <-holder.sentinel:
				logger.Info("[Analyser] Holder Sentinel Called: %s", stockID)
				return
			}
		}
	}
	commons.InvokeGoroutine(fmt.Sprintf("[Broker][%s]", stockID), funcWork)
}

// StopFeedingPrice Stop feeding price of all analysers
func (b *Broker) StopFeedingPrice() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for _, holder := range b.analysers {
		close(holder.sentinel)
		holder.sentinel = make(chan struct{})
	}
}

// UpdatePastPrice is for updating the past price of the stock.
func (b *Broker) UpdatePastPrice() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	for stockID, holder := range b.analysers {
		b.updatePastPriceOfStockImpl(stockID, holder)
	}
	logger.Info("[Analyser] UpdatePastPrice: %d analysers", len(b.analysers))
}

//UpdatePastPriceOfStock is for updating the past price of the specific stock.
func (b *Broker) UpdatePastPriceOfStock(stockID string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	holder, ok := b.analysers[stockID]
	if !ok {
		logger.Error("[Analyser] Error while updating past price of %s: no such analyser registered", stockID)
		return
	}
	b.updatePastPriceOfStockImpl(stockID, holder)
	logger.Info("[Analyser] UpdatePastPriceOfStock %s", stockID)
}

func (b *Broker) updatePastPriceOfStockImpl(stockID string, holder *analyserHolder) {
	timestampFrom := holder.analyser.NeedPriceFrom()
	var prices []structs.StockPrice
	_, err := b.dbClient.Select(&prices,
		"where StockID=? and Timestamp>=? order by Timestamp",
		stockID, timestampFrom)
	if err != nil {
		logger.Error("[Analyser] Error while updating past price of %s since %s: %s",
			stockID, commons.Unix(timestampFrom).String(), err.Error())
		return
	}
	for i := range prices {
		holder.analyser.AppendPastPrice(prices[i])
	}
	logger.Info("[Analyser] Updated past price info of %s: %d cases", stockID, len(prices))
}

// Description description of this Watcher
func (b *Broker) Description() string {
	now := commons.Now()
	var buf bytes.Buffer

	addLine := func(str string, args ...interface{}) {
		if len(args) > 0 {
			str = fmt.Sprintf(str, args...)
		}
		buf.WriteString(str)
		buf.WriteString("\n")
	}

	addLine("[AnalyserBroker] Status \n%v", now)
	addLine("Users: %v", len(b.users))
	for userid, stocks := range b.users {
		addLine("    [UserID: #%v]", userid)
		for stock := range stocks {
			addLine("        [Stock ID: %v]", stock)
		}
	}

	addLine("Analysers: %v", len(b.analysers))
	for stockid, holder := range b.analysers {
		addLine("    [Analyser#%v]", stockid)
		addLine("        [IsWatching: %v]", holder.analyser.isWatchingPrice())
		addLine("        [Reference Count: %v]", holder.analyser.Count())
		addLine("        [Time Series: %v]", holder.analyser.timeSeries.LastIndex()+1)
		lastCandle := holder.analyser.timeSeries.LastCandle()
		addLine("            [Last Candle]")
		addLine("                [Time:   %v]", lastCandle.Period)
		addLine("                [Open:   %v]", lastCandle.OpenPrice.FormattedString(2))
		addLine("                [Close:  %v]", lastCandle.ClosePrice.FormattedString(2))
		addLine("                [High:   %v]", lastCandle.MaxPrice.FormattedString(2))
		addLine("                [Low:    %v]", lastCandle.MinPrice.FormattedString(2))
		addLine("                [Volume: %v]", lastCandle.Volume.FormattedString(2))
	}

	return buf.String()
}

func (b *Broker) Situation(stockName, stockID string) (bool, string, string) {
	stockAccess := watcher.NewStockItemChecker(b.dbClient)
	didDraw, savePath := NewCandlePlotter(b.dbClient, 10, stockID, stockAccess)
	if !didDraw {
		return false, "", ""
	}

	var buf bytes.Buffer

	addLine := func(str string, args ...interface{}) {
		if len(args) > 0 {
			str = fmt.Sprintf(str, args...)
		}
		buf.WriteString(str)
		buf.WriteString("\n")
	}

	now := commons.Now()
	addLine("[AnalyserBroker] Prospect of the day: #%s(%s) \n%v", stockID, stockName, now)

	return true, savePath, buf.String()
}
