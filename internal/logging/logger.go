package logging

import (
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger represents a structured logger
type Logger struct {
	component string
	level     LogLevel
}

// New creates a new logger with a component name
func New(component string) *Logger {
	return &Logger{
		component: component,
		level:     INFO, // Default level
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...Field) {
	l.log(DEBUG, msg, fields...)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...Field) {
	l.log(INFO, msg, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...Field) {
	l.log(WARN, msg, fields...)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...Field) {
	l.log(ERROR, msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields ...Field) {
	l.log(FATAL, msg, fields...)
	os.Exit(1)
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Error creates an error field
func Error(err error) Field {
	return Field{Key: "error", Value: err.Error()}
}

// Duration creates a duration field
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// log performs the actual logging
func (l *Logger) log(level LogLevel, msg string, fields ...Field) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logLine := fmt.Sprintf("[%s] %s [%s] %s", timestamp, level.String(), l.component, msg)

	// Add fields if any
	if len(fields) > 0 {
		logLine += " |"
		for _, field := range fields {
			logLine += fmt.Sprintf(" %s=%v", field.Key, field.Value)
		}
	}

	log.Println(logLine)
}

// Global logger instances for different components
var (
	ServerLogger   = New("mcp-server")
	ToolLogger     = New("tools")
	ResourceLogger = New("resources")
	DBLogger       = New("database")
	LokiLogger     = New("loki")
	S3Logger       = New("s3")
	LLMLogger      = New("llm")
)

// SetGlobalLevel sets the log level for all loggers
func SetGlobalLevel(level LogLevel) {
	ServerLogger.SetLevel(level)
	ToolLogger.SetLevel(level)
	ResourceLogger.SetLevel(level)
	DBLogger.SetLevel(level)
	LokiLogger.SetLevel(level)
	S3Logger.SetLevel(level)
	LLMLogger.SetLevel(level)
}

// EnableDebugMode enables debug logging for all components
func EnableDebugMode() {
	SetGlobalLevel(DEBUG)
}
