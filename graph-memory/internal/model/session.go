package model

import (
	"time"
)

// SessionStatus 会话状态
type SessionStatus string

const (
	SessionStatusActive    SessionStatus = "active"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusArchived  SessionStatus = "archived"
)

// Session 会话
type Session struct {
	// 基础信息
	ID     string        `json:"id"`
	Status SessionStatus `json:"status"`

	// 时间戳
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`

	// 统计信息
	MessageCount int `json:"message_count"`

	// 扩展元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewSession 创建新会话
func NewSession() *Session {
	return &Session{
		ID:           GenerateID("session"),
		Status:       SessionStatusActive,
		StartedAt:    time.Now().UTC(), // 使用 UTC 时区，避免 Neo4j 时区错误
		MessageCount: 0,
		Metadata:     make(map[string]interface{}),
	}
}

// Validate 验证会话数据
func (s *Session) Validate() error {
	if s.ID == "" {
		return ErrSessionIDRequired
	}
	return nil
}

// IncrementMessageCount 增加消息计数
func (s *Session) IncrementMessageCount() {
	s.MessageCount++
}

// Complete 完成会话
func (s *Session) Complete() {
	s.Status = SessionStatusCompleted
	now := time.Now().UTC()
	s.EndedAt = &now
}

// Archive 归档会话
func (s *Session) Archive() {
	s.Status = SessionStatusArchived
}

// ToMap 转换为map
func (s *Session) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"id":            s.ID,
		"status":        string(s.Status),
		"started_at":    s.StartedAt,
		"message_count": s.MessageCount,
		"metadata":      s.Metadata,
	}

	if s.EndedAt != nil {
		m["ended_at"] = *s.EndedAt
	}

	return m
}

// SessionFromMap 从map创建Session
func SessionFromMap(m map[string]interface{}) (*Session, error) {
	session := &Session{
		Metadata: make(map[string]interface{}),
	}

	if v, ok := m["id"].(string); ok {
		session.ID = v
	}
	if v, ok := m["status"].(string); ok {
		session.Status = SessionStatus(v)
	}
	if v, ok := m["message_count"].(int64); ok {
		session.MessageCount = int(v)
	}
	if v, ok := m["metadata"].(map[string]interface{}); ok {
		session.Metadata = v
	}

	return session, nil
}

// SessionWithMessages 会话及其消息
type SessionWithMessages struct {
	Session  *Session   `json:"session"`
	Messages []*Message `json:"messages"`
}
