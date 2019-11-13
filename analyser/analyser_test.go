package analyser

import (
	"fmt"
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/sdcoffey/techan"
)

func TestRuleGeneration(t *testing.T) {
	handleErr := func(err error) {
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("------------------")
	}
	analyser := newTestAnalyser()
	tokens, err := analyser.parseTokens("extrema(Price(), 1, 5) < 0")
	handleErr(err)
	tokens, err = analyser.tidyTokens(tokens)
	handleErr(err)
	fcns, err := analyser.reorderTokenByPostfix(tokens)
	handleErr(err)
	for _, f := range fcns {
		fmt.Println(f.t.Kind, f.t.Value, f.argc)
	}
	event, err := analyser.createEvent(fcns, techan.BUY, func(price structs.StockPrice, orderSide int, userid int64, repeat bool) {
		fmt.Println("Event Callback: ", price.Close, orderSide, userid, repeat)
	})
	handleErr(err)

	for i := 0; i < 100; i++ {
		fmt.Println(event.IsTriggered(i, nil))
	}
}
