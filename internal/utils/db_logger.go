package utils

import (
	"context"
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

	// If no patterns matched, pass to the original logger
	l.Interface.Trace(ctx, begin, func() (string, int64) {
		return sql, rows
	}, err)
}
