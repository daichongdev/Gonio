package database

import (
	"context"
	"fmt"
	"time"

	"goflow/internal/config"
	"goflow/internal/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func InitRedis(cfg *config.RedisConfig, logCfg *config.LogConfig) (*redis.Client, error) {
	// 设置超时默认值
	dialTimeout := time.Duration(cfg.DialTimeout) * time.Second
	if cfg.DialTimeout <= 0 {
		dialTimeout = 5 * time.Second
	}
	readTimeout := time.Duration(cfg.ReadTimeout) * time.Second
	if cfg.ReadTimeout <= 0 {
		readTimeout = 3 * time.Second
	}
	writeTimeout := time.Duration(cfg.WriteTimeout) * time.Second
	if cfg.WriteTimeout <= 0 {
		writeTimeout = 3 * time.Second
	}
	poolTimeout := time.Duration(cfg.PoolTimeout) * time.Second
	if cfg.PoolTimeout <= 0 {
		poolTimeout = dialTimeout + time.Second // 略大于 dialTimeout
	}
	pingTimeout := time.Duration(cfg.PingTimeout) * time.Second
	if cfg.PingTimeout <= 0 {
		pingTimeout = 3 * time.Second
	}

	// 连接池默认值
	poolSize := cfg.PoolSize
	if poolSize <= 0 {
		poolSize = 100
	}
	minIdleConns := cfg.MinIdleConns
	if minIdleConns <= 0 {
		minIdleConns = 10
	}
	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 50
	}

	// 连接生命周期默认值
	connMaxIdleTime := time.Duration(cfg.ConnMaxIdleTime) * time.Second
	if cfg.ConnMaxIdleTime <= 0 {
		connMaxIdleTime = 5 * time.Minute
	}
	connMaxLifetime := time.Duration(cfg.ConnMaxLifetime) * time.Second
	if cfg.ConnMaxLifetime <= 0 {
		connMaxLifetime = 1 * time.Hour
	}

	// 重试策略默认值
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	minRetryBackoff := time.Duration(cfg.MinRetryBackoff) * time.Millisecond
	if cfg.MinRetryBackoff <= 0 {
		minRetryBackoff = 8 * time.Millisecond
	}
	maxRetryBackoff := time.Duration(cfg.MaxRetryBackoff) * time.Millisecond
	if cfg.MaxRetryBackoff <= 0 {
		maxRetryBackoff = 512 * time.Millisecond
	}

	opts := &redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,

		// 连接池
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
		MaxIdleConns: maxIdleConns,
		PoolTimeout:  poolTimeout,

		// 超时
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,

		// 连接生命周期
		ConnMaxIdleTime: connMaxIdleTime,
		ConnMaxLifetime: connMaxLifetime,

		// 自动重试：网络抖动时自动重试，避免单次失败向上层透传
		MaxRetries:      maxRetries,
		MinRetryBackoff: minRetryBackoff,
		MaxRetryBackoff: maxRetryBackoff,

		// 让 context 的 deadline 自动作用于连接读写超时
		ContextTimeoutEnabled: true,
	}

	rdb := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	logger.Log.Info("redis connected successfully",
		zap.String("addr", cfg.Addr),
		zap.Int("pool_size", poolSize),
		zap.Int("min_idle_conns", minIdleConns),
		zap.Int("max_idle_conns", maxIdleConns),
		zap.Duration("dial_timeout", dialTimeout),
		zap.Duration("read_timeout", readTimeout),
		zap.Duration("write_timeout", writeTimeout),
		zap.Duration("conn_max_idle_time", connMaxIdleTime),
		zap.Duration("conn_max_lifetime", connMaxLifetime),
		zap.Int("max_retries", maxRetries),
	)
	rdb.AddHook(newRedisLogHook(logCfg))
	return rdb, nil
}

// CloseRedis 关闭 Redis 连接，接收显式参数避免全局状态
func CloseRedis(rdb *redis.Client) {
	if rdb != nil {
		_ = rdb.Close()
	}
}
