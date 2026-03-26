package handler

import (
	"net/http"
	"strconv"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/service"
	"github.com/gin-gonic/gin"
)

// RecallHandler 召回处理器
type RecallHandler struct {
	recallService *service.RecallService
}

// NewRecallHandler 创建召回处理器
func NewRecallHandler(recallService *service.RecallService) *RecallHandler {
	return &RecallHandler{
		recallService: recallService,
	}
}

// Recall 执行知识召回
// @Summary 知识召回
// @Description 使用双路径召回策略（精确路径+泛化路径）检索相关知识节点
// @Tags recall
// @Accept json
// @Produce json
// @Param request body model.RecallRequest true "召回请求"
// @Success 200 {object} model.RecallResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/recall [post]
func (h *RecallHandler) Recall(c *gin.Context) {
	var req model.RecallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// 验证请求
	if req.Query == "" {
		RespondWithError(c, http.StatusBadRequest, "Query is required")
		return
	}

	// 执行召回
	response, err := h.recallService.Recall(c.Request.Context(), &req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "Recall failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, response)
}

// BuildContext 构建上下文
// @Summary 构建上下文
// @Description 根据召回结果构建上下文文本
// @Tags recall
// @Accept json
// @Produce json
// @Param request body model.ContextBuildRequest true "上下文构建请求"
// @Success 200 {object} model.ContextBuilder
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/recall/context [post]
func (h *RecallHandler) BuildContext(c *gin.Context) {
	var req model.ContextBuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// 构建上下文
	response, err := h.recallService.BuildContext(c.Request.Context(), &req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "Build context failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetRecallConfig 获取召回配置
// @Summary 获取召回配置
// @Description 获取当前召回服务的配置
// @Tags recall
// @Produce json
// @Success 200 {object} model.RecallConfig
// @Router /api/v1/recall/config [get]
func (h *RecallHandler) GetRecallConfig(c *gin.Context) {
	config := h.recallService.GetRecallConfig()
	c.JSON(http.StatusOK, config)
}

// UpdateRecallConfig 更新召回配置
// @Summary 更新召回配置
// @Description 更新召回服务的配置
// @Tags recall
// @Accept json
// @Produce json
// @Param request body model.RecallConfig true "召回配置"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/recall/config [put]
func (h *RecallHandler) UpdateRecallConfig(c *gin.Context) {
	var config model.RecallConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	h.recallService.SetRecallConfig(&config)
	c.JSON(http.StatusOK, gin.H{"message": "Config updated successfully"})
}

// QuickRecall 快速召回（简化接口）
// @Summary 快速召回
// @Description 使用默认配置进行知识召回
// @Tags recall
// @Produce json
// @Param query query string true "查询文本"
// @Param limit query int false "返回数量限制" default(10)
// @Success 200 {object} model.RecallResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/recall/quick [get]
func (h *RecallHandler) QuickRecall(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		RespondWithError(c, http.StatusBadRequest, "Query parameter is required")
		return
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil {
			limit = parsedLimit
		}
	}

	req := &model.RecallRequest{
		Query: query,
		Limit: limit,
	}

	response, err := h.recallService.Recall(c.Request.Context(), req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "Recall failed: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, response)
}

// VectorSearch 向量搜索
// @Summary 向量搜索
// @Description 使用向量相似度搜索节点
// @Tags recall
// @Accept json
// @Produce json
// @Param request body VectorSearchAPIRequest true "向量搜索请求"
// @Success 200 {object} VectorSearchAPIResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/recall/vector [post]
func (h *RecallHandler) VectorSearch(c *gin.Context) {
	var req VectorSearchAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.Text == "" {
		RespondWithError(c, http.StatusBadRequest, "Text is required")
		return
	}

	// 通过召回服务执行搜索
	response, err := h.recallService.Recall(c.Request.Context(), &model.RecallRequest{
		Query:                 req.Text,
		Limit:                 req.Limit,
		EnablePrecisePath:     true,
		EnableGeneralizedPath: false, // 仅向量搜索
	})

	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, "Vector search failed: "+err.Error())
		return
	}

	// 转换响应
	apiResponse := &VectorSearchAPIResponse{
		Results: make([]*VectorSearchResultItem, len(response.Results)),
		Total:   response.Total,
	}

	for i, r := range response.Results {
		apiResponse.Results[i] = &VectorSearchResultItem{
			NodeID:      r.Node.ID,
			Name:        r.Node.Name,
			Type:        string(r.Node.Type),
			Description: r.Node.Description,
			Score:       r.Score,
		}
	}

	c.JSON(http.StatusOK, apiResponse)
}

// VectorSearchAPIRequest 向量搜索API请求
type VectorSearchAPIRequest struct {
	Text  string `json:"text"`
	Limit int    `json:"limit,omitempty"`
}

// VectorSearchAPIResponse 向量搜索API响应
type VectorSearchAPIResponse struct {
	Results []*VectorSearchResultItem `json:"results"`
	Total   int                       `json:"total"`
}

// VectorSearchResultItem 向量搜索结果项
type VectorSearchResultItem struct {
	NodeID      string  `json:"node_id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}
