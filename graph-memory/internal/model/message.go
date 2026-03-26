package model

import (
	"time"
)

// MessageRole 消息角色
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
)

// Message 对话消息
type Message struct {
	// 基础信息
	ID        string      `json:"id"`
	SessionID string      `json:"session_id"`
	TurnIndex int         `json:"turn_index"`
	Role      MessageRole `json:"role"`
	Content   string      `json:"content"`

	// 时间戳
	Timestamp time.Time `json:"timestamp"`

	// 提取状态
	Extracted bool `json:"extracted"`

	// 扩展元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewMessage 创建新消息
func NewMessage(sessionID string, turnIndex int, role MessageRole, content string) *Message {
	return &Message{
		ID:        GenerateID("msg"),
		SessionID: sessionID,
		TurnIndex: turnIndex,
		Role:      role,
		Content:   content,
		Timestamp: time.Now().UTC(), // 使用 UTC 时区，避免 Neo4j 时区错误
		Extracted: false,
		Metadata:  make(map[string]interface{}),
	}
}

// Validate 验证消息数据
func (m *Message) Validate() error {
	if m.SessionID == "" {
		return ErrMessageSessionIDRequired
	}
	if m.Role == "" {
		return ErrMessageRoleRequired
	}
	if m.Content == "" {
		return ErrMessageContentRequired
	}
	return nil
}

// MarkExtracted 标记为已提取
func (m *Message) MarkExtracted() {
	m.Extracted = true
}

// ToMap 转换为map
func (m *Message) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":         m.ID,
		"session_id": m.SessionID,
		"turn_index": m.TurnIndex,
		"role":       string(m.Role),
		"content":    m.Content,
		"timestamp":  m.Timestamp,
		"extracted":  m.Extracted,
		"metadata":   m.Metadata,
	}
}

// MessageFromMap 从map创建Message
func MessageFromMap(m map[string]interface{}) (*Message, error) {
	message := &Message{
		Metadata: make(map[string]interface{}),
	}

	if v, ok := m["id"].(string); ok {
		message.ID = v
	}
	if v, ok := m["session_id"].(string); ok {
		message.SessionID = v
	}
	if v, ok := m["turn_index"].(int64); ok {
		message.TurnIndex = int(v)
	}
	if v, ok := m["role"].(string); ok {
		message.Role = MessageRole(v)
	}
	if v, ok := m["content"].(string); ok {
		message.Content = v
	}
	if v, ok := m["extracted"].(bool); ok {
		message.Extracted = v
	}
	if v, ok := m["metadata"].(map[string]interface{}); ok {
		message.Metadata = v
	}

	return message, nil
}
