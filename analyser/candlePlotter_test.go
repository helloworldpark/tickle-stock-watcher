package analyser

import (
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
	"testing"
)

func TestCandlePlotterValidity(t *testing.T) {
	dbClient := prepareDBClient()
	defer func() {
		dbClient.Close()
	}()

	stockItemChecker := watcher.NewStockItemChecker(dbClient)

	if didDraw, _ := NewCandlePlotter(dbClient, 10, "003490", stockItemChecker); !didDraw {
		t.FailNow()
	}
}
