package mq

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
