package mq

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"goflow/internal/config"
	"goflow/internal/pkg/logger"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
)

// Publisher 封装 watermill 发布者
type Publisher struct {
	pub message.Publisher
}

// NewPublisher 根据配置创建 Publisher（支持 redis / mysql）
func NewPublisher(cfg *config.MQConfig, rdb *redis.Client, sqlDB *sql.DB) (*Publisher, error) {
	wlog := newLogger()
	var pub message.Publisher
	var err error

	switch cfg.Driver {
	case "redis":
		// 构建 per-topic maxlen 映射（short name → full topic name）
		topicMaxLens := make(map[string]int64, len(cfg.TopicMaxLen))
		shortToTopic := map[string]string{
			"email": TopicEmail,
			"sms":   TopicSMS,
			"stats": TopicStats,
		}
		for short, maxLen := range cfg.TopicMaxLen {
			if topic, ok := shortToTopic[short]; ok {
				topicMaxLens[topic] = int64(maxLen)
			}
		}
		pub, err = redisstream.NewPublisher(
			redisstream.PublisherConfig{
				Client:        rdb,
				DefaultMaxlen: cfg.DefaultMaxLen,
				Maxlens:       topicMaxLens,
			},
			wlog,
		)
	case "mysql":
		pub, err = watermillsql.NewPublisher(
			sqlDB,
			watermillsql.PublisherConfig{
				SchemaAdapter:        watermillsql.DefaultMySQLSchema{},
				AutoInitializeSchema: true,
			},
			wlog,
		)
	default:
		return nil, fmt.Errorf("unsupported mq driver: %s", cfg.Driver)
	}
	if err != nil {
		return nil, fmt.Errorf("create mq publisher failed: %w", err)
	}
	return &Publisher{pub: pub}, nil
}

// Publish 将任意 payload 序列化为 JSON 发布到指定 topic，透传 context 的超时/取消信号。
// message UUID 优先复用 ctx 中的 request_id，保证 HTTP 日志与 MQ 消费者日志链路可关联。
func (p *Publisher) Publish(ctx context.Context, topic string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload failed: %w", err)
	}
	msgID := watermill.NewUUID()
	if reqID, ok := ctx.Value(logger.RequestIDKey).(string); ok && reqID != "" {
		msgID = reqID
	}
	msg := message.NewMessage(msgID, data)
	msg.SetContext(ctx)
	return p.pub.Publish(topic, msg)
}

// PublishEmail 发布邮件任务
func (p *Publisher) PublishEmail(ctx context.Context, payload EmailPayload) error {
	return p.Publish(ctx, TopicEmail, payload)
}

// PublishSMS 发布短信任务
func (p *Publisher) PublishSMS(ctx context.Context, payload SMSPayload) error {
	return p.Publish(ctx, TopicSMS, payload)
}

// PublishStats 发布统计事件
func (p *Publisher) PublishStats(ctx context.Context, payload StatsPayload) error {
	return p.Publish(ctx, TopicStats, payload)
}

// Close 关闭发布者
func (p *Publisher) Close() error {
	return p.pub.Close()
}
