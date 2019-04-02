package logger

import (
	"log"

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
)

func init() {
	log.Print("DJFKLJDLFJLD")
	ctx := context.Background()

	// Sets your Google Cloud Platform project ID.
	projectID := "ticklemeta-203110"

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

	// Sets the name of the log to write to.
	logName := "Collector"

	loggerInfo = client.Logger(logName).StandardLogger(logging.Info)
	loggerWarn = client.Logger(logName).StandardLogger(logging.Warning)
	loggerError = client.Logger(logName).StandardLogger(logging.Error)
	loggerPanic = client.Logger(logName).StandardLogger(logging.Critical)

	isLoggerGCE = true

	// Logs "hello world", log entry is visible at
	// Stackdriver Logs.
	loggerInfo.Println("Logger init")
}

// IsLoggerGCE provides interface if current logger is GCE
func IsLoggerGCE() bool {
	return isLoggerGCE
}

// Close closes GCE Client
func Close() {
	loggerClient.Close()
}

// Info prints logs as this format: [INFO]
func Info(format string, v ...interface{}) {
	if loggerInfo == nil {
		log.Printf("[INFO] "+format, v)
	} else {
		if len(v) > 0 {
			loggerInfo.Printf("[INFO] "+format, v...)
		} else {
			loggerInfo.Printf("[INFO] " + format)
		}
	}
}

// Warn prints logs as this format: [WARN]
func Warn(format string, v ...interface{}) {
	if loggerWarn == nil {
		log.Printf("[WARN] "+format, v)
	} else {
		if len(v) > 0 {
			loggerWarn.Printf("[WARN] "+format, v...)
		} else {
			loggerWarn.Printf("[WARN] " + format)
		}
	}
}

// Error prints logs as this format: [ERROR]
func Error(format string, v ...interface{}) {
	if loggerError == nil {
		log.Printf("[ERROR] "+format, v)
	} else {
		if len(v) > 0 {
			loggerError.Printf("[ERROR] "+format, v...)
		} else {
			loggerError.Printf("[ERROR] " + format)
		}
	}
}

// Panic prints logs as this format: [PANIC]
func Panic(format string, v ...interface{}) {
	if loggerPanic == nil {
		log.Panicf("[PANIC] "+format, v)
	} else {
		if len(v) > 0 {
			loggerPanic.Panicf("[PANIC] "+format, v...)
		} else {
			loggerPanic.Panicf("[PANIC] " + format)
		}
	}
}
