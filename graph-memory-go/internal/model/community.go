package model

import (
	"time"
)

// Community 社区
type Community struct {
	// 基础信息
	ID string `json:"id"`

	// 摘要信息
	Summary string `json:"summary,omitempty"`

	// 统计信息
	MemberCount int `json:"member_count"`

	// 向量相关
	EmbeddingHash string    `json:"embedding_hash,omitempty"`
	Embedding     []float32 `json:"embedding,omitempty"`

	// 时间戳
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewCommunity 创建新社区
func NewCommunity() *Community {
	now := time.Now().UTC() // 使用 UTC 时区，避免 Neo4j 时区错误
	return &Community{
		ID:           GenerateID("community"),
		MemberCount:  0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// SetSummary 设置摘要
func (c *Community) SetSummary(summary string) {
	c.Summary = summary
	c.UpdatedAt = time.Now().UTC()
}

// SetMemberCount 设置成员数量
func (c *Community) SetMemberCount(count int) {
	c.MemberCount = count
	c.UpdatedAt = time.Now().UTC()
}

// SetEmbedding 设置向量
func (c *Community) SetEmbedding(embedding []float32) {
	c.Embedding = embedding
	c.EmbeddingHash = ComputeEmbeddingHash(embedding)
	c.UpdatedAt = time.Time{}
}

// ToMap 转换为map
func (c *Community) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":             c.ID,
		"summary":        c.Summary,
		"member_count":   c.MemberCount,
		"embedding_hash": c.EmbeddingHash,
		"embedding":      c.Embedding,
		"created_at":     c.CreatedAt,
		"updated_at":     c.UpdatedAt,
	}
}

// CommunityFromMap 从map创建Community
func CommunityFromMap(m map[string]interface{}) (*Community, error) {
	community := &Community{}

	if v, ok := m["id"].(string); ok {
		community.ID = v
	}
	if v, ok := m["summary"].(string); ok {
		community.Summary = v
	}
	if v, ok := m["member_count"].(int64); ok {
		community.MemberCount = int(v)
	}
	if v, ok := m["embedding_hash"].(string); ok {
		community.EmbeddingHash = v
	}
	if v, ok := m["embedding"].([]interface{}); ok {
		community.Embedding = make([]float32, 0, len(v))
		for _, f := range v {
			if val, ok := f.(float64); ok {
				community.Embedding = append(community.Embedding, float32(val))
			}
		}
	}

	return community, nil
}

// CommunityWithMembers 社区及其成员
type CommunityWithMembers struct {
	Community *Community `json:"community"`
	Members   []*Node    `json:"members"`
}
