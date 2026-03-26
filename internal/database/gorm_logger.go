package database

import (
	"context"
	"fmt"
	"time"

	"goflow/internal/pkg/logger"

	"go.uber.org/zap"
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

	// 获取高性能 zap.Logger，并注入 request_id
	var rawLog *zap.Logger
	if reqID, ok := ctx.Value(logger.RequestIDKey).(string); ok && reqID != "" {
		rawLog = logger.RawLog.With(zap.String("request_id", reqID))
	} else {
		rawLog = logger.RawLog
	}

	switch {
	case err != nil && l.level >= gormlogger.Error:
		rawLog.Error("[sql]",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Int64("elapsed_ms", elapsed.Milliseconds()),
			zap.Error(err),
		)
	case elapsed >= l.slowThreshold && l.level >= gormlogger.Warn:
		rawLog.Warn("[sql] slow query",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Int64("elapsed_ms", elapsed.Milliseconds()),
		)
	case l.level >= gormlogger.Info:
		rawLog.Debug("[sql]",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Int64("elapsed_ms", elapsed.Milliseconds()),
		)
	}
}
