package commons

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

// InvokeGoroutine Invoke goroutine with handling panicking
// Sends log to GCE with stack trace
// https://jacking75.github.io/go_spew/
func InvokeGoroutine(tag string, f func()) {
	go func() {
		defer func() {
			if v := recover(); v != nil {
				msg := strings.Builder{}
				msg.WriteString(Now().String())
				msg.WriteString("\n")
				msg.WriteString(fmt.Sprintf("[Reason] %+v\n", v))
				msg.WriteString("[Position] goroutine ")
				msg.WriteString(tag)
				msg.WriteString("\n")

				i := 1 // 첫잔은 버린다
				funcName, file, line, ok := runtime.Caller(i)

				for ok {
					msg.WriteString(runtime.FuncForPC(funcName).Name())
					msg.WriteString("\n    ")
					msg.WriteString(file)
					msg.WriteString(":")
					msg.WriteString(strconv.FormatInt(int64(line), 10))
					msg.WriteString("\n")
					i++
					funcName, file, line, ok = runtime.Caller(i)
				}

				fmt.Println("Is GCE Logger: ", logger.IsLoggerGCE())

				logger.Panic(msg.String())
				<-time.NewTimer(time.Second * 5).C
				panic(v)
			}
		}()

		f()
	}()
}
