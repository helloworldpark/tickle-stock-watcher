package structs

import "github.com/helloworldpark/tickle-stock-watcher/database"

// UserStock is a struct describing the users' stock strategy
type UserStock struct {
	UserID    int
	StockID   string
	Strategy  string
	OrderSide int
	Repeat    bool
}

// GetDBRegisterForm is just an implementation
func (s UserStock) GetDBRegisterForm() database.DBRegisterForm {
	form := database.DBRegisterForm{
		BaseStruct:    UserStock{},
		UniqueColumns: []string{"UserID", "StockID", "OrderSide"},
	}
	return form
}
