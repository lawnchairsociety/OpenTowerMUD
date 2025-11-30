package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Custom log level for messages that should always be logged
const LevelAlways = slog.Level(12) // Higher than Error (8), ensures it's always logged

var (
	logger *slog.Logger
)

// Initialize sets up the logger with the provided configuration
func Initialize(config Config) error {
	var handlers []slog.Handler

	// Parse the log level
	level := parseLogLevel(config.Level)

	// Console handler
	if config.ConsoleEnabled {
		var consoleHandler slog.Handler
		opts := &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Customize ALWAYS level display
				if a.Key == slog.LevelKey {
					if level, ok := a.Value.Any().(slog.Level); ok && level == LevelAlways {
						a.Value = slog.StringValue("ALWAYS")
					}
				}
				return a
			},
		}

		if config.ConsoleFormat == "json" {
			consoleHandler = slog.NewJSONHandler(os.Stdout, opts)
		} else {
			consoleHandler = slog.NewTextHandler(os.Stdout, opts)
		}
		handlers = append(handlers, consoleHandler)
	}

	// File handler
	if config.FileEnabled {
		// Create lumberjack logger for log rotation
		logFile := &lumberjack.Logger{
			Filename:   config.FilePath,
			MaxSize:    config.FileMaxSizeMB,
			MaxBackups: config.FileMaxBackups,
			MaxAge:     config.FileMaxAgeDays,
			Compress:   false, // Don't compress old logs by default
		}

		var fileHandler slog.Handler
		opts := &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Customize ALWAYS level display
				if a.Key == slog.LevelKey {
					if level, ok := a.Value.Any().(slog.Level); ok && level == LevelAlways {
						a.Value = slog.StringValue("ALWAYS")
					}
				}
				return a
			},
		}

		if config.FileFormat == "json" {
			fileHandler = slog.NewJSONHandler(logFile, opts)
		} else {
			fileHandler = slog.NewTextHandler(logFile, opts)
		}
		handlers = append(handlers, fileHandler)
	}

	// If no handlers configured, use default console handler
	if len(handlers) == 0 {
		handlers = append(handlers, slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	}

	// Create multi-handler if we have multiple outputs
	if len(handlers) == 1 {
		logger = slog.New(handlers[0])
	} else {
		logger = slog.New(newMultiHandler(handlers...))
	}

	return nil
}

// parseLogLevel converts a string log level to slog.Level
func parseLogLevel(level string) slog.Level {
	switch level {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARNING", "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo // Default to INFO
	}
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	if logger != nil {
		logger.Debug(msg, args...)
	}
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...any) {
	Debug(fmt.Sprintf(format, args...))
}

// Info logs an info message
func Info(msg string, args ...any) {
	if logger != nil {
		logger.Info(msg, args...)
	}
}

// Infof logs a formatted info message
func Infof(format string, args ...any) {
	Info(fmt.Sprintf(format, args...))
}

// Warning logs a warning message
func Warning(msg string, args ...any) {
	if logger != nil {
		logger.Warn(msg, args...)
	}
}

// Warningf logs a formatted warning message
func Warningf(format string, args ...any) {
	Warning(fmt.Sprintf(format, args...))
}

// Error logs an error message
func Error(msg string, args ...any) {
	if logger != nil {
		logger.Error(msg, args...)
	}
}

// Errorf logs a formatted error message
func Errorf(format string, args ...any) {
	Error(fmt.Sprintf(format, args...))
}

// Always logs a message that bypasses log level filtering
// This is used for audit trails (e.g., chat messages) that must always be logged
func Always(msg string, args ...any) {
	if logger != nil {
		logger.Log(nil, LevelAlways, msg, args...)
	}
}

// Alwaysf logs a formatted message that bypasses log level filtering
func Alwaysf(format string, args ...any) {
	Always(fmt.Sprintf(format, args...))
}

// multiHandler is a handler that writes to multiple underlying handlers
type multiHandler struct {
	handlers []slog.Handler
}

// newMultiHandler creates a new multi-handler
func newMultiHandler(handlers ...slog.Handler) *multiHandler {
	return &multiHandler{handlers: handlers}
}

// Enabled reports whether the handler handles records at the given level
func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// If any handler is enabled for this level, return true
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle handles the Record
func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	// Write to all handlers
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments
func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return newMultiHandler(handlers...)
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups
func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return newMultiHandler(handlers...)
}
