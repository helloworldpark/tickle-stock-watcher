package commons

const (
	// BUY const
	BUY int = 0
	// SELL const
	SELL int = 1

	// DEV Phase DEV. No real API calls. Only mocking.
	DEV int = 2
	// RC Phase RC. Only testing if trading works.
	RC int = 3
	// REAL Phase REAL. Real mode.
	REAL int = 4
)

// PHASE Phase of the server.
var PHASE = DEV
