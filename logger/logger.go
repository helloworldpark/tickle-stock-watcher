package logger

import (
	"log"
	"runtime"

	// Imports the Stackdriver Logging client package.
	"cloud.google.com/go/logging"
	"golang.org/x/net/context"
)

var (
	isLoggerGCE  bool
	loggerClient *logging.Client
	loggerInfo   *log.Logger
	loggerWarn   *log.Logger
	loggerError  *log.Logger
	loggerPanic  *log.Logger
	logName      string
)

func init() {
	log.Print("DJFKLJDLFJLD")
	ctx := context.Background()

	// Sets your Google Cloud Platform project ID.
	projectID := "ticklemeta-203110"

	// Sets the name of the log to write to.
	logName = "StockWatcher"

	// Checks OS
	// If Linux, use GCP Logger
	// Else, use only stdout
	if runtime.GOOS == "linux" {
		// Creates a client.
		client, err := logging.NewClient(ctx, projectID)
		if err != nil {
			loggerInfo = nil
			loggerWarn = nil
			loggerError = nil
			loggerPanic = nil
			isLoggerGCE = false
			log.Printf("Failed to create client: %v, using builtin logger", err)
			return
		}
		loggerClient = client

		loggerInfo = client.Logger(logName).StandardLogger(logging.Info)
		loggerWarn = client.Logger(logName).StandardLogger(logging.Warning)
		loggerError = client.Logger(logName).StandardLogger(logging.Error)
		loggerPanic = client.Logger(logName).StandardLogger(logging.Critical)

		isLoggerGCE = true
	} else {
		isLoggerGCE = false
	}

	// Logs "hello world", log entry is visible at
	// Stackdriver Logs.
	Info("Logger init")
}

// IsLoggerGCE provides interface if current logger is GCE
func IsLoggerGCE() bool {
	return isLoggerGCE
}

// Close closes GCE Client
func Close() {
	if loggerClient != nil {
		loggerClient.Close()
	}
}

// Info prints logs as this format: [INFO]
func Info(format string, v ...interface{}) {
	handleLog(loggerInfo, "INFO", format, v...)
}

// Warn prints logs as this format: [WARN]
func Warn(format string, v ...interface{}) {
	handleLog(loggerWarn, "WARN", format, v...)
}

// Error prints logs as this format: [ERROR]
func Error(format string, v ...interface{}) {
	handleLog(loggerError, "ERROR", format, v...)
}

// Panic prints logs as this format: [PANIC]
func Panic(format string, v ...interface{}) {
	handleLog(loggerPanic, "PANIC", format, v...)
}

func handleLog(logHandle *log.Logger, severity, format string, v ...interface{}) {
	msgFormat := "[" + logName + "][" + severity + "] " + format

	checkExtrasLogPanic := func(msgFormatStr string, vv []interface{}) {
		if len(vv) > 0 {
			log.Panicf(msgFormatStr, vv...)
		} else {
			log.Panicf(msgFormatStr)
		}
	}

	checkExtrasLoggerPanic := func(internalLogHandle *log.Logger, msgFormatStr string, vv []interface{}) {
		if len(vv) > 0 {
			internalLogHandle.Panicf(msgFormatStr, vv...)
		} else {
			internalLogHandle.Panicf(msgFormatStr)
		}
	}

	checkExtrasLogPrint := func(msgFormatStr string, vv []interface{}) {
		if len(vv) > 0 {
			log.Printf(msgFormatStr, vv...)
		} else {
			log.Printf(msgFormatStr)
		}
	}

	checkExtrasLoggerPrint := func(internalLogHandle *log.Logger, msgFormatStr string, vv []interface{}) {
		if len(vv) > 0 {
			internalLogHandle.Printf(msgFormatStr, vv...)
		} else {
			internalLogHandle.Printf(msgFormatStr)
		}
	}

	if logHandle == nil {
		if severity == "PANIC" {
			checkExtrasLogPanic(msgFormat, v)
		} else {
			checkExtrasLogPrint(msgFormat, v)
		}
	} else {
		if logHandle == loggerPanic {
			checkExtrasLoggerPanic(logHandle, msgFormat, v)
		} else {
			checkExtrasLoggerPrint(logHandle, msgFormat, v)
		}
	}
}
