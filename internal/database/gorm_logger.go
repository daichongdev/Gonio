package database

import (
	"context"
	"fmt"
	"time"

	"goflow/internal/pkg/logger"

	gormlogger "gorm.io/gorm/logger"
)

// zapGormLogger 自定义 GORM logger，将 SQL 日志通过 zap 输出并携带 request_id
type zapGormLogger struct {
	level         gormlogger.LogLevel
	slowThreshold time.Duration
}

func newZapGormLogger(level gormlogger.LogLevel) gormlogger.Interface {
	return &zapGormLogger{
		level:         level,
		slowThreshold: 200 * time.Millisecond,
	}
}

func (l *zapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &zapGormLogger{level: level, slowThreshold: l.slowThreshold}
}

func (l *zapGormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Info {
		logger.WithCtx(ctx).Infow("[gorm] " + fmt.Sprintf(msg, args...))
	}
}

func (l *zapGormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Warn {
		logger.WithCtx(ctx).Warnw("[gorm] " + fmt.Sprintf(msg, args...))
	}
}

func (l *zapGormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.level >= gormlogger.Error {
		logger.WithCtx(ctx).Errorw("[gorm] " + fmt.Sprintf(msg, args...))
	}
}

func (l *zapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	log := logger.WithCtx(ctx)

	switch {
	case err != nil && l.level >= gormlogger.Error:
		log.Errorw("[sql]", "sql", sql, "rows", rows, "elapsed_ms", elapsed.Milliseconds(), "error", err)
	case elapsed >= l.slowThreshold && l.level >= gormlogger.Warn:
		log.Warnw("[sql] slow query", "sql", sql, "rows", rows, "elapsed_ms", elapsed.Milliseconds())
	case l.level >= gormlogger.Info:
		log.Debugw("[sql]", "sql", sql, "rows", rows, "elapsed_ms", elapsed.Milliseconds())
	}
}
