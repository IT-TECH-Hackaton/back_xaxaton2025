package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Logger struct {
	infoLog  *log.Logger
	errorLog *log.Logger
	warnLog  *log.Logger
}

var defaultLogger *Logger

func Init() {
	defaultLogger = &Logger{
		infoLog:  log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLog: log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile),
		warnLog:  log.New(os.Stdout, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func Info(format string, v ...interface{}) {
	if defaultLogger == nil {
		log.Printf("[INFO] "+format, v...)
		return
	}
	defaultLogger.infoLog.Printf(format, v...)
}

func Error(format string, v ...interface{}) {
	if defaultLogger == nil {
		log.Printf("[ERROR] "+format, v...)
		return
	}
	defaultLogger.errorLog.Printf(format, v...)
}

func Warn(format string, v ...interface{}) {
	if defaultLogger == nil {
		log.Printf("[WARN] "+format, v...)
		return
	}
	defaultLogger.warnLog.Printf(format, v...)
}

func WithFields(level string, fields map[string]interface{}) {
	timestamp := time.Now().Format(time.RFC3339)
	msg := "[" + level + "] [" + timestamp + "]"
	for k, v := range fields {
		msg += " " + k + "=" + formatValue(v)
	}
	log.Println(msg)
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%.2f", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
