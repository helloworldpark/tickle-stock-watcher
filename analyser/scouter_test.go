package analyser

import (
	"fmt"
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/storage"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

func TestFindProspect(t *testing.T) {
	dbClient := prepareDBClient()
	defer dbClient.Close()

	storage.InitStorage()

	itemChecker := watcher.NewStockItemChecker(dbClient)

	onFind := func(msg, savePath string) {
		fmt.Println(msg, savePath)
	}

	FindProspects(dbClient, itemChecker, onFind)
}
