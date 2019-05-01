package commons_test

import (
	"fmt"
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
)

var myerror = commons.NewTaggedError("MY")

func TestErrors(t *testing.T) {
	fmt.Println(myerror("HELLO!"))
	fmt.Println(myerror("WORLD!"))
}
