package analyser

import (
	"fmt"
	"testing"

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
	tokens, err := analyser.parseTokens("Price() < 0")
	handleErr(err)
	tokens, err = analyser.searchAndReplaceToFunctionTokens(tokens, "123456")
	handleErr(err)
	tokens, err = analyser.reorderTokenByPostfix(tokens)
	handleErr(err)
	for _, t := range tokens {
		fmt.Println(t.Kind, t.Value)
	}
	event, err := analyser.createEvent(tokens, techan.BUY)
	handleErr(err)

	for i := 0; i < 100; i++ {
		fmt.Println(event.HasHappened(i, nil))
	}
}
