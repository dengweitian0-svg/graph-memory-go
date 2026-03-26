// Package llm provides LLM integration using eino framework
// Supports deepseek-v3-2-251201 for chat and doubao-embedding-vision-251215 for embeddings
package llm

import (
	"context"
	"fmt"
	"io"
	"sync"

	arkembedding "github.com/cloudwego/eino-ext/components/embedding/ark"
	arkmodel "github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/schema"
	"github.com/example/graph-memory/pkg/logger"
)

// EinoClient LLM客户端，基于eino框架
type EinoClient struct {
	chatModel    *arkmodel.ChatModel
	embedding    *arkembedding.Embedder
	log          *logger.Logger
	model        string
	embeddingDim int
}

// EinoConfig eino客户端配置
type EinoConfig struct {
	APIKey         string
	BaseURL        string
	Model          string // ChatModel ID
	EmbeddingModel string // Embedding Model ID
	Temperature    float64
	MaxTokens      int
}

// DefaultEinoConfig 默认配置
func DefaultEinoConfig() *EinoConfig {
	return &EinoConfig{
		BaseURL:        "https://ark.cn-beijing.volces.com/api/v3",
		Model:          "doubao-seed-2-0-pro-260215",
		EmbeddingModel: "doubao-embedding-vision-251215",
		Temperature:    0.7,
		MaxTokens:      4096,
	}
}

// NewEinoClient 创建eino客户端
func NewEinoClient(config *EinoConfig) (*EinoClient, error) {
	if config == nil {
		config = DefaultEinoConfig()
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// 创建ChatModel
	chatModel, err := arkmodel.NewChatModel(context.Background(), &arkmodel.ChatModelConfig{
		Model:       config.Model,
		BaseURL:     config.BaseURL,
		APIKey:      config.APIKey,
		Temperature: ptrFloat32(float32(config.Temperature)),
		MaxTokens:   ptrInt(config.MaxTokens),
	})
	if err != nil {
		return nil, fmt.Errorf("create chat model: %w", err)
	}

	// 创建Embedding
	embedding, err := arkembedding.NewEmbedder(context.Background(), &arkembedding.EmbeddingConfig{
		Model:   config.EmbeddingModel,
		BaseURL: config.BaseURL,
		APIKey:  config.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("create embedding: %w", err)
	}

	return &EinoClient{
		chatModel:    chatModel,
		embedding:    embedding,
		log:          logger.NewLogger("info"),
		model:        config.Model,
		embeddingDim: 2048, // doubao-embedding-vision-251215 默认维度
	}, nil
}

// ==================== ChatModel API ====================

// Chat 同步聊天
func (c *EinoClient) Chat(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	resp, err := c.chatModel.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("chat generate: %w", err)
	}
	return resp, nil
}

// ChatStream 流式聊天
func (c *EinoClient) ChatStream(ctx context.Context, messages []*schema.Message) (<-chan *schema.Message, <-chan error) {
	stream, err := c.chatModel.Stream(ctx, messages)
	if err != nil {
		errChan := make(chan error, 1)
		errChan <- err
		close(errChan)
		msgChan := make(chan *schema.Message)
		close(msgChan)
		return msgChan, errChan
	}

	msgChan := make(chan *schema.Message, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(msgChan)
		defer close(errChan)
		defer stream.Close()

		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				errChan <- err
				return
			}
			msgChan <- msg
		}
	}()

	return msgChan, errChan
}

// ChatWithSystem 带系统提示的聊天
func (c *EinoClient) ChatWithSystem(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userMessage),
	}

	resp, err := c.Chat(ctx, messages)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// ==================== Embedding API ====================

// GenerateEmbedding 生成单个文本的嵌入向量
func (c *EinoClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := c.embedding.EmbedStrings(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("generate embedding: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}

	// 转换为float64
	result := make([]float64, len(embeddings[0]))
	for i, v := range embeddings[0] {
		result[i] = float64(v)
	}

	return result, nil
}

// GenerateBatchEmbeddings 批量生成嵌入向量
func (c *EinoClient) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	embeddings, err := c.embedding.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("generate batch embeddings: %w", err)
	}

	// 转换为float64
	results := make([][]float64, len(embeddings))
	for i, emb := range embeddings {
		results[i] = make([]float64, len(emb))
		for j, v := range emb {
			results[i][j] = float64(v)
		}
	}

	return results, nil
}

// ==================== 知识图谱专用API ====================

// ExtractTriples 从文本中提取知识三元组
func (c *EinoClient) ExtractTriples(ctx context.Context, text string) (*ExtractionResult, error) {
	systemPrompt := `你是一个专业的知识图谱构建专家。请从文本中提取知识三元组。

输出JSON格式：
{
  "triples": [
    {
      "subject": "主体名称",
      "predicate": "关系",
      "object": "客体名称",
      "subject_type": "主体类型",
      "object_type": "客体类型",
      "confidence": 0.95
    }
  ],
  "entities": [
    {
      "name": "实体名称",
      "type": "实体类型",
      "description": "实体描述"
    }
  ],
  "summary": "文本摘要"
}

实体类型包括：TASK(任务)、SKILL(技能)、CONCEPT(概念)、EVENT(事件)、PERSON(人物)、ORGANIZATION(组织)

请确保输出是有效的JSON格式。`

	userPrompt := fmt.Sprintf("请从以下文本中提取知识三元组：\n\n%s", text)

	resp, err := c.ChatWithSystem(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("extract triples: %w", err)
	}

	// 解析JSON响应
	result, err := parseExtractionResult(resp)
	if err != nil {
		c.log.Warn("Failed to parse extraction result, using fallback", "error", err)
		// 降级处理
		return &ExtractionResult{
			Triples:  []Triple{},
			Entities: []ExtractedEntity{},
			Summary:  resp,
		}, nil
	}

	return result, nil
}

// RecognizeEntities 实体识别
func (c *EinoClient) RecognizeEntities(ctx context.Context, text string, entityTypes []string) (*EntityRecognitionResult, error) {
	if len(entityTypes) == 0 {
		entityTypes = []string{"TASK", "SKILL", "CONCEPT", "EVENT", "PERSON", "ORGANIZATION"}
	}

	systemPrompt := fmt.Sprintf(`你是一个实体识别专家。请识别文本中的实体。

实体类型：%v

输出JSON格式：
{
  "entities": [
    {
      "name": "实体名称",
      "type": "实体类型",
      "description": "实体描述",
      "confidence": 0.95
    }
  ]
}

请确保输出是有效的JSON格式。`, entityTypes)

	userPrompt := fmt.Sprintf("请识别以下文本中的实体：\n\n%s", text)

	resp, err := c.ChatWithSystem(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("recognize entities: %w", err)
	}

	// 解析JSON响应
	result, err := parseEntityRecognitionResult(resp)
	if err != nil {
		c.log.Warn("Failed to parse entity result, using fallback", "error", err)
		return &EntityRecognitionResult{Entities: []Entity{}}, nil
	}

	return result, nil
}

// SummarizeCommunity 生成社区摘要
func (c *EinoClient) SummarizeCommunity(ctx context.Context, communityID string, nodeDescriptions []string) (*SummarizeCommunityResult, error) {
	systemPrompt := `你是一个知识图谱社区摘要专家。请根据社区内的节点描述生成社区摘要。

输出JSON格式：
{
  "community_id": "社区ID",
  "summary": "社区核心内容摘要",
  "key_topics": ["主题1", "主题2", "主题3"],
  "importance_score": 0.85
}

请确保输出是有效的JSON格式。`

	userPrompt := fmt.Sprintf("社区ID: %s\n\n节点描述：\n%s", communityID, nodeDescriptions)

	resp, err := c.ChatWithSystem(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("summarize community: %w", err)
	}

	// 解析JSON响应
	result, err := parseCommunitySummary(resp, communityID)
	if err != nil {
		c.log.Warn("Failed to parse community summary, using fallback", "error", err)
		return &SummarizeCommunityResult{
			CommunityID:     communityID,
			Summary:         resp,
			KeyTopics:       []string{},
			ImportanceScore: 0.5,
		}, nil
	}

	return result, nil
}

// HealthCheck 健康检查
func (c *EinoClient) HealthCheck(ctx context.Context) error {
	// 简单的测试调用
	_, err := c.ChatWithSystem(ctx, "You are a health checker.", "Reply with 'ok'")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}

// ==================== 辅助函数 ====================

// parseExtractionResult 解析提取结果
func parseExtractionResult(content string) (*ExtractionResult, error) {
	// 尝试提取JSON
	jsonStr := extractJSON(content)

	var result ExtractionResult
	if err := jsonUnmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	return &result, nil
}

// parseEntityRecognitionResult 解析实体识别结果
func parseEntityRecognitionResult(content string) (*EntityRecognitionResult, error) {
	jsonStr := extractJSON(content)

	var result EntityRecognitionResult
	if err := jsonUnmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	return &result, nil
}

// parseCommunitySummary 解析社区摘要
func parseCommunitySummary(content, communityID string) (*SummarizeCommunityResult, error) {
	jsonStr := extractJSON(content)

	var result SummarizeCommunityResult
	if err := jsonUnmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	if result.CommunityID == "" {
		result.CommunityID = communityID
	}

	return &result, nil
}

// EinoService 基于eino的LLM服务
type EinoService struct {
	client *EinoClient
	log    *logger.Logger

	// 嵌入缓存
	embeddingCache sync.Map
}

// NewEinoService 创建eino服务
func NewEinoService(config *EinoConfig) (*EinoService, error) {
	client, err := NewEinoClient(config)
	if err != nil {
		return nil, err
	}

	return &EinoService{
		client: client,
		log:    logger.NewLogger("info"),
	}, nil
}

// ExtractKnowledgeFromText 从文本提取知识
func (s *EinoService) ExtractKnowledgeFromText(ctx context.Context, text string) (*KnowledgeExtraction, error) {
	s.log.Debug("Extracting knowledge from text", "length", len(text))

	// 1. 提取三元组
	extraction, err := s.client.ExtractTriples(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("extract triples: %w", err)
	}

	// 2. 识别实体
	entities, err := s.client.RecognizeEntities(ctx, text, nil)
	if err != nil {
		s.log.Warn("Entity recognition failed", "error", err)
	}

	// 3. 合并实体
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

// GenerateEmbedding 生成嵌入向量
func (s *EinoService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
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

// GenerateBatchEmbeddings 批量生成嵌入
func (s *EinoService) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	return s.client.GenerateBatchEmbeddings(ctx, texts)
}

// HealthCheck 健康检查
func (s *EinoService) HealthCheck(ctx context.Context) error {
	return s.client.HealthCheck(ctx)
}

// mergeEntities 合并实体
func mergeEntities(entities1 []ExtractedEntity, entities2 []Entity) []*ExtractedEntity {
	entityMap := make(map[string]*ExtractedEntity)

	for _, e := range entities1 {
		key := e.Name + "_" + e.Type
		entityMap[key] = &ExtractedEntity{
			Name:        e.Name,
			Type:        e.Type,
			Description: e.Description,
		}
	}

	for _, e := range entities2 {
		key := e.Name + "_" + e.Type
		if _, exists := entityMap[key]; !exists {
			entityMap[key] = &ExtractedEntity{
				Name:        e.Name,
				Type:        e.Type,
				Description: e.Description,
			}
		}
	}

	result := make([]*ExtractedEntity, 0, len(entityMap))
	for _, e := range entityMap {
		result = append(result, e)
	}

	return result
}

// ==================== 辅助函数 ====================

func ptrFloat32(v float32) *float32 {
	return &v
}

func ptrInt(v int) *int {
	return &v
}
