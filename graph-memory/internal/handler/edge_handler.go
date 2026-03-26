package handler

import (
	"net/http"
	"strconv"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/service"
	"github.com/gin-gonic/gin"
)

// EdgeHandler 边处理器
type EdgeHandler struct {
	edgeService *service.EdgeService
}

// NewEdgeHandler 创建边处理器
func NewEdgeHandler(edgeService *service.EdgeService) *EdgeHandler {
	return &EdgeHandler{
		edgeService: edgeService,
	}
}

// CreateEdge 创建边
// @Summary 创建边
// @Description 创建一个新的关系边
// @Tags edges
// @Accept json
// @Produce json
// @Param edge body service.CreateEdgeRequest true "边信息"
// @Success 201 {object} model.Edge
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/edges [post]
func (h *EdgeHandler) CreateEdge(c *gin.Context) {
	var req service.CreateEdgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	edge, err := h.edgeService.CreateEdge(c.Request.Context(), &req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, edge)
}

// GetEdge 获取边
// @Summary 获取边
// @Description 根据ID获取边详情
// @Tags edges
// @Produce json
// @Param id path string true "边ID"
// @Success 200 {object} model.Edge
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/edges/{id} [get]
func (h *EdgeHandler) GetEdge(c *gin.Context) {
	id := c.Param("id")

	edge, err := h.edgeService.GetEdge(c.Request.Context(), id)
	if err != nil {
		RespondWithError(c, http.StatusNotFound, "Edge not found")
		return
	}

	c.JSON(http.StatusOK, edge)
}

// UpdateEdge 更新边
// @Summary 更新边
// @Description 更新边信息
// @Tags edges
// @Accept json
// @Produce json
// @Param id path string true "边ID"
// @Param edge body service.UpdateEdgeRequest true "边更新信息"
// @Success 200 {object} model.Edge
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/edges/{id} [put]
func (h *EdgeHandler) UpdateEdge(c *gin.Context) {
	id := c.Param("id")

	var req service.UpdateEdgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	edge, err := h.edgeService.UpdateEdge(c.Request.Context(), id, &req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, edge)
}

// DeleteEdge 删除边
// @Summary 删除边
// @Description 根据ID删除边
// @Tags edges
// @Param id path string true "边ID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/edges/{id} [delete]
func (h *EdgeHandler) DeleteEdge(c *gin.Context) {
	id := c.Param("id")

	if err := h.edgeService.DeleteEdge(c.Request.Context(), id); err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// ListEdges 列出边
// @Summary 列出边
// @Description 分页列出边
// @Tags edges
// @Produce json
// @Param node_id query string false "节点ID"
// @Param type query string false "边类型"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} service.ListEdgesResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/edges [get]
func (h *EdgeHandler) ListEdges(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	req := &service.ListEdgesRequest{
		NodeID:   c.Query("node_id"),
		Type:     model.EdgeType(c.Query("type")),
		Page:     page,
		PageSize: pageSize,
	}

	response, err := h.edgeService.ListEdges(c.Request.Context(), req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, response)
}
