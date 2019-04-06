package structs

import "github.com/helloworldpark/tickle-stock-watcher/database"

// Market is an enum type representing the type of the stock market
type Market string

const (
	// KOSPI market
	KOSPI = "kospi"
	// KOSDAC market
	KOSDAC = "kosdac"
)

// Stock is a struct describing each stock item
type Stock struct {
	Name       string
	StockID    string
	MarketType Market
}

// GetDBRegisterForm is just an implementation
func (s Stock) GetDBRegisterForm() database.DBRegisterForm {
	keyCols := make([]string, 1)
	keyCols[0] = "StockID"
	form := database.DBRegisterForm{
		BaseStruct: Stock{},
		KeyColumns: keyCols,
	}
	return form
}
