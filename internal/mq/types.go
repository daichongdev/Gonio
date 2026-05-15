package mq

// Topic 常量，所有消息队列主题统一在此定义
const (
	TopicEmail = "silk_route.email"
	TopicSMS   = "silk_route.sms"
	TopicStats = "silk_route.stats"
)

// ============================================================
// Payload 定义
// ============================================================

// EmailPayload 邮件消息体
type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// SMSPayload 短信消息体
type SMSPayload struct {
	Phone   string `json:"phone"`
	Content string `json:"content"`
}

// StatsPayload 数据统计消息体
type StatsPayload struct {
	Event      string         `json:"event"`
	UserID     uint           `json:"user_id,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

// ============================================================
// 消息类型注册（新增消息类型只需在此添加一个 Register 调用）
// ============================================================

func init() {
	// 注册邮件消息
	Register(MessageType[EmailPayload]{
		ShortName: "email",
		Topic:     TopicEmail,
		Handler:   handleEmail,
	})

	// 注册短信消息
	Register(MessageType[SMSPayload]{
		ShortName: "sms",
		Topic:     TopicSMS,
		Handler:   handleSMS,
	})

	// 注册统计消息
	Register(MessageType[StatsPayload]{
		ShortName: "stats",
		Topic:     TopicStats,
		Handler:   handleStats,
	})
}
