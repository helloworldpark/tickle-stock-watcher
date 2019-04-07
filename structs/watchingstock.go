package structs

import "github.com/helloworldpark/tickle-stock-watcher/database"

// WatchingStock is a struct for describing if the stock item is being watched or not.
type WatchingStock struct {
	StockID            string
	IsWatching         bool
	LastPriceTimestamp int64
}

// GetDBRegisterForm is just an implementation
func (s WatchingStock) GetDBRegisterForm() database.DBRegisterForm {
	form := database.DBRegisterForm{
		BaseStruct: WatchingStock{},
		KeyColumns: []string{"StockID"},
	}
	return form
}
