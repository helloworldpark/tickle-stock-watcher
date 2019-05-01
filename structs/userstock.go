package structs

import "github.com/helloworldpark/tickle-stock-watcher/database"
import "github.com/helloworldpark/tickle-stock-watcher/logger"

// UserStock is a struct describing the users' stock strategy
type UserStock struct {
	UserID    int64
	StockID   string
	Strategy  string
	OrderSide int
	Repeat    bool `db:"RepeatStrategy"`
}

// GetDBRegisterForm is just an implementation
func (s UserStock) GetDBRegisterForm() database.DBRegisterForm {
	form := database.DBRegisterForm{
		BaseStruct:    UserStock{},
		UniqueColumns: []string{"UserID", "StockID", "OrderSide"},
	}
	return form
}

// AllStrategies returns all strategies
func AllStrategies(client *database.DBClient) []UserStock {
	var userStrategyList []UserStock
	_, err := client.Select(&userStrategyList, "where true")
	if err != nil {
		logger.Error("[Structs] Error while selecting user strategies: %s", err.Error())
	}
	return userStrategyList
}
