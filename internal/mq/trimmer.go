package mq

import (
	"context"
	"time"

	"gonio/internal/config"
	"gonio/internal/pkg/logger"

	"github.com/redis/go-redis/v9"
)

// StartTrimmer 启动后台定期 XTRIM goroutine
// 仅在 driver=redis 且 trim_interval>0 时生效
func StartTrimmer(ctx context.Context, cfg *config.MQConfig, rdb *redis.Client) {
	if cfg.Driver != "redis" || cfg.TrimInterval <= 0 {
		return
	}

	// 构建需要裁剪的 topic → maxLen 映射
	shortToTopic := GetShortToTopicMap()

	topics := map[string]int64{}
	for short, topic := range shortToTopic {
		maxLen := cfg.DefaultMaxLen
		if n, ok := cfg.TopicMaxLen[short]; ok && n > 0 {
			maxLen = int64(n)
		}
		if maxLen > 0 {
			topics[topic] = maxLen
		}
	}

	if len(topics) == 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(time.Duration(cfg.TrimInterval) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for topic, maxLen := range topics {
					n, err := rdb.XTrimMaxLen(ctx, topic, maxLen).Result()
					if err != nil {
						logger.Log.Warnw("[mq] xtrim failed", "topic", topic, "err", err)
						continue
					}
					if n > 0 {
						logger.Log.Infow("[mq] xtrim done", "topic", topic, "removed", n, "max_len", maxLen)
					}
				}
			}
		}
	}()
}
