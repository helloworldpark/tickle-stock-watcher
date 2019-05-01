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
	tz := time.FixedZone("Asia/Seoul", 9*60*60)
	tt = time.Now().In(tz)
	fmt.Println("Timezone: ", tz)
	fmt.Println("commons utc Hour: ", tt.Hour())
}
