package llm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/pkg/logger"
)

// Service LLM服务 - 基于eino框架
// 提供知识提取、嵌入生成、社区摘要等功能
type Service struct {
	client *EinoClient
	log    *logger.Logger

	// 缓存
	embeddingCache sync.Map
	cacheTTL       time.Duration
}

// NewService 创建LLM服务
func NewService(config *ServiceConfig) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if config.EinoConfig == nil {
		config.EinoConfig = DefaultEinoConfig()
	}

	client, err := NewEinoClient(config.EinoConfig)
	if err != nil {
		return nil, fmt.Errorf("create eino client: %w", err)
	}

	cacheTTL := 24 * time.Hour
	if config.CacheTTL > 0 {
		cacheTTL = time.Duration(config.CacheTTL) * time.Second
	}

	return &Service{
		client:   client,
		log:      logger.NewLogger("info"),
		cacheTTL: cacheTTL,
	}, nil
}

// ==================== 核心API ====================

// ExtractKnowledgeFromText 从文本中提取知识
func (s *Service) ExtractKnowledgeFromText(ctx context.Context, text string) (*KnowledgeExtraction, error) {
	s.log.Debug("Extracting knowledge from text", "length", len(text))

	// 提取三元组
	extraction, err := s.client.ExtractTriples(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("extract triples: %w", err)
	}

	// 识别实体
	entities, err := s.client.RecognizeEntities(ctx, text, nil)
	if err != nil {
		s.log.Warn("Entity recognition failed", "error", err)
	}

	// 合并实体
	allEntities := mergeEntities(extraction.Entities, entities.Entities)

	// 转换 Triples 为指针切片
	triples := make([]*Triple, len(extraction.Triples))
	for i := range extraction.Triples {
		triples[i] = &extraction.Triples[i]
	}

	return &KnowledgeExtraction{
		Triples:  triples,
		Entities: allEntities,
		Summary:  extraction.Summary,
	}, nil
}

// GenerateNodeEmbedding 为节点生成嵌入向量
func (s *Service) GenerateNodeEmbedding(ctx context.Context, node *model.Node) ([]float64, error) {
	// 检查缓存
	cacheKey := fmt.Sprintf("node_%s", node.ID)
	if cached, ok := s.embeddingCache.Load(cacheKey); ok {
		if entry, ok := cached.(*cacheEntry); ok && time.Since(entry.timestamp) < s.cacheTTL {
			return entry.embedding, nil
		}
	}

	// 构建文本
	text := fmt.Sprintf("%s: %s", node.Name, node.Description)
	if node.Type != "" {
		text = fmt.Sprintf("[%s] %s", node.Type, text)
	}

	// 生成嵌入
	embedding, err := s.client.GenerateEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}

	// 存入缓存
	s.embeddingCache.Store(cacheKey, &cacheEntry{
		embedding: embedding,
		timestamp: time.Now().UTC(),
	})

	return embedding, nil
}

// GenerateBatchEmbeddings 批量生成嵌入
func (s *Service) GenerateBatchEmbeddings(ctx context.Context, nodes []*model.Node) (map[string][]float64, error) {
	if len(nodes) == 0 {
		return make(map[string][]float64), nil
	}

	// 构建文本列表
	texts := make([]string, len(nodes))
	nodeIDs := make([]string, len(nodes))

	for i, node := range nodes {
		texts[i] = fmt.Sprintf("[%s] %s: %s", node.Type, node.Name, node.Description)
		nodeIDs[i] = node.ID
	}

	// 生成嵌入
	embeddings, err := s.client.GenerateBatchEmbeddings(ctx, texts)
	if err != nil {
		return nil, err
	}

	// 构建结果映射
	result := make(map[string][]float64)
	for i, embedding := range embeddings {
		if i < len(nodeIDs) {
			result[nodeIDs[i]] = embedding
		}
	}

	return result, nil
}

// SummarizeCommunity 为社区生成摘要
func (s *Service) SummarizeCommunity(ctx context.Context, communityID string, nodes []*model.Node) (*CommunitySummary, error) {
	if len(nodes) == 0 {
		return &CommunitySummary{
			CommunityID: communityID,
			Summary:     "空社区",
		}, nil
	}

	// 构建社区信息
	descriptions := make([]string, len(nodes))
	for i, node := range nodes {
		descriptions[i] = fmt.Sprintf("- %s (%s): %s", node.Name, node.Type, node.Description)
	}

	// 生成摘要
	result, err := s.client.SummarizeCommunity(ctx, communityID, descriptions)
	if err != nil {
		return nil, err
	}

	return &CommunitySummary{
		CommunityID:     result.CommunityID,
		Summary:         result.Summary,
		KeyTopics:       result.KeyTopics,
		ImportanceScore: result.ImportanceScore,
	}, nil
}

// GenerateEmbedding 生成文本嵌入（简化接口）
func (s *Service) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// 检查缓存
	if cached, ok := s.embeddingCache.Load(text); ok {
		return cached.([]float64), nil
	}

	embedding, err := s.client.GenerateEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}

	// 存入缓存
	s.embeddingCache.Store(text, embedding)

	return embedding, nil
}

// HealthCheck 健康检查
func (s *Service) HealthCheck(ctx context.Context) error {
	return s.client.HealthCheck(ctx)
}

// GetClient 获取底层客户端（用于高级操作）
func (s *Service) GetClient() *EinoClient {
	return s.client
}

// ==================== 工厂函数 ====================

// NewDefaultService 创建默认LLM服务
func NewDefaultService(apiKey string) (*Service, error) {
	return NewService(&ServiceConfig{
		EinoConfig: &EinoConfig{
			APIKey: apiKey,
		},
	})
}

// DefaultServiceConfig 默认服务配置
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		EinoConfig: DefaultEinoConfig(),
		CacheTTL:   int64(24 * time.Hour / time.Second), // 24小时（秒）
	}
}
