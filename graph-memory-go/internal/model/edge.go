package model

import (
	"time"
)

// EdgeType 边类型（关系类型）
type EdgeType string

const (
	EdgeTypeRequires  EdgeType = "REQUIRES"   // 任务需要技能
	EdgeTypeTriggered EdgeType = "TRIGGERED"  // 技能触发事件
	EdgeTypeSolves    EdgeType = "SOLVES"     // 技能解决问题
	EdgeTypeRelated   EdgeType = "RELATED"    // 相似关系
	EdgeTypeFollows   EdgeType = "FOLLOWS"    // 顺序关系
	EdgeTypeContains  EdgeType = "CONTAINS"   // 包含关系
	EdgeTypeDependsOn EdgeType = "DEPENDS_ON" // 依赖关系
)

// Edge 知识边（关系）
type Edge struct {
	// 基础信息
	ID     string  `json:"id"`
	Type   EdgeType `json:"type"`
	FromID string  `json:"from_id"`
	ToID   string  `json:"to_id"`

	// 关系属性
	Weight float64 `json:"weight"`

	// 时间戳
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 来源信息
	SourceSession string `json:"source_session,omitempty"`

	// 扩展元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewEdge 创建新边
func NewEdge(edgeType EdgeType, fromID, toID string) *Edge {
	now := time.Now().UTC() // 使用 UTC 时区，避免 Neo4j 时区错误
	return &Edge{
		ID:        GenerateID("edge"),
		Type:      edgeType,
		FromID:    fromID,
		ToID:      toID,
		Weight:    1.0,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]interface{}),
	}
}

// Validate 验证边数据
func (e *Edge) Validate() error {
	if e.FromID == "" {
		return ErrEdgeFromIDRequired
	}
	if e.ToID == "" {
		return ErrEdgeToIDRequired
	}
	if e.Type == "" {
		return ErrEdgeTypeRequired
	}
	if e.FromID == e.ToID {
		return ErrEdgeSelfLoop
	}
	return nil
}

// SetWeight 设置权重
func (e *Edge) SetWeight(weight float64) {
	e.Weight = weight
	e.UpdatedAt = time.Now().UTC()
}

// SetSourceSession 设置来源会话
func (e *Edge) SetSourceSession(sessionID string) {
	e.SourceSession = sessionID
	e.UpdatedAt = time.Now().UTC()
}

// ToMap 转换为map
func (e *Edge) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":             e.ID,
		"type":           string(e.Type),
		"from_id":        e.FromID,
		"to_id":          e.ToID,
		"weight":         e.Weight,
		"created_at":     e.CreatedAt,
		"updated_at":     e.UpdatedAt,
		"source_session": e.SourceSession,
		"metadata":       e.Metadata,
	}
}

// EdgeFromMap 从map创建Edge
func EdgeFromMap(m map[string]interface{}) (*Edge, error) {
	edge := &Edge{
		Metadata: make(map[string]interface{}),
	}

	if v, ok := m["id"].(string); ok {
		edge.ID = v
	}
	if v, ok := m["type"].(string); ok {
		edge.Type = EdgeType(v)
	}
	if v, ok := m["from_id"].(string); ok {
		edge.FromID = v
	}
	if v, ok := m["to_id"].(string); ok {
		edge.ToID = v
	}
	if v, ok := m["weight"].(float64); ok {
		edge.Weight = v
	}
	if v, ok := m["source_session"].(string); ok {
		edge.SourceSession = v
	}
	if v, ok := m["metadata"].(map[string]interface{}); ok {
		edge.Metadata = v
	}

	return edge, nil
}

// EdgeQueryResult 边查询结果
type EdgeQueryResult struct {
	Edge     *Edge `json:"edge"`
	FromNode *Node `json:"from_node,omitempty"`
	ToNode   *Node `json:"to_node,omitempty"`
}
