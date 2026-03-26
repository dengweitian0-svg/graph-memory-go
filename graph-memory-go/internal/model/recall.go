package model

import "time"

// ==================== 召回请求/响应类型 ====================

// RecallRequest 召回请求
type RecallRequest struct {
	// 查询文本
	Query string `json:"query"`
	// 返回数量限制
	Limit int `json:"limit,omitempty"`
	// 是否启用精确路径召回
	EnablePrecisePath bool `json:"enable_precise_path,omitempty"`
	// 是否启用泛化路径召回
	EnableGeneralizedPath bool `json:"enable_generalized_path,omitempty"`
	// 精确路径权重 (0-1)
	PreciseWeight float64 `json:"precise_weight,omitempty"`
	// 最大召回深度
	MaxDepth int `json:"max_depth,omitempty"`
	// 会话ID (可选，用于上下文关联)
	SessionID string `json:"session_id,omitempty"`
	// 过滤条件
	Filters *RecallFilters `json:"filters,omitempty"`
}

// RecallFilters 召回过滤条件
type RecallFilters struct {
	// 节点类型过滤
	NodeTypes []NodeType `json:"node_types,omitempty"`
	// 状态过滤
	Status NodeStatus `json:"status,omitempty"`
	// 社区ID过滤
	CommunityID string `json:"community_id,omitempty"`
	// 最小PageRank分数
	MinPageRank float64 `json:"min_pagerank,omitempty"`
}

// RecallResponse 召回响应
type RecallResponse struct {
	// 召回结果列表
	Results []*RecallResult `json:"results"`
	// 总数量
	Total int `json:"total"`
	// 查询耗时 (毫秒)
	Duration int64 `json:"duration_ms"`
	// Token估算
	TokenEstimate *TokenEstimate `json:"token_estimate,omitempty"`
	// 召回统计
	Stats *RecallStats `json:"stats,omitempty"`
}

// RecallResult 单个召回结果
type RecallResult struct {
	// 节点信息
	Node *Node `json:"node"`
	// 相关性分数 (0-1)
	Score float64 `json:"score"`
	// 召回路径
	Path string `json:"path"` // "precise" 或 "generalized"
	// 匹配原因
	MatchReason string `json:"match_reason"`
	// 上下文路径 (从查询节点到该节点的路径)
	ContextPath []*Node `json:"context_path,omitempty"`
	// 邻居节点 (相关联的节点)
	Neighbors []*Node `json:"neighbors,omitempty"`
}

// RecallStats 召回统计
type RecallStats struct {
	// 精确路径召回数量
	PreciseCount int `json:"precise_count"`
	// 泛化路径召回数量
	GeneralizedCount int `json:"generalized_count"`
	// 去重后数量
	DeduplicatedCount int `json:"deduplicated_count"`
	// 向量搜索耗时 (毫秒)
	VectorSearchDuration int64 `json:"vector_search_duration_ms"`
	// 图遍历耗时 (毫秒)
	GraphTraversalDuration int64 `json:"graph_traversal_duration_ms"`
}

// ==================== Token估算相关 ====================

// TokenEstimate Token估算结果
type TokenEstimate struct {
	// 总Token数
	TotalTokens int `json:"total_tokens"`
	// 节点描述Token数
	NodeTokens int `json:"node_tokens"`
	// 上下文Token数
	ContextTokens int `json:"context_tokens"`
	// 关系Token数
	RelationTokens int `json:"relation_tokens"`
	// 是否超出限制
	ExceedsLimit bool `json:"exceeds_limit"`
	// Token限制
	TokenLimit int `json:"token_limit"`
}

// TokenEstimator Token估算器配置
type TokenEstimatorConfig struct {
	// Token限制 (默认4000)
	TokenLimit int
	// 平均每字符Token数 (中文约0.5，英文约0.25)
	CharsPerToken float64
	// 节点描述平均长度
	AvgNodeDescLength int
	// 关系描述平均长度
	AvgRelationLength int
}

// DefaultTokenEstimatorConfig 默认Token估算器配置
func DefaultTokenEstimatorConfig() *TokenEstimatorConfig {
	return &TokenEstimatorConfig{
		TokenLimit:        4000,
		CharsPerToken:     0.5,
		AvgNodeDescLength: 100,
		AvgRelationLength: 50,
	}
}

// ==================== 上下文构建相关 ====================

// ContextBuilder 上下文构建结果
type ContextBuilder struct {
	// 构建的上下文文本
	Context string `json:"context"`
	// 包含的节点
	Nodes []*Node `json:"nodes"`
	// 包含的边
	Edges []*Edge `json:"edges"`
	// Token估算
	TokenEstimate *TokenEstimate `json:"token_estimate"`
	// 构建时间
	BuildTime time.Time `json:"build_time"`
}

// ContextBuildRequest 上下文构建请求
type ContextBuildRequest struct {
	// 召回结果
	Results []*RecallResult `json:"results"`
	// 最大Token数
	MaxTokens int `json:"max_tokens"`
	// 是否包含邻居节点
	IncludeNeighbors bool `json:"include_neighbors"`
	// 是否包含上下文路径
	IncludeContextPath bool `json:"include_context_path"`
	// 格式类型 (json, text, markdown)
	Format string `json:"format"`
}

// ==================== 向量搜索相关 ====================

// VectorSearchResult 向量搜索结果
type VectorSearchResult struct {
	NodeID   string  `json:"node_id"`
	Score    float64 `json:"score"`
	Distance float64 `json:"distance"`
}

// VectorSearchRequest 向量搜索请求
type VectorSearchRequest struct {
	// 查询向量
	Vector []float64 `json:"vector"`
	// 返回数量
	Limit int `json:"limit"`
	// 最小相似度
	MinScore float64 `json:"min_score,omitempty"`
	// 过滤条件
	Filters *RecallFilters `json:"filters,omitempty"`
}

// ==================== 双路径召回相关 ====================

// RecallPathType 召回路径类型
type RecallPathType string

const (
	// RecallPathPrecise 精确路径 - 直接匹配查询中的实体
	RecallPathPrecise RecallPathType = "precise"
	// RecallPathGeneralized 泛化路径 - 通过图遍历扩展相关节点
	RecallPathGeneralized RecallPathType = "generalized"
)

// RecallPath 召回路径
type RecallPath struct {
	// 路径类型
	Type RecallPathType `json:"type"`
	// 起始节点ID
	StartNodeID string `json:"start_node_id"`
	// 路径上的节点
	Nodes []*Node `json:"nodes"`
	// 路径上的边
	Edges []*Edge `json:"edges"`
	// 路径分数
	Score float64 `json:"score"`
	// 路径深度
	Depth int `json:"depth"`
}

// PrecisePathResult 精确路径召回结果
type PrecisePathResult struct {
	// 匹配的节点
	MatchedNodes []*Node `json:"matched_nodes"`
	// 直接邻居
	DirectNeighbors []*Node `json:"direct_neighbors"`
	// 匹配分数
	Scores map[string]float64 `json:"scores"`
}

// GeneralizedPathResult 泛化路径召回结果
type GeneralizedPathResult struct {
	// 通过图遍历发现的节点
	DiscoveredNodes []*Node `json:"discovered_nodes"`
	// 扩展路径
	ExpandedPaths []*RecallPath `json:"expanded_paths"`
	// 社区相关节点
	CommunityNodes []*Node `json:"community_nodes"`
}

// ==================== 召回配置 ====================

// RecallConfig 召回配置
type RecallConfig struct {
	// 默认召回数量
	DefaultLimit int `json:"default_limit"`
	// 精确路径召回权重
	PreciseWeight float64 `json:"precise_weight"`
	// 泛化路径召回权重
	GeneralizedWeight float64 `json:"generalized_weight"`
	// 向量搜索最小相似度
	MinVectorSimilarity float64 `json:"min_vector_similarity"`
	// 图遍历最大深度
	MaxTraversalDepth int `json:"max_traversal_depth"`
	// 是否启用社区扩展
	EnableCommunityExpansion bool `json:"enable_community_expansion"`
	// Token估算器配置
	TokenEstimator *TokenEstimatorConfig `json:"token_estimator,omitempty"`
}

// DefaultRecallConfig 默认召回配置
func DefaultRecallConfig() *RecallConfig {
	return &RecallConfig{
		DefaultLimit:             10,
		PreciseWeight:            0.7,
		GeneralizedWeight:        0.3,
		MinVectorSimilarity:      0.5,
		MaxTraversalDepth:        3,
		EnableCommunityExpansion: true,
		TokenEstimator:           DefaultTokenEstimatorConfig(),
	}
}

// Validate 验证召回配置
func (c *RecallConfig) Validate() error {
	if c.DefaultLimit <= 0 {
		c.DefaultLimit = 10
	}
	if c.PreciseWeight < 0 || c.PreciseWeight > 1 {
		c.PreciseWeight = 0.7
	}
	if c.GeneralizedWeight < 0 || c.GeneralizedWeight > 1 {
		c.GeneralizedWeight = 0.3
	}
	if c.MinVectorSimilarity < 0 || c.MinVectorSimilarity > 1 {
		c.MinVectorSimilarity = 0.5
	}
	if c.MaxTraversalDepth <= 0 {
		c.MaxTraversalDepth = 3
	}
	return nil
}
