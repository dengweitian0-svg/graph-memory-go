package handler

import (
	"net/http"
	"strconv"

	"github.com/example/graph-memory/internal/service"
	"github.com/gin-gonic/gin"
)

// GraphHandler 图处理器
type GraphHandler struct {
	graphService *service.GraphService
}

// NewGraphHandler 创建图处理器
func NewGraphHandler(graphService *service.GraphService) *GraphHandler {
	return &GraphHandler{
		graphService: graphService,
	}
}

// GetSubgraph 获取子图
// @Summary 获取子图
// @Description 获取指定节点及其邻居的子图
// @Tags graph
// @Accept json
// @Produce json
// @Param request body service.GetSubgraphRequest true "子图请求"
// @Success 200 {object} model.Subgraph
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/graph/subgraph [post]
func (h *GraphHandler) GetSubgraph(c *gin.Context) {
	var req service.GetSubgraphRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	subgraph, err := h.graphService.GetSubgraph(c.Request.Context(), &req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, subgraph)
}

// GetNeighbors 获取邻居节点
// @Summary 获取邻居节点
// @Description 获取指定节点的邻居节点
// @Tags graph
// @Produce json
// @Param id path string true "节点ID"
// @Param depth query int false "深度" default(1)
// @Success 200 {object} service.GetNeighborsResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/graph/nodes/{id}/neighbors [get]
func (h *GraphHandler) GetNeighbors(c *gin.Context) {
	id := c.Param("id")
	depth, _ := strconv.Atoi(c.DefaultQuery("depth", "1"))

	req := &service.GetNeighborsRequest{
		NodeID: id,
		Depth:  depth,
	}

	response, err := h.graphService.GetNeighbors(c.Request.Context(), req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, response)
}

// FindPath 查找路径
// @Summary 查找路径
// @Description 查找两个节点之间的最短路径
// @Tags graph
// @Accept json
// @Produce json
// @Param request body service.FindPathRequest true "路径请求"
// @Success 200 {object} service.FindPathResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/graph/path [post]
func (h *GraphHandler) FindPath(c *gin.Context) {
	var req service.FindPathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.MaxDepth == 0 {
		req.MaxDepth = 5
	}

	response, err := h.graphService.FindPath(c.Request.Context(), &req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if response.Path == nil {
		RespondWithError(c, http.StatusNotFound, "Path not found")
		return
	}

	c.JSON(http.StatusOK, response)
}

// RunPageRank 运行PageRank算法
// @Summary 运行PageRank算法
// @Description 对指定节点运行PageRank算法
// @Tags algorithms
// @Accept json
// @Produce json
// @Param request body service.RunPageRankRequest true "PageRank请求"
// @Success 200 {object} algorithm.PageRankResult
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/algorithms/pagerank [post]
func (h *GraphHandler) RunPageRank(c *gin.Context) {
	var req service.RunPageRankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.graphService.RunPageRank(c.Request.Context(), &req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

// RunCommunityDetection 运行社区检测
// @Summary 运行社区检测
// @Description 对指定节点运行社区检测算法
// @Tags algorithms
// @Accept json
// @Produce json
// @Param request body service.RunCommunityDetectionRequest true "社区检测请求"
// @Success 200 {object} algorithm.CommunityResult
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/algorithms/community [post]
func (h *GraphHandler) RunCommunityDetection(c *gin.Context) {
	var req service.RunCommunityDetectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := h.graphService.RunCommunityDetection(c.Request.Context(), &req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

// RunDeduplication 运行去重检测
// @Summary 运行去重检测
// @Description 检测重复节点
// @Tags algorithms
// @Accept json
// @Produce json
// @Param request body service.RunDeduplicationRequest true "去重请求"
// @Success 200 {object} algorithm.DeduplicationResult
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/algorithms/deduplication [post]
func (h *GraphHandler) RunDeduplication(c *gin.Context) {
	var req service.RunDeduplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Threshold == 0 {
		req.Threshold = 0.95
	}

	result, err := h.graphService.RunDeduplication(c.Request.Context(), &req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

// RunFullPipeline 运行完整算法流水线
// @Summary 运行完整算法流水线
// @Description 运行PageRank、社区检测和去重算法
// @Tags algorithms
// @Produce json
// @Success 200 {object} algorithm.PipelineResult
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/algorithms/pipeline [post]
func (h *GraphHandler) RunFullPipeline(c *gin.Context) {
	result, err := h.graphService.RunFullPipeline(c.Request.Context())
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}
