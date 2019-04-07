package structs

import "github.com/helloworldpark/tickle-stock-watcher/database"

// StockPrice is a struct describing price of the stock
type StockPrice struct {
	StockID   string
	Timestamp int64
	Open      int
	Close     int
	High      int
	Low       int
	Volume    float64
}

// GetDBRegisterForm is just an implementation
func (s StockPrice) GetDBRegisterForm() database.DBRegisterForm {
	form := database.DBRegisterForm{
		BaseStruct:    StockPrice{},
		UniqueColumns: []string{"StockID", "Timestamp"},
	}
	return form
}
