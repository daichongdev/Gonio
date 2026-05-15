package mq

import (
	"context"
	"database/sql"
	"fmt"

	"gonio/internal/config"

	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/redis/go-redis/v9"
)

// Router 封装 watermill Router
type Router struct {
	router *message.Router
}

// NewRouter 创建消息路由并注册所有 handler
func NewRouter(cfg *config.MQConfig, rdb *redis.Client, sqlDB *sql.DB) (*Router, error) {
	log := newLogger()

	r, err := message.NewRouter(message.RouterConfig{}, log)
	if err != nil {
		return nil, fmt.Errorf("create mq router failed: %w", err)
	}

	// 创建死信队列 publisher（与消费者使用相同后端）
	poisonPub, err := newPublisher(cfg, rdb, sqlDB)
	if err != nil {
		return nil, fmt.Errorf("create poison publisher failed: %w", err)
	}

	// 失败时自动重试（最多3次），超过后转入死信队列
	r.AddMiddleware(middleware.Retry{
		MaxRetries:      3,
		InitialInterval: 100,
		Logger:          log,
	}.Middleware)

	// 超过重试次数后将消息发送到 <topic>.poison，避免无限循环
	poisonQueue, err := middleware.PoisonQueue(poisonPub, "mq.poison")
	if err != nil {
		return nil, fmt.Errorf("create poison queue middleware failed: %w", err)
	}
	r.AddMiddleware(poisonQueue)

	// 捕获 handler panic
	r.AddMiddleware(middleware.Recoverer)

	// topicConcurrency 获取某个 topic 的并发数，未配置默认为 1
	// config 中用短名（email/sms/stats），与 Topic 常量做映射
	topicShortName := GetTopicToShortMap()
	topicConcurrency := func(topic string) int {
		key := topicShortName[topic]
		if n, ok := cfg.TopicConcurrency[key]; ok && n > 0 {
			return n
		}
		return 1
	}

	// 从注册表获取所有 handler 定义
	handlers := GetHandlers()

	for _, h := range handlers {
		n := topicConcurrency(h.Topic)
		for i := range n {
			sub, err := newSubscriber(cfg, rdb, sqlDB)
			if err != nil {
				return nil, fmt.Errorf("create subscriber for %s[%d]: %w", h.Name, i, err)
			}
			handlerName := fmt.Sprintf("%s_handler_%d", h.Name, i)
			r.AddConsumerHandler(handlerName, h.Topic, sub, h.Handler)
		}
	}

	return &Router{router: r}, nil
}

// Run 启动路由（阻塞，需在 goroutine 中调用）
func (r *Router) Run(ctx context.Context) error {
	return r.router.Run(ctx)
}

// Close 关闭路由
func (r *Router) Close() error {
	return r.router.Close()
}

// newPublisher 创建底层 publisher（供内部使用）
func newPublisher(cfg *config.MQConfig, rdb *redis.Client, sqlDB *sql.DB) (message.Publisher, error) {
	l := newLogger()
	switch cfg.Driver {
	case "redis":
		return redisstream.NewPublisher(redisstream.PublisherConfig{Client: rdb}, l)
	case "mysql":
		return watermillsql.NewPublisher(sqlDB, watermillsql.PublisherConfig{
			SchemaAdapter: watermillsql.DefaultMySQLSchema{},
		}, l)
	default:
		return nil, fmt.Errorf("unsupported mq driver: %s", cfg.Driver)
	}
}

// newSubscriber 根据配置创建 subscriber
func newSubscriber(cfg *config.MQConfig, rdb *redis.Client, sqlDB *sql.DB) (message.Subscriber, error) {
	l := newLogger()
	switch cfg.Driver {
	case "redis":
		return redisstream.NewSubscriber(
			redisstream.SubscriberConfig{
				Client:        rdb,
				ConsumerGroup: cfg.ConsumerGroup,
			},
			l,
		)
	case "mysql":
		return watermillsql.NewSubscriber(
			sqlDB,
			watermillsql.SubscriberConfig{
				SchemaAdapter:    watermillsql.DefaultMySQLSchema{},
				OffsetsAdapter:   watermillsql.DefaultMySQLOffsetsAdapter{},
				InitializeSchema: true,
			},
			l,
		)
	default:
		return nil, fmt.Errorf("unsupported mq driver: %s", cfg.Driver)
	}
}
