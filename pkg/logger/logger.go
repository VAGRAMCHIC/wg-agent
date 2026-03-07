package logger

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type ctxKey string

const RequestIDKey ctxKey = "request_id"

type Logger struct {
	logger *log.Logger
}

func New(logFilePath string) (*Logger, error) {
	var writers []io.Writer

	writers = append(writers, os.Stdout)

	if logFilePath != "" {
		dir := filepath.Dir(logFilePath)

		// создаём папку если нет
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}

		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}

		writers = append(writers, file)
	}

	multiWriter := io.MultiWriter(writers...)

	return &Logger{
		logger: log.New(multiWriter, "", 0),
	}, nil
}

func (l *Logger) log(level string, msg string, fields map[string]interface{}) {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   msg,
	}

	for k, v := range fields {
		entry[k] = v
	}

	b, _ := json.Marshal(entry)
	l.logger.Println(string(b))
}

func (l *Logger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	l.logWithContext(ctx, "INFO", msg, fields)
}

func (l *Logger) Error(ctx context.Context, msg string, fields map[string]interface{}) {
	l.logWithContext(ctx, "ERROR", msg, fields)
}

func (l *Logger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	l.logWithContext(ctx, "DEBUG", msg, fields)
}

func (l *Logger) Fatal(ctx context.Context, msg string, fields map[string]interface{}) {
	l.logWithContext(ctx, "FATAL", msg, fields)
	os.Exit(1)
}

func (l *Logger) Panic(ctx context.Context, msg string, fields map[string]interface{}) {
	l.logWithContext(ctx, "PANIC", msg, fields)
	panic(msg)
}

func (l *Logger) logWithContext(ctx context.Context, level string, msg string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}

	if ctx != nil {
		if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
			fields["request_id"] = requestID
		}
	}

	l.log(level, msg, fields)
}
