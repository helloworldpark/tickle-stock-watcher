package watcher

import (
	"fmt"
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

func TestDownload(t *testing.T) {
	result := downloadStockSymbols(structs.KOSDAQ)
	for _, v := range result {
		fmt.Println(v)
	}
}
