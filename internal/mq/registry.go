package mq

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
)

// MessageType 消息类型定义
type MessageType[T any] struct {
	ShortName string                          // 配置文件中使用的短名称（如 "email"）
	Topic     string                          // 完整的 topic 名称（如 "silk_route.email"）
	Handler   func(*message.Message, T) error // 业务处理函数
}

// registry 全局消息类型注册表
var registry = &messageRegistry{
	types: make(map[string]*registryEntry),
}

type registryEntry struct {
	shortName string
	topic     string
	handler   message.NoPublishHandlerFunc
}

type messageRegistry struct {
	types map[string]*registryEntry // key: topic
}

// Register 注册消息类型（泛型方法，自动处理序列化）
func Register[T any](mt MessageType[T]) {
	handler := func(msg *message.Message) error {
		var payload T
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal %s payload: %w", mt.ShortName, err)
		}
		return mt.Handler(msg, payload)
	}

	registry.types[mt.Topic] = &registryEntry{
		shortName: mt.ShortName,
		topic:     mt.Topic,
		handler:   handler,
	}
}

// GetAllTopics 获取所有已注册的 topic
func GetAllTopics() []string {
	topics := make([]string, 0, len(registry.types))
	for topic := range registry.types {
		topics = append(topics, topic)
	}
	return topics
}

// GetShortToTopicMap 获取 shortName -> topic 映射
func GetShortToTopicMap() map[string]string {
	m := make(map[string]string, len(registry.types))
	for _, entry := range registry.types {
		m[entry.shortName] = entry.topic
	}
	return m
}

// GetTopicToShortMap 获取 topic -> shortName 映射
func GetTopicToShortMap() map[string]string {
	m := make(map[string]string, len(registry.types))
	for _, entry := range registry.types {
		m[entry.topic] = entry.shortName
	}
	return m
}

// GetHandlers 获取所有 handler 定义（用于 router 注册）
func GetHandlers() []struct {
	Name    string
	Topic   string
	Handler message.NoPublishHandlerFunc
} {
	handlers := make([]struct {
		Name    string
		Topic   string
		Handler message.NoPublishHandlerFunc
	}, 0, len(registry.types))

	for _, entry := range registry.types {
		handlers = append(handlers, struct {
			Name    string
			Topic   string
			Handler message.NoPublishHandlerFunc
		}{
			Name:    entry.shortName,
			Topic:   entry.topic,
			Handler: entry.handler,
		})
	}
	return handlers
}
