package commons

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
