package model

import (
	"time"
)

// NodeType 节点类型
type NodeType string

const (
	NodeTypeTask   NodeType = "TASK"
	NodeTypeSkill  NodeType = "SKILL"
	NodeTypeEvent  NodeType = "EVENT"
	NodeTypeConcept NodeType = "CONCEPT"
)

// NodeStatus 节点状态
type NodeStatus string

const (
	NodeStatusActive     NodeStatus = "active"
	NodeStatusDeprecated NodeStatus = "deprecated"
)

// Node 知识节点
type Node struct {
	// 基础信息
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        NodeType   `json:"type"`
	Description string     `json:"description"`
	Status      NodeStatus `json:"status"`

	// 统计信息
	ValidatedCount int `json:"validated_count"`

	// 图算法相关
	PageRank    float64 `json:"pagerank,omitempty"`
	CommunityID string  `json:"community_id,omitempty"`

	// 时间戳
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 来源信息
	SourceSessions []string `json:"source_sessions"`

	// 向量相关
	EmbeddingHash string    `json:"embedding_hash,omitempty"`
	Embedding     []float32 `json:"embedding,omitempty"`

	// 扩展元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewNode 创建新节点
func NewNode(name string, nodeType NodeType, description string) *Node {
	now := time.Now().UTC() // 使用 UTC 时区，避免 Neo4j 时区错误
	return &Node{
		ID:             GenerateID("node"),
		Name:           name,
		Type:           nodeType,
		Description:    description,
		Status:         NodeStatusActive,
		ValidatedCount: 0,
		PageRank:       0,
		CommunityID:    "",
		CreatedAt:      now,
		UpdatedAt:      now,
		SourceSessions: []string{},
		Metadata:       make(map[string]interface{}),
	}
}

// Validate 验证节点数据
func (n *Node) Validate() error {
	if n.Name == "" {
		return ErrNodeNameRequired
	}
	if n.Type == "" {
		return ErrNodeTypeRequired
	}
	if n.Description == "" {
		return ErrNodeDescriptionRequired
	}
	return nil
}

// AddSourceSession 添加来源会话
func (n *Node) AddSourceSession(sessionID string) {
	for _, s := range n.SourceSessions {
		if s == sessionID {
			return
		}
	}
	n.SourceSessions = append(n.SourceSessions, sessionID)
	n.UpdatedAt = time.Now().UTC()
}

// IncrementValidatedCount 增加验证次数
func (n *Node) IncrementValidatedCount() {
	n.ValidatedCount++
	n.UpdatedAt = time.Now().UTC()
}

// SetPageRank 设置PageRank值
func (n *Node) SetPageRank(pr float64) {
	n.PageRank = pr
	n.UpdatedAt = time.Now().UTC()
}

// SetCommunityID 设置社区ID
func (n *Node) SetCommunityID(communityID string) {
	n.CommunityID = communityID
	n.UpdatedAt = time.Time{}
}

// SetEmbedding 设置向量
func (n *Node) SetEmbedding(embedding []float32) {
	n.Embedding = embedding
	n.EmbeddingHash = ComputeEmbeddingHash(embedding)
	n.UpdatedAt = time.Now().UTC()
}

// ToMap 转换为map
func (n *Node) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":              n.ID,
		"name":            n.Name,
		"type":            string(n.Type),
		"description":     n.Description,
		"status":          string(n.Status),
		"validated_count": n.ValidatedCount,
		"pagerank":        n.PageRank,
		"community_id":    n.CommunityID,
		"created_at":      n.CreatedAt,
		"updated_at":      n.UpdatedAt,
		"source_sessions": n.SourceSessions,
		"embedding_hash":  n.EmbeddingHash,
		"embedding":       n.Embedding,
		"metadata":        n.Metadata,
	}
}

// NodeFromMap 从map创建Node
func NodeFromMap(m map[string]interface{}) (*Node, error) {
	node := &Node{
		Metadata: make(map[string]interface{}),
	}

	if v, ok := m["id"].(string); ok {
		node.ID = v
	}
	if v, ok := m["name"].(string); ok {
		node.Name = v
	}
	if v, ok := m["type"].(string); ok {
		node.Type = NodeType(v)
	}
	if v, ok := m["description"].(string); ok {
		node.Description = v
	}
	if v, ok := m["status"].(string); ok {
		node.Status = NodeStatus(v)
	}
	if v, ok := m["validated_count"].(int64); ok {
		node.ValidatedCount = int(v)
	}
	if v, ok := m["pagerank"].(float64); ok {
		node.PageRank = v
	}
	if v, ok := m["community_id"].(string); ok {
		node.CommunityID = v
	}
	if v, ok := m["source_sessions"].([]interface{}); ok {
		node.SourceSessions = make([]string, 0, len(v))
		for _, s := range v {
			if str, ok := s.(string); ok {
				node.SourceSessions = append(node.SourceSessions, str)
			}
		}
	}
	if v, ok := m["embedding_hash"].(string); ok {
		node.EmbeddingHash = v
	}
	if v, ok := m["embedding"].([]interface{}); ok {
		node.Embedding = make([]float32, 0, len(v))
		for _, f := range v {
			if val, ok := f.(float64); ok {
				node.Embedding = append(node.Embedding, float32(val))
			}
		}
	}
	if v, ok := m["metadata"].(map[string]interface{}); ok {
		node.Metadata = v
	}

	return node, nil
}
