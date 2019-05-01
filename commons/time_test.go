package commons

import (
	"fmt"
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	tt := Now()
	fmt.Println("Hour: ", tt.Hour())
	tt = tt.In(time.UTC)
	fmt.Println("Hour: ", tt.Hour())
}
