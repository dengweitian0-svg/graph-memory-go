package handler

import (
	"net/http"

	"github.com/example/graph-memory/internal/llm"
	"github.com/example/graph-memory/internal/model"
	"github.com/gin-gonic/gin"
)

// LLMHandler LLM服务处理器
type LLMHandler struct {
	llmService *llm.Service
	extractor  *llm.EnhancedExtractor
	disabled   bool // 标记服务是否被禁用
}

// NewLLMHandler 创建LLM处理器
func NewLLMHandler(llmService *llm.Service) *LLMHandler {
	return &LLMHandler{
		llmService: llmService,
		extractor:  llm.NewEnhancedExtractor(llmService),
		disabled:   false,
	}
}

// NewDisabledLLMHandler 创建禁用状态的LLM处理器
func NewDisabledLLMHandler() *LLMHandler {
	return &LLMHandler{
		llmService: nil,
		extractor:  nil,
		disabled:   true,
	}
}

// IsDisabled 检查服务是否被禁用
func (h *LLMHandler) IsDisabled() bool {
	return h.disabled
}

// ExtractTriples 提取三元组
// @Summary 从文本中提取知识三元组
// @Description 使用LLM从文本中提取实体-关系-实体三元组
// @Tags llm
// @Accept json
// @Produce json
// @Param request body ExtractTriplesRequest true "提取请求"
// @Success 200 {object} llm.KnowledgeExtraction
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/extract [post]
func (h *LLMHandler) ExtractTriples(c *gin.Context) {
	if h.disabled {
		RespondWithError(c, http.StatusServiceUnavailable, "LLM service is disabled. Please enable it in config.yaml and provide a valid API key")
		return
	}

	var req ExtractTriplesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Text == "" {
		RespondWithError(c, http.StatusBadRequest, "Text is required")
		return
	}

	result, err := h.llmService.ExtractKnowledgeFromText(c.Request.Context(), req.Text)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

// GenerateEmbeddings 生成嵌入向量
// @Summary 生成文本嵌入向量
// @Description 为文本生成语义向量表示
// @Tags llm
// @Accept json
// @Produce json
// @Param request body GenerateEmbeddingsRequest true "嵌入请求"
// @Success 200 {object} EmbeddingResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/embeddings [post]
func (h *LLMHandler) GenerateEmbeddings(c *gin.Context) {
	if h.disabled {
		RespondWithError(c, http.StatusServiceUnavailable, "LLM service is disabled. Please enable it in config.yaml and provide a valid API key")
		return
	}

	var req GenerateEmbeddingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Texts) == 0 {
		RespondWithError(c, http.StatusBadRequest, "Texts are required")
		return
	}

	// 批量生成嵌入
	embeddings := make([][]float64, len(req.Texts))
	for i, text := range req.Texts {
		embedding, err := h.llmService.GenerateEmbedding(c.Request.Context(), text)
		if err != nil {
			RespondWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
		embeddings[i] = embedding
	}

	c.JSON(http.StatusOK, EmbeddingResponse{
		Embeddings: embeddings,
		Model:      "doubao-embedding-vision-251215",
		Dimension:  2048,
	})
}

// SummarizeCommunity 社区摘要
// @Summary 生成社区摘要
// @Description 为知识图谱社区生成摘要
// @Tags llm
// @Accept json
// @Produce json
// @Param request body SummarizeCommunityRequest true "摘要请求"
// @Success 200 {object} llm.CommunitySummary
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/summarize [post]
func (h *LLMHandler) SummarizeCommunity(c *gin.Context) {
	if h.disabled {
		RespondWithError(c, http.StatusServiceUnavailable, "LLM service is disabled. Please enable it in config.yaml and provide a valid API key")
		return
	}

	var req SummarizeCommunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 转换节点信息
	nodes := make([]*model.Node, len(req.Nodes))
	for i, n := range req.Nodes {
		nodes[i] = &model.Node{
			ID:          n.ID,
			Name:        n.Name,
			Type:        model.NodeType(n.Type),
			Description: n.Description,
		}
	}

	result, err := h.llmService.SummarizeCommunity(
		c.Request.Context(),
		req.CommunityID,
		nodes,
	)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

// RecognizeEntities 实体识别
// @Summary 识别文本中的实体
// @Description 使用LLM识别文本中的特定类型实体
// @Tags llm
// @Accept json
// @Produce json
// @Param request body RecognizeEntitiesRequest true "识别请求"
// @Success 200 {object} EntityRecognitionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/llm/entities [post]
func (h *LLMHandler) RecognizeEntities(c *gin.Context) {
	if h.disabled {
		RespondWithError(c, http.StatusServiceUnavailable, "LLM service is disabled. Please enable it in config.yaml and provide a valid API key")
		return
	}

	var req RecognizeEntitiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Text == "" {
		RespondWithError(c, http.StatusBadRequest, "Text is required")
		return
	}

	// 使用知识提取来识别实体
	result, err := h.llmService.ExtractKnowledgeFromText(c.Request.Context(), req.Text)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// 转换为实体识别结果
	entities := make([]EntityInfo, len(result.Entities))
	for i, e := range result.Entities {
		entities[i] = EntityInfo{
			Name:        e.Name,
			Type:        e.Type,
			Description: e.Description,
		}
	}

	c.JSON(http.StatusOK, EntityRecognitionResponse{
		Entities: entities,
	})
}

// LLMHealthCheck LLM服务健康检查
// @Summary LLM服务健康检查
// @Description 检查LLM服务是否可用
// @Tags llm
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} ErrorResponse
// @Router /api/v1/llm/health [get]
func (h *LLMHandler) LLMHealthCheck(c *gin.Context) {
	// 如果服务被禁用，返回禁用状态
	if h.disabled {
		c.JSON(http.StatusOK, gin.H{
			"status":   "disabled",
			"service":  "llm",
			"provider": "eino",
			"message":  "LLM service is disabled. Enable it in config.yaml with a valid API key",
		})
		return
	}

	if err := h.llmService.HealthCheck(c.Request.Context()); err != nil {
		RespondWithError(c, http.StatusServiceUnavailable, "LLM service unavailable: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"service":  "llm",
		"provider": "eino",
	})
}

// ==================== 请求/响应模型 ====================

// ExtractTriplesRequest 提取三元组请求
type ExtractTriplesRequest struct {
	Text string `json:"text"`
}

// GenerateEmbeddingsRequest 生成嵌入请求
type GenerateEmbeddingsRequest struct {
	Texts []string `json:"texts"`
}

// EmbeddingResponse 嵌入响应
type EmbeddingResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
	Model      string      `json:"model"`
	Dimension  int         `json:"dimension"`
}

// SummarizeCommunityRequest 社区摘要请求
type SummarizeCommunityRequest struct {
	CommunityID string         `json:"community_id"`
	Nodes       []*llmNodeInfo `json:"nodes"`
}

type llmNodeInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// RecognizeEntitiesRequest 实体识别请求
type RecognizeEntitiesRequest struct {
	Text        string   `json:"text"`
	EntityTypes []string `json:"entity_types,omitempty"`
}

// EntityRecognitionResponse 实体识别响应
type EntityRecognitionResponse struct {
	Entities []EntityInfo `json:"entities"`
}

// EntityInfo 实体信息
type EntityInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}
