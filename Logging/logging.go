package logging

import (
	"fmt"
	"os"
	"time"
)

var logFile *os.File

func Init() {
	var err error
	logFile, err = os.OpenFile("appkiller.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		logFile = nil
	}
}

func Info(format string, args ...interface{}) {
	logWithLevel("INFO", format, args...)
}

func Warning(format string, args ...interface{}) {
	logWithLevel("WARN", format, args...)
}

func Error(format string, args ...interface{}) {
	logWithLevel("ERROR", format, args...)
}

func logWithLevel(level, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("%s %s: %s\n", timestamp, level, msg)
	fmt.Fprint(os.Stderr, logLine)
	if logFile != nil {
		logFile.WriteString(logLine)
	}
}
