package logger

import (
	"log"
	"os"
	"path/filepath"
)

var (
	activityLogger *log.Logger
	errorLogger    *log.Logger
)

func init() {
	projectRoot, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	logDir := filepath.Join(projectRoot, "logs")
	_ = os.MkdirAll(logDir, os.ModePerm)

	activityFile, err := os.OpenFile(filepath.Join(logDir, "activity.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open activity log: %v", err)
	}

	errorFile, err := os.OpenFile(filepath.Join(logDir, "error.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open error log: %v", err)
	}

	activityLogger = log.New(activityFile, "[ACTIVITY] ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger = log.New(errorFile, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Info logs to activity.log and stdout
func Info(msg string) {
	activityLogger.Println("[INFO] " + msg)
}

// Error logs to error.log and stdout
func Error(msg string, err error) {
	errorLogger.Printf("[ERROR] %s: %v\n", msg, err)
}
