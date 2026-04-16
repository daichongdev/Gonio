package logger

import (
	"context"
	"os"
	"path/filepath"

	"gonio/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

var (
	// Log 是全局 SugaredLogger，方便开发使用
	Log *zap.SugaredLogger
	// RawLog 是全局 zap.Logger，高性能场景使用
	RawLog *zap.Logger
)

// init 提供一个开箱即用的 fallback logger
func init() {
	RawLog = zap.Must(zap.NewProduction())
	Log = RawLog.Sugar()
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
		newBufferedFileWriter := func(filename string) zapcore.WriteSyncer {
			return &zapcore.BufferedWriteSyncer{WS: newFileWriter(filename)}
		}

		// app.log：所有级别
		cores = append(cores, zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			newBufferedFileWriter("app.log"),
			level,
		))

		// warn.log：仅 Warn 级别
		cores = append(cores, zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			newBufferedFileWriter("warn.log"),
			zap.LevelEnablerFunc(func(l zapcore.Level) bool {
				return l == zapcore.WarnLevel
			}),
		))

		// error.log：Error 及以上
		cores = append(cores, zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			newBufferedFileWriter("error.log"),
			zap.LevelEnablerFunc(func(l zapcore.Level) bool {
				return l >= zapcore.ErrorLevel
			}),
		))
	}

	zapLog := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(1))
	RawLog = zapLog
	Log = zapLog.Sugar()
}

// WithCtx 从 context 中提取 request_id 附加到日志
func WithCtx(ctx context.Context) *zap.SugaredLogger {
	if RawLog == nil {
		return zap.NewNop().Sugar()
	}
	if ctx == nil {
		return Log
	}
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		// 直接使用 RawLog.With 构造，效率更高
		return RawLog.With(zap.String("request_id", reqID)).Sugar()
	}
	return Log
}

// Sync 刷新日志缓冲
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
