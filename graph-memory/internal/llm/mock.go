package llm

import (
	"context"
	"io"
	"sync"

	"github.com/cloudwego/eino/schema"
)

// ==================== Mock接口定义 ====================

// ChatModel Chat模型接口（用于Mock）
type ChatModel interface {
	Generate(ctx context.Context, messages []*schema.Message) (*schema.Message, error)
	Stream(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error)
}

// EmbeddingModel Embedding模型接口（用于Mock）
type EmbeddingModel interface {
	EmbedStrings(ctx context.Context, texts []string) ([][]float32, error)
}

// ==================== Mock实现 ====================

// MockChatModel Mock Chat模型
type MockChatModel struct {
	mu              sync.Mutex
	Responses       []*schema.Message
	StreamResponses [][]*schema.Message
	CallCount       int
	LastMessages    []*schema.Message
	Error           error
	StreamError     error
}

// NewMockChatModel 创建Mock Chat模型
func NewMockChatModel() *MockChatModel {
	return &MockChatModel{
		Responses:       make([]*schema.Message, 0),
		StreamResponses: make([][]*schema.Message, 0),
	}
}

// Generate Mock生成方法
func (m *MockChatModel) Generate(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CallCount++
	m.LastMessages = messages

	if m.Error != nil {
		return nil, m.Error
	}

	if len(m.Responses) == 0 {
		return schema.AssistantMessage("mock response", nil), nil
	}

	// 返回第一个响应，并从列表中移除
	resp := m.Responses[0]
	m.Responses = m.Responses[1:]
	return resp, nil
}

// Stream Mock流式生成方法
func (m *MockChatModel) Stream(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CallCount++
	m.LastMessages = messages

	if m.StreamError != nil {
		return nil, m.StreamError
	}

	// 获取响应块
	var chunks []*schema.Message
	if len(m.StreamResponses) > 0 {
		chunks = m.StreamResponses[0]
		m.StreamResponses = m.StreamResponses[1:]
	} else {
		chunks = []*schema.Message{
			schema.AssistantMessage("mock ", nil),
			schema.AssistantMessage("stream ", nil),
			schema.AssistantMessage("response", nil),
		}
	}

	// 创建StreamReader
	return schema.StreamReaderFromArray(chunks), nil
}

// AddResponse 添加响应
func (m *MockChatModel) AddResponse(content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Responses = append(m.Responses, schema.AssistantMessage(content, nil))
}

// AddJSONResponse 添加JSON格式响应
func (m *MockChatModel) AddJSONResponse(jsonContent string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Responses = append(m.Responses, schema.AssistantMessage(jsonContent, nil))
}

// SetError 设置错误
func (m *MockChatModel) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Error = err
}

// MockEmbeddingModel Mock Embedding模型
type MockEmbeddingModel struct {
	mu         sync.Mutex
	Embeddings [][]float32
	CallCount  int
	LastTexts  []string
	Error      error
	DefaultDim int
}

// NewMockEmbeddingModel 创建Mock Embedding模型
func NewMockEmbeddingModel() *MockEmbeddingModel {
	return &MockEmbeddingModel{
		Embeddings: make([][]float32, 0),
		DefaultDim: 2048,
	}
}

// EmbedStrings Mock嵌入方法
func (m *MockEmbeddingModel) EmbedStrings(ctx context.Context, texts []string) ([][]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CallCount++
	m.LastTexts = texts

	if m.Error != nil {
		return nil, m.Error
	}

	if len(m.Embeddings) > 0 {
		result := m.Embeddings
		m.Embeddings = make([][]float32, 0)
		return result, nil
	}

	// 生成默认嵌入向量
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = generateMockEmbedding(m.DefaultDim, texts[i])
	}

	return result, nil
}

// AddEmbedding 添加嵌入向量
func (m *MockEmbeddingModel) AddEmbedding(embedding []float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Embeddings = append(m.Embeddings, embedding)
}

// SetError 设置错误
func (m *MockEmbeddingModel) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Error = err
}

// ==================== 辅助函数 ====================

// generateMockEmbedding 生成Mock嵌入向量
func generateMockEmbedding(dim int, seed string) []float32 {
	embedding := make([]float32, dim)
	// 使用种子字符串生成确定性的向量
	for i := range embedding {
		embedding[i] = float32((i+len(seed))%1000) / 1000.0
	}
	return embedding
}

// ==================== Mock EinoClient ====================

// MockEinoClient Mock Eino客户端
type MockEinoClient struct {
	ChatModel *MockChatModel
	Embed     *MockEmbeddingModel
	Model     string
	EmbedDim  int
}

// NewMockEinoClient 创建Mock Eino客户端
func NewMockEinoClient() *MockEinoClient {
	return &MockEinoClient{
		ChatModel: NewMockChatModel(),
		Embed:     NewMockEmbeddingModel(),
		Model:     "mock-model",
		EmbedDim:  2048,
	}
}

// Chat 同步聊天
func (c *MockEinoClient) Chat(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	return c.ChatModel.Generate(ctx, messages)
}

// ChatStream 流式聊天
func (c *MockEinoClient) ChatStream(ctx context.Context, messages []*schema.Message) (<-chan *schema.Message, <-chan error) {
	msgChan := make(chan *schema.Message, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(msgChan)
		defer close(errChan)

		stream, err := c.ChatModel.Stream(ctx, messages)
		if err != nil {
			errChan <- err
			return
		}
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
func (c *MockEinoClient) ChatWithSystem(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userMessage),
	}

	resp, err := c.ChatModel.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// GenerateEmbedding 生成嵌入向量
func (c *MockEinoClient) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	embeddings, err := c.Embed.EmbedStrings(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, io.EOF
	}

	// 转换为float64
	result := make([]float64, len(embeddings[0]))
	for i, v := range embeddings[0] {
		result[i] = float64(v)
	}

	return result, nil
}

// GenerateBatchEmbeddings 批量生成嵌入向量
func (c *MockEinoClient) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	embeddings, err := c.Embed.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, err
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

// ExtractTriples 提取三元组
func (c *MockEinoClient) ExtractTriples(ctx context.Context, text string) (*ExtractionResult, error) {
	resp, err := c.ChatWithSystem(ctx, "system prompt", text)
	if err != nil {
		return nil, err
	}

	// 解析JSON响应
	result, err := parseExtractionResult(resp)
	if err != nil {
		return &ExtractionResult{
			Triples:  []Triple{},
			Entities: []ExtractedEntity{},
			Summary:  resp,
		}, nil
	}

	return result, nil
}

// RecognizeEntities 实体识别
func (c *MockEinoClient) RecognizeEntities(ctx context.Context, text string, entityTypes []string) (*EntityRecognitionResult, error) {
	resp, err := c.ChatWithSystem(ctx, "entity recognition", text)
	if err != nil {
		return nil, err
	}

	result, err := parseEntityRecognitionResult(resp)
	if err != nil {
		return &EntityRecognitionResult{Entities: []Entity{}}, nil
	}

	return result, nil
}

// SummarizeCommunity 社区摘要
func (c *MockEinoClient) SummarizeCommunity(ctx context.Context, communityID string, nodeDescriptions []string) (*SummarizeCommunityResult, error) {
	resp, err := c.ChatWithSystem(ctx, "summarize", communityID)
	if err != nil {
		return nil, err
	}

	result, err := parseCommunitySummary(resp, communityID)
	if err != nil {
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
func (c *MockEinoClient) HealthCheck(ctx context.Context) error {
	return nil
}

// ==================== Mock服务 ====================

// MockService Mock LLM服务
type MockService struct {
	client *MockEinoClient
}

// NewMockService 创建Mock服务
func NewMockService() *MockService {
	return &MockService{
		client: NewMockEinoClient(),
	}
}

// GetClient 获取Mock客户端
func (s *MockService) GetClient() *MockEinoClient {
	return s.client
}

// ExtractKnowledgeFromText 提取知识
func (s *MockService) ExtractKnowledgeFromText(ctx context.Context, text string) (*KnowledgeExtraction, error) {
	result, err := s.client.ExtractTriples(ctx, text)
	if err != nil {
		return nil, err
	}

	entities := make([]*ExtractedEntity, len(result.Entities))
	for i, e := range result.Entities {
		entities[i] = &ExtractedEntity{
			Name:        e.Name,
			Type:        e.Type,
			Description: e.Description,
		}
	}

	triples := make([]*Triple, len(result.Triples))
	for i, t := range result.Triples {
		triples[i] = &Triple{
			Subject:     t.Subject,
			Predicate:   t.Predicate,
			Object:      t.Object,
			SubjectType: t.SubjectType,
			ObjectType:  t.ObjectType,
			Confidence:  t.Confidence,
		}
	}

	return &KnowledgeExtraction{
		Triples:  triples,
		Entities: entities,
		Summary:  result.Summary,
	}, nil
}

// GenerateEmbedding 生成嵌入
func (s *MockService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	return s.client.GenerateEmbedding(ctx, text)
}

// HealthCheck 健康检查
func (s *MockService) HealthCheck(ctx context.Context) error {
	return s.client.HealthCheck(ctx)
}
