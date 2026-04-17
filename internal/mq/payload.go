package mq

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
