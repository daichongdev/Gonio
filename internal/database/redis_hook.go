package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"goflow/internal/config"
	"goflow/internal/pkg/logger"

	"github.com/redis/go-redis/v9"
)

// isExpectedErr 过滤掉正常业务/生命周期语义的 Redis 错误，避免日志噪音：
//   - redis.Nil：key 不存在或 block 轮询超时无消息
//   - context.Canceled：优雅关闭时主动取消
//   - context.DeadlineExceeded：请求超时，由上层处理
//   - BUSYGROUP：服务重启时消费者组已存在，watermill 幂等创建的正常副作用
func isExpectedErr(err error) bool {
	if errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return err != nil && strings.HasPrefix(err.Error(), "BUSYGROUP")
}

// redisLogHook 在 debug 模式下记录每次 Redis 命令的耗时和 request_id
type redisLogHook struct {
	enabled bool
}

func newRedisLogHook(logCfg *config.LogConfig) *redisLogHook {
	return &redisLogHook{enabled: logCfg.Level == "debug"}
}

func (h *redisLogHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *redisLogHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmd)
		if !h.enabled {
			return err
		}
		elapsed := time.Since(start)
		args := fmt.Sprintf("%v", cmd.Args())
		// redis.Nil 表示 key 不存在或 block 超时，是正常业务语义，不记录为错误
		if err != nil && !isExpectedErr(err) {
			logger.WithCtx(ctx).Errorw("[redis]", "cmd", args, "elapsed_ms", elapsed.Milliseconds(), "error", err)
		} else if err == nil {
			logger.WithCtx(ctx).Debugw("[redis]", "cmd", args, "elapsed_ms", elapsed.Milliseconds())
		}
		return err
	}
}

func (h *redisLogHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		start := time.Now()
		err := next(ctx, cmds)
		if !h.enabled {
			return err
		}
		elapsed := time.Since(start)
		for _, cmd := range cmds {
			args := fmt.Sprintf("%v", cmd.Args())
			if cmd.Err() != nil && !errors.Is(cmd.Err(), redis.Nil) {
				logger.WithCtx(ctx).Errorw("[redis pipeline]", "cmd", args, "elapsed_ms", elapsed.Milliseconds(), "error", cmd.Err())
			} else if cmd.Err() == nil {
				logger.WithCtx(ctx).Debugw("[redis pipeline]", "cmd", args, "elapsed_ms", elapsed.Milliseconds())
			}
		}
		return err
	}
}
