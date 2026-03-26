package mq

import (
	"strings"

	"goflow/internal/pkg/logger"

	"github.com/ThreeDotsLabs/watermill"
	"go.uber.org/zap"
)

// zapLogger 适配 Watermill 的 LoggerAdapter 接口，转接到项目 zap logger
type zapLogger struct {
	sugar *zap.SugaredLogger
}

func newLogger() watermill.LoggerAdapter {
	return &zapLogger{sugar: logger.Log}
}

func (z *zapLogger) Error(msg string, err error, fields watermill.LogFields) {
	// 多个 subscriber 共享同一 Redis client，关闭时后续 subscriber 会收到
	// "redis: client is closed" 错误，属于预期行为，降级为 debug 日志
	if err != nil && strings.Contains(err.Error(), "client is closed") {
		z.sugar.Debugw(msg, append(fieldsToArgs(fields), "error", err)...)
		return
	}
	args := fieldsToArgs(fields)
	if err != nil {
		args = append(args, "error", err)
	}
	z.sugar.Errorw(msg, args...)
}

func (z *zapLogger) Info(msg string, fields watermill.LogFields) {
	z.sugar.Infow(msg, fieldsToArgs(fields)...)
}

func (z *zapLogger) Debug(msg string, fields watermill.LogFields) {
	z.sugar.Debugw(msg, fieldsToArgs(fields)...)
}

func (z *zapLogger) Trace(msg string, fields watermill.LogFields) {
	z.sugar.Debugw("[trace] "+msg, fieldsToArgs(fields)...)
}

// With 正确实现字段附加，返回新的 logger 实例携带上下文字段
func (z *zapLogger) With(fields watermill.LogFields) watermill.LoggerAdapter {
	if len(fields) == 0 {
		return z
	}
	return &zapLogger{sugar: z.sugar.With(fieldsToArgs(fields)...)}
}

func fieldsToArgs(fields watermill.LogFields) []interface{} {
	if len(fields) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return args
}
