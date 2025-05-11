package utils

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"gorm.io/gorm/logger"
)

// CustomGormLogger is a custom logger for GORM that filters out specific queries
type CustomGormLogger struct {
	logger.Interface
	ignoredQueryPatterns []string
}

// NewCustomGormLogger creates a new custom logger with the given ignored query patterns
func NewCustomGormLogger(l logger.Interface, ignoredPatterns ...string) *CustomGormLogger {
	return &CustomGormLogger{
		Interface:            l,
		ignoredQueryPatterns: ignoredPatterns,
	}
}

// LogMode implements logger.Interface
func (l *CustomGormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := l.Interface.LogMode(level)
	return &CustomGormLogger{
		Interface:            newLogger,
		ignoredQueryPatterns: l.ignoredQueryPatterns,
	}
}

// Trace implements logger.Interface
func (l *CustomGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rows := fc()

	// Skip logging if the SQL matches any of the ignored patterns
	for _, pattern := range l.ignoredQueryPatterns {
		if strings.Contains(sql, pattern) {
			return
		}
	}

	// Find the caller in the application code by examining the stack
	callerInfo := findCaller()

	// Create a wrapper function that includes caller information
	wrappedFC := func() (string, int64) {
		if callerInfo != "" {
			return fmt.Sprintf("[Caller: %s] %s", callerInfo, sql), rows
		}
		return sql, rows
	}

	// If no patterns matched, pass to the original logger with caller info
	l.Interface.Trace(ctx, begin, wrappedFC, err)
}

// findCaller looks through the call stack to find the first non-GORM, non-database caller
func findCaller() string {
	// Start from a depth that skips immediate callers (GORM internals)
	for i := 2; i < 10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		// Skip GORM internal packages and our own database package
		if strings.Contains(file, "gorm.io") ||
			strings.Contains(file, "internal/database") ||
			strings.Contains(file, "internal/utils/db_logger.go") {
			continue
		}

		// Get function name if possible
		funcName := ""
		if fn := runtime.FuncForPC(pc); fn != nil {
			funcName = fn.Name()
			// Extract just the function name without the full package path
			if idx := strings.LastIndexByte(funcName, '.'); idx != -1 {
				funcName = funcName[idx+1:]
			}
		}

		// Return a descriptive caller string
		if funcName != "" {
			return fmt.Sprintf("%s() at %s:%d", funcName, file, line)
		}
		return fmt.Sprintf("%s:%d", file, line)
	}

	return ""
}
