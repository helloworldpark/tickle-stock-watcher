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
	uniqueCols := make([]string, 2)
	uniqueCols[0] = "StockID"
	uniqueCols[1] = "Timestamp"
	form := database.DBRegisterForm{
		BaseStruct:    StockPrice{},
		UniqueColumns: uniqueCols,
	}
	return form
}
