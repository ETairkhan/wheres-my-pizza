package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"
)

type LogEntry struct {
	Timestamp string      `json:"timestamp"`
	Level     string      `json:"level"`
	Service   string      `json:"service"`
	Action    string      `json:"action"`
	Message   string      `json:"message"`
	Hostname  string      `json:"hostname"`
	RequestID string      `json:"request_id"`
	Error     *ErrorEntry `json:"error,omitempty"`
}

type ErrorEntry struct {
	Msg   string `json:"msg"`
	Stack string `json:"stack"`
}

type Logger struct {
	service  string
	hostname string
}

func NewLogger(service string) *Logger {
	hostname, _ := os.Hostname()
	return &Logger{
		service:  service,
		hostname: hostname,
	}
}

func (l *Logger) Info(requestID, action, message string) {
	l.log("INFO", requestID, action, message, nil)
}

func (l *Logger) Debug(requestID, action, message string) {
	l.log("DEBUG", requestID, action, message, nil)
}

func (l *Logger) Error(requestID, action, message string, err error) {
	stack := ""
	if err != nil {
		buf := make([]byte, 1024)
		n := runtime.Stack(buf, false)
		stack = string(buf[:n])
	}

	errorEntry := &ErrorEntry{
		Msg:   err.Error(),
		Stack: stack,
	}
	l.log("ERROR", requestID, action, message, errorEntry)
}

func (l *Logger) log(level, requestID, action, message string, errorEntry *ErrorEntry) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Service:   l.service,
		Action:    action,
		Message:   message,
		Hostname:  l.hostname,
		RequestID: requestID,
		Error:     errorEntry,
	}

	jsonData, _ := json.Marshal(entry)
	fmt.Println(string(jsonData))
}
