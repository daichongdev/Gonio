package logger

import (
	"context"
	"os"
	"path/filepath"

	"goflow/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/lumberjack.v2"
)

var Log *zap.SugaredLogger

type contextKey string

const RequestIDKey contextKey = "request_id"

// init 提供一个开箱即用的 fallback logger，避免 Init 调用前 Log 为 nil 导致 panic
func init() {
	Log = zap.Must(zap.NewProduction()).Sugar()
}

func Init(cfg *config.LogConfig) {
	// 编码器配置
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	// 日志级别
	level := zapcore.DebugLevel
	switch cfg.Level {
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	maxSize := cfg.MaxSize
	if maxSize == 0 {
		maxSize = 100
	}
	maxBackups := cfg.MaxBackups
	if maxBackups == 0 {
		maxBackups = 7
	}
	maxAge := cfg.MaxAge
	if maxAge == 0 {
		maxAge = 30
	}

	newFileWriter := func(filename string) zapcore.WriteSyncer {
		return zapcore.AddSync(&lumberjack.Logger{
			Filename:   filepath.Join(cfg.LogDir, filename),
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   cfg.Compress,
		})
	}

	var cores []zapcore.Core

	// 控制台输出：所有级别
	if cfg.Mode == "dev" {
		devcfg := encoderCfg
		devcfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cores = append(cores, zapcore.NewCore(
			zapcore.NewConsoleEncoder(devcfg),
			zapcore.AddSync(os.Stdout),
			level,
		))
	} else {
		cores = append(cores, zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.AddSync(os.Stdout),
			level,
		))
	}

	// 文件输出：按级别分文件
	if cfg.LogDir != "" {
		jsonEncoder := zapcore.NewJSONEncoder(encoderCfg)

		// app.log：所有级别
		cores = append(cores, zapcore.NewCore(
			jsonEncoder,
			newFileWriter("app.log"),
			level,
		))

		// warn.log：仅 Warn 级别
		cores = append(cores, zapcore.NewCore(
			jsonEncoder,
			newFileWriter("warn.log"),
			zap.LevelEnablerFunc(func(l zapcore.Level) bool {
				return l == zapcore.WarnLevel
			}),
		))

		// error.log：Error 及以上
		cores = append(cores, zapcore.NewCore(
			jsonEncoder,
			newFileWriter("error.log"),
			zap.LevelEnablerFunc(func(l zapcore.Level) bool {
				return l >= zapcore.ErrorLevel
			}),
		))
	}

	zapLog := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(0))
	Log = zapLog.Sugar()
}

// WithCtx 从 context 中提取 request_id 附加到日志
func WithCtx(ctx context.Context) *zap.SugaredLogger {
	if Log == nil {
		return zap.NewNop().Sugar()
	}
	if ctx == nil {
		return Log
	}
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		return Log.With("request_id", reqID)
	}
	return Log
}

// Sync 刷新日志缓冲
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
