package commons

import "testing"

import "time"

func TestInvoker(t *testing.T) {
	timer := time.NewTimer(time.Second * 10)

	InvokeGoroutine("aaa", func() {
		intermediate()
	})

	<-timer.C
}

func mustFail() {
	panic("ttttt")
}

func intermediate() {
	mustFail()
}
