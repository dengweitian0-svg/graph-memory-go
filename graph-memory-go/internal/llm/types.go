package llm

import "time"

// ==================== 三元组提取相关类型 ====================

// Triple 知识三元组
type Triple struct {
	Subject     string  `json:"subject"`
	Predicate   string  `json:"predicate"`
	Object      string  `json:"object"`
	SubjectType string  `json:"subject_type,omitempty"`
	ObjectType  string  `json:"object_type,omitempty"`
	Confidence  float64 `json:"confidence"`
}

// ExtractedEntity 提取的实体
type ExtractedEntity struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence,omitempty"`
}

// Entity 实体识别结果中的实体
type Entity struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence,omitempty"`
}

// ExtractionResult LLM三元组提取结果
type ExtractionResult struct {
	Triples  []Triple          `json:"triples"`
	Entities []ExtractedEntity `json:"entities"`
	Summary  string            `json:"summary"`
}

// EntityRecognitionResult 实体识别结果
type EntityRecognitionResult struct {
	Entities []Entity `json:"entities"`
}

// SummarizeCommunityResult 社区摘要结果
type SummarizeCommunityResult struct {
	CommunityID     string   `json:"community_id"`
	Summary         string   `json:"summary"`
	KeyTopics       []string `json:"key_topics"`
	ImportanceScore float64  `json:"importance_score"`
}

// ==================== 知识提取结果类型 ====================

// KnowledgeExtraction 知识提取结果（服务层使用）
type KnowledgeExtraction struct {
	Triples  []*Triple         `json:"triples"`
	Entities []*ExtractedEntity `json:"entities"`
	Summary  string            `json:"summary"`
}

// CommunitySummary 社区摘要（服务层使用）
type CommunitySummary struct {
	CommunityID     string   `json:"community_id"`
	Summary         string   `json:"summary"`
	KeyTopics       []string `json:"key_topics"`
	ImportanceScore float64  `json:"importance_score"`
}

// ==================== 缓存相关类型 ====================

// cacheEntry 缓存条目（内部使用）
type cacheEntry struct {
	embedding []float64
	timestamp time.Time
}

// ==================== 配置相关类型 ====================

// ServiceConfig LLM服务配置
type ServiceConfig struct {
	EinoConfig *EinoConfig
	CacheTTL   int64 // 缓存TTL（秒），0表示使用默认值
}

// CacheEntry 缓存条目（导出版本）
type CacheEntry struct {
	Embedding []float64
	Timestamp time.Time
}
