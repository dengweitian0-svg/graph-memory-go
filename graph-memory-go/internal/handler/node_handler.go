package handler

import (
	"net/http"
	"strconv"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/service"
	"github.com/gin-gonic/gin"
)

// NodeHandler 节点处理器
type NodeHandler struct {
	nodeService *service.NodeService
}

// NewNodeHandler 创建节点处理器
func NewNodeHandler(nodeService *service.NodeService) *NodeHandler {
	return &NodeHandler{
		nodeService: nodeService,
	}
}

// CreateNode 创建节点
// @Summary 创建节点
// @Description 创建一个新的知识节点
// @Tags nodes
// @Accept json
// @Produce json
// @Param node body service.CreateNodeRequest true "节点信息"
// @Success 201 {object} model.Node
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/nodes [post]
func (h *NodeHandler) CreateNode(c *gin.Context) {
	var req service.CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	node, err := h.nodeService.CreateNode(c.Request.Context(), &req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, node)
}

// GetNode 获取节点
// @Summary 获取节点
// @Description 根据ID获取节点详情
// @Tags nodes
// @Produce json
// @Param id path string true "节点ID"
// @Success 200 {object} model.Node
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/nodes/{id} [get]
func (h *NodeHandler) GetNode(c *gin.Context) {
	id := c.Param("id")

	node, err := h.nodeService.GetNode(c.Request.Context(), id)
	if err != nil {
		RespondWithError(c, http.StatusNotFound, "Node not found")
		return
	}

	c.JSON(http.StatusOK, node)
}

// UpdateNode 更新节点
// @Summary 更新节点
// @Description 更新节点信息
// @Tags nodes
// @Accept json
// @Produce json
// @Param id path string true "节点ID"
// @Param node body service.UpdateNodeRequest true "节点更新信息"
// @Success 200 {object} model.Node
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/nodes/{id} [put]
func (h *NodeHandler) UpdateNode(c *gin.Context) {
	id := c.Param("id")

	var req service.UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	node, err := h.nodeService.UpdateNode(c.Request.Context(), id, &req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, node)
}

// DeleteNode 删除节点
// @Summary 删除节点
// @Description 根据ID删除节点
// @Tags nodes
// @Param id path string true "节点ID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/nodes/{id} [delete]
func (h *NodeHandler) DeleteNode(c *gin.Context) {
	id := c.Param("id")

	if err := h.nodeService.DeleteNode(c.Request.Context(), id); err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// ListNodes 列出节点
// @Summary 列出节点
// @Description 分页列出节点
// @Tags nodes
// @Produce json
// @Param type query string false "节点类型"
// @Param status query string false "节点状态"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} service.ListNodesResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/nodes [get]
func (h *NodeHandler) ListNodes(c *gin.Context) {
	// Parse page parameter with proper error handling
	page := 1
	if pageStr := c.DefaultQuery("page", "1"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage >= 1 {
			page = parsedPage
		}
	}

	// Parse page_size parameter with proper error handling
	pageSize := 10
	if pageSizeStr := c.DefaultQuery("page_size", "10"); pageSizeStr != "" {
		if parsedPageSize, err := strconv.Atoi(pageSizeStr); err == nil && parsedPageSize >= 1 && parsedPageSize <= 100 {
			pageSize = parsedPageSize
		}
	}

	req := &service.ListNodesRequest{
		Type:     model.NodeType(c.Query("type")),
		Status:   model.NodeStatus(c.Query("status")),
		Page:     page,
		PageSize: pageSize,
	}

	response, err := h.nodeService.ListNodes(c.Request.Context(), req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, response)
}

// SearchNodes 搜索节点
// @Summary 搜索节点
// @Description 根据关键词搜索节点
// @Tags nodes
// @Produce json
// @Param q query string true "搜索关键词"
// @Param limit query int false "返回数量限制" default(10)
// @Success 200 {object} service.SearchNodesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/nodes/search [get]
func (h *NodeHandler) SearchNodes(c *gin.Context) {
	searchQuery := c.Query("q")
	if searchQuery == "" {
		RespondWithError(c, http.StatusBadRequest, "Query parameter 'q' is required")
		return
	}

	// Parse limit parameter with proper error handling
	limit := 10
	if limitStr := c.DefaultQuery("limit", "10"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit >= 1 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	req := &service.SearchNodesRequest{
		Query: searchQuery,
		Limit: limit,
	}

	response, err := h.nodeService.SearchNodes(c.Request.Context(), req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, response)
}
