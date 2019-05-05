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
	fmt.Println("Today: ", Today())
	tm := GetTimestamp("2006-01-02", "2019-05-05")
	fmt.Println("Childrens Day: ", tm)
	y, m, d := Now().Date()
	tmm := time.Date(y, m, d, 0, 0, 0, 0, AsiaSeoul)
	fmt.Println("Mayday? ", tmm.Unix())
}
