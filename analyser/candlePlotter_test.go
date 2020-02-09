package analyser

import (
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

func TestCandlePlotterValidity(t *testing.T) {
	dbClient := prepareDBClient()
	defer func() {
		dbClient.Close()
	}()

	if err := CleanupOldCandleplots(); err != nil {
		panic(err)
	}
	if err := MkCandlePlotDir(); err != nil {
		panic(err)
	}

	stockItemChecker := watcher.NewStockItemChecker(dbClient)

	if didDraw, _ := NewCandlePlot(dbClient, 10, "003490", stockItemChecker); !didDraw {
		t.FailNow()
	}
}
