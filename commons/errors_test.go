package commons

import (
	"fmt"
	"testing"
)

var myerror = NewTaggedError("MY")

func TestErrors(t *testing.T) {
	fmt.Println(myerror("HELLO!"))
	fmt.Println(myerror("WORLD!"))
}
