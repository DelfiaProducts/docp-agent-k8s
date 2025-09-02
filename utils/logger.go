package utils

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// K8sLogger is struct for logger the k8s
type K8sLogger struct {
	logger *slog.Logger
}

func getLogLevel() slog.Level {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// NewK8sLoggerJSON return instance of k8s logger
// with formatter json
func NewK8sLoggerJSON(writter io.Writer) *K8sLogger {
	return &K8sLogger{
		logger: slog.New(slog.NewJSONHandler(writter, &slog.HandlerOptions{
			Level: getLogLevel(),
		})),
	}
}

// NewK8sLoggerText return instance of k8s logger
// with formatter text
func NewK8sLoggerText(writter io.Writer) *K8sLogger {
	return &K8sLogger{
		logger: slog.New(slog.NewTextHandler(writter, &slog.HandlerOptions{
			Level: getLogLevel(),
		})),
	}
}

// Debug execute logging the debug
func (k *K8sLogger) Debug(msg string, args ...any) {
	k.logger.Debug(msg, args...)
}

// Info execute logging the info
func (k *K8sLogger) Info(msg string, args ...any) {
	k.logger.Info(msg, args...)
}

// Warn execute logging the warning
func (k *K8sLogger) Warn(msg string, args ...any) {
	k.logger.Warn(msg, args...)
}

// Error execut logging the error
func (k *K8sLogger) Error(msg string, args ...any) {
	k.logger.Error(msg, args...)
}
