package analyser_test

import (
	"fmt"
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/analyser"
)

func TestTest(t *testing.T) {
	analyser.Test()
}

func TestStrategy(t *testing.T) {
	analyser := analyser.NewTestAnalyser()
	analyser.CacheFunctions()
	result, err := analyser.ParseAndCacheStrategy(1, "123456", 0, "(-macd(26, 9, 6) == 0 - 3) && (macd(26, 9, 6) <= 30)")
	if !result {
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	analyser.PrintAllStrategy()
}
