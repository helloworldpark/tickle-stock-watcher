package commons

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
