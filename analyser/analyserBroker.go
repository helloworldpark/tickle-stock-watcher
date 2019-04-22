package analyser

import (
	"fmt"

	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/sdcoffey/techan"
)

// User alias
type User = structs.User

// UserStock alias
type UserStock = structs.UserStock

type analyserHolder struct {
	analyser *Analyser
	sentinel chan bool
}

// AnalyserBroker is an Analysis Manager
type AnalyserBroker struct {
	analysers map[string]*analyserHolder
	users     map[int]map[string]bool
	dbClient  *database.DBClient
}

// NewAnalyserBroker creates a new initialized pointer of AnalyserBroker
func NewAnalyserBroker(dbClient *database.DBClient) *AnalyserBroker {
	newBroker := AnalyserBroker{}
	newBroker.analysers = make(map[string]*analyserHolder)
	newBroker.dbClient = dbClient
	newBroker.users = make(map[int]map[string]bool)

	return &newBroker
}

func newHolder(stockID string) *analyserHolder {
	newAnalyser := newAnalyser(stockID)
	newAnalyser.Retain()
	holder := analyserHolder{
		analyser: newAnalyser,
		sentinel: make(chan bool),
	}
	return &holder
}

// AddStrategy adds a user's strategy with a callback which will be for sending push messages.
func (b *AnalyserBroker) AddStrategy(userStrategy UserStock, callback EventCallback) (bool, error) {
	// Handle analysers
	holder, ok := b.analysers[userStrategy.StockID]
	userStockList, userOK := b.users[userStrategy.UserID]
	retainedAnalyser := false

	if ok {
		if !userOK {
			// 이 주식은 다른 사람이 전략을 넣은 적이 있는데, 이 유저는 처음
			holder.analyser.Retain()

			userStockList = make(map[string]bool)
			userStockList[userStrategy.StockID] = true
			b.users[userStrategy.UserID] = userStockList

			retainedAnalyser = true
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
		b.analysers[userStrategy.StockID] = newHolder(userStrategy.StockID)
		retainedAnalyser = true
	}

	// Add or update strategy of the analyser
	ok, err := b.analysers[userStrategy.StockID].analyser.appendStrategy(userStrategy, callback)
	if !ok {
		if retainedAnalyser {
			holder.analyser.Release()
			if holder.analyser.Count() <= 0 {
				// Deactivate analyser
				close(holder.sentinel)
				// Delete analyser from list
				delete(b.analysers, userStrategy.StockID)
			}
		}
		return false, err
	}

	// Handle DB
	ok, err = b.dbClient.Upsert(&userStrategy)
	if !ok {
		return ok, err
	}

	return ok, err
}

// DeleteStrategy deletes a strategy from the managing list.
// Analyser will be destroyed only if there are no need to manage it.
func (b *AnalyserBroker) DeleteStrategy(user User, stockID string, orderSide int) (bool, error) {
	// Handle analysers
	holder, ok := b.analysers[stockID]
	if ok {
		holder.analyser.Release()
		if holder.analyser.Count() <= 0 {
			// Deactivate analyser
			close(holder.sentinel)
			// Delete analyser from list
			delete(b.analysers, stockID)
		} else {
			holder.analyser.deleteStrategy(user.UserID, techan.OrderSide(orderSide))
		}
	} else {
		return false, newError(fmt.Sprintf("Trying to delete an analyser which was not registered for stock ID %s", stockID))
	}

	// Delete from DB
	ok, err := b.dbClient.Delete(UserStock{}, "where UserID=? and StockID=? and OrderSide=?", user.UserID, stockID, orderSide)
	return ok, err
}

// GetStrategy gets strategy of a specific user.
func (b *AnalyserBroker) GetStrategy(user User) []UserStock {
	var result []UserStock
	_, err := b.dbClient.Select(result, "where UserID=?", user.UserID)
	if err != nil {
		logger.Error("[Analyser] Error while selecting strategy from database: %s", err.Error())
	}
	return result
}

// UpdateStrategyTriggers calculates all triggers and check if any push messages need to be sent.
func (b *AnalyserBroker) UpdateStrategyTriggers() {
	for _, v := range b.analysers {
		v.analyser.calculateStrategies()
	}
}

// FeedPrice is a function for updating the latest stock price.
func (b *AnalyserBroker) FeedPrice(stockID string, provider <-chan structs.StockPrice) {
	holder, ok := b.analysers[stockID]
	if !ok {
		return
	}
	// will be reconnected later
	if provider == nil {
		return
	}
	go func() {
		holder.analyser.prepareWatching()
		for price := range provider {
			holder.analyser.watchStockPrice(price)
			select {
			case <-holder.sentinel:
				return
			}
		}
	}()
}

// UpdatePastPrice is for updating the past price of the stock.
func (b *AnalyserBroker) UpdatePastPrice(stockPrice structs.StockPrice) {
	holder, ok := b.analysers[stockPrice.StockID]
	if ok {
		holder.analyser.appendPastStockPrice(stockPrice)
	}
}
