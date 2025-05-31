package logging

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

var logRotator *lumberjack.Logger

func Init() {
	logPath := "appkiller.log"

	logPath = filepath.Join(os.Getenv("ProgramData"), "AppKiller", "appkiller.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)

	logRotator = &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    1,  // megabytes before rotation
		MaxBackups: 3,  // number of backups to keep
		MaxAge:     30, // days to keep backups
		Compress:   false,
	}
}

func Show() error {
	if logRotator == nil {
		return fmt.Errorf("logging not initialized")
	}

	logFilePath := logRotator.Filename
	if logFilePath == "" {
		return fmt.Errorf("log file path is empty")
	}

	cmd := exec.Command("notepad.exe", logFilePath)
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("could not open log file: %w", err)
	}

	return nil
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

	if logRotator != nil {
		logRotator.Write([]byte(logLine))
	}
}

func Close() {
	if logRotator != nil {
		logRotator.Close()
	}
}
