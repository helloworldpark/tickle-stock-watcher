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

type AnalyserBroker struct {
	analysers map[string]*analyserHolder
	dbClient  *database.DBClient
}

func NewAnalyserBroker(dbClient *database.DBClient) *AnalyserBroker {
	newBroker := AnalyserBroker{}
	newBroker.analysers = make(map[string]*analyserHolder)
	newBroker.dbClient = dbClient

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

func (b *AnalyserBroker) AddStrategy(userStrategy UserStock, provider <-chan structs.StockPrice, callback EventCallback) (bool, error) {
	// Handle analysers
	holder, ok := b.analysers[userStrategy.StockID]
	if ok {
		holder.analyser.Retain()
	} else {
		// Create analyser
		b.analysers[userStrategy.StockID] = newHolder(userStrategy.StockID)
		// Activate analyser
		b.FeedPrice(userStrategy.StockID, provider)
	}

	// Handle DB
	ok, err := b.dbClient.Upsert(&userStrategy)
	if !ok {
		return ok, err
	}

	// Add or update strategy of the analyser
	return b.analysers[userStrategy.StockID].analyser.appendStrategy(userStrategy, callback)
}

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
	if err != nil {
		return false, err
	}
	return true, nil
}

func (b *AnalyserBroker) GetStrategy(user User) []UserStock {
	var result []UserStock
	_, err := b.dbClient.Select(result, "where UserID=?", user.UserID)
	if err != nil {
		logger.Error("[Analyser] Error while selecting strategy from database: %s", err.Error())
	}
	return result
}

func (b *AnalyserBroker) UpdateStrategyTriggers() {
	for _, v := range b.analysers {
		v.analyser.calculateStrategies()
	}
}

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

func (b *AnalyserBroker) UpdatePastPrice(stockPrice structs.StockPrice) {
	holder, ok := b.analysers[stockPrice.StockID]
	if !ok {
		return
	}
	holder.analyser.appendPastStockPrice(stockPrice)
}
