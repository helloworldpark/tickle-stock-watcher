package logger_test

import (
	"testing"

	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

func TestLogger(t *testing.T) {
	logger.Info("This is info")
	logger.Info("This is info %s %s", "meme", "papa")
}
