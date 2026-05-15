package mq

import (
	"gonio/internal/pkg/logger"

	"github.com/ThreeDotsLabs/watermill/message"
)

// ============================================================
// Handler 实现
// ============================================================

func handleEmail(msg *message.Message, payload EmailPayload) error {
	// TODO: 替换为真实的邮件发送逻辑（如 SMTP / SendGrid）
	logger.Log.Infow("[mq] send email",
		"msg_uuid", msg.UUID,
		"to", payload.To,
		"subject", payload.Subject,
	)
	return nil
}

func handleSMS(msg *message.Message, payload SMSPayload) error {
	// TODO: 替换为真实的短信发送逻辑（如阿里云 / 腾讯云短信）
	logger.Log.Infow("[mq] send sms",
		"msg_uuid", msg.UUID,
		"phone", payload.Phone,
		"content", payload.Content,
	)
	return nil
}

func handleStats(msg *message.Message, payload StatsPayload) error {
	// TODO: 替换为真实的统计逻辑（如写入 ClickHouse / 更新聚合表）
	logger.Log.Infow("[mq] record stats",
		"msg_uuid", msg.UUID,
		"event", payload.Event,
		"user_id", payload.UserID,
		"properties", payload.Properties,
	)
	return nil
}
