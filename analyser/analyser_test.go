package analyser

import (
	"fmt"
	"testing"

	"github.com/sdcoffey/techan"
)

func TestTest(t *testing.T) {
	Test()
}

func TestStrategy(t *testing.T) {
	analyser := NewTestAnalyser()
	result, err := analyser.ParseAndCacheStrategy(1, "123456", 0, "(-macd(26, 9, 6) == 0 - 3) && (macd(26, 9, 6) <= 30)")
	if !result {
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

func TestRuleGeneration(t *testing.T) {
	handleErr := func(err error) {
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("------------------")
	}
	analyser := NewTestAnalyser()
	tokens, err := analyser.parseTokens("close() > 0")
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
