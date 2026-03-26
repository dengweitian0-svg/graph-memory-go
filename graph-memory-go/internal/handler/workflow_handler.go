package handler

import (
	"net/http"
	"strconv"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/service"
	"github.com/example/graph-memory/internal/workflow"
	"github.com/gin-gonic/gin"
)

// WorkflowHandler 工作流处理器
type WorkflowHandler struct {
	orchestrator *workflow.Orchestrator
	sessionSvc   *service.SessionService
}

// NewWorkflowHandler 创建工作流处理器
func NewWorkflowHandler(orchestrator *workflow.Orchestrator, sessionSvc *service.SessionService) *WorkflowHandler {
	return &WorkflowHandler{
		orchestrator: orchestrator,
		sessionSvc:   sessionSvc,
	}
}

// ExecuteWorkflow 执行工作流
// @Summary 执行工作流
// @Description 执行指定类型的工作流
// @Tags workflows
// @Accept json
// @Produce json
// @Param request body ExecuteWorkflowRequest true "工作流请求"
// @Success 200 {object} workflow.WorkflowResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/workflows/execute [post]
func (h *WorkflowHandler) ExecuteWorkflow(c *gin.Context) {
	var req ExecuteWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	config := &workflow.WorkflowConfig{
		Type:                req.Type,
		SessionID:           req.SessionID,
		MessageIDs:          req.MessageIDs,
		EnableExtraction:    req.EnableExtraction,
		EnableGraphBuilding: req.EnableGraphBuilding,
		EnableAlgorithms:    req.EnableAlgorithms,
		EnableDeduplication: req.EnableDeduplication,
		AlgorithmThreshold:  req.AlgorithmThreshold,
	}

	if config.Type == "" {
		config.Type = workflow.WorkflowTypeFull
	}

	result, err := h.orchestrator.ExecuteWorkflow(c.Request.Context(), config)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetWorkflowStatus 获取工作流状态
// @Summary 获取工作流状态
// @Description 获取正在执行的工作流状态
// @Tags workflows
// @Produce json
// @Param id path string true "工作流ID"
// @Success 200 {object} workflow.Workflow
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/workflows/{id}/status [get]
func (h *WorkflowHandler) GetWorkflowStatus(c *gin.Context) {
	id := c.Param("id")

	workflow, err := h.orchestrator.GetWorkflowStatus(id)
	if err != nil {
		RespondWithError(c, http.StatusNotFound, "Workflow not found")
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// CancelWorkflow 取消工作流
// @Summary 取消工作流
// @Description 取消正在执行的工作流
// @Tags workflows
// @Param id path string true "工作流ID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/workflows/{id}/cancel [post]
func (h *WorkflowHandler) CancelWorkflow(c *gin.Context) {
	id := c.Param("id")

	if err := h.orchestrator.CancelWorkflow(id); err != nil {
		RespondWithError(c, http.StatusNotFound, "Workflow not found")
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateSession 创建会话
// @Summary 创建会话
// @Description 创建一个新的会话
// @Tags sessions
// @Produce json
// @Success 201 {object} model.Session
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sessions [post]
func (h *WorkflowHandler) CreateSession(c *gin.Context) {
	session, err := h.sessionSvc.CreateSession(c.Request.Context())
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, session)
}

// GetSession 获取会话
// @Summary 获取会话
// @Description 根据ID获取会话详情（包含消息）
// @Tags sessions
// @Produce json
// @Param id path string true "会话ID"
// @Success 200 {object} model.SessionWithMessages
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/sessions/{id} [get]
func (h *WorkflowHandler) GetSession(c *gin.Context) {
	id := c.Param("id")

	result, err := h.sessionSvc.GetSession(c.Request.Context(), id)
	if err != nil {
		RespondWithError(c, http.StatusNotFound, "Session not found")
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListSessions 列出会话
// @Summary 列出会话
// @Description 分页列出会话
// @Tags sessions
// @Produce json
// @Param status query string false "会话状态"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} service.ListSessionsResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sessions [get]
func (h *WorkflowHandler) ListSessions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	req := &service.ListSessionsRequest{
		Status:   model.SessionStatus(c.Query("status")),
		Page:     page,
		PageSize: pageSize,
	}

	response, err := h.sessionSvc.ListSessions(c.Request.Context(), req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, response)
}

// AddMessage 添加消息
// @Summary 添加消息
// @Description 向会话添加消息
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "会话ID"
// @Param message body service.AddMessageRequest true "消息内容"
// @Success 201 {object} model.Message
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/sessions/{id}/messages [post]
func (h *WorkflowHandler) AddMessage(c *gin.Context) {
	sessionID := c.Param("id")

	var req service.AddMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	message, err := h.sessionSvc.AddMessage(c.Request.Context(), sessionID, &req)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, message)
}

// CompleteSession 完成会话
// @Summary 完成会话
// @Description 标记会话为已完成
// @Tags sessions
// @Param id path string true "会话ID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/sessions/{id}/complete [post]
func (h *WorkflowHandler) CompleteSession(c *gin.Context) {
	id := c.Param("id")

	if err := h.sessionSvc.CompleteSession(c.Request.Context(), id); err != nil {
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// ExecuteWorkflowRequest 执行工作流请求
type ExecuteWorkflowRequest struct {
	Type                workflow.WorkflowType `json:"type"`
	SessionID           string                `json:"session_id,omitempty"`
	MessageIDs          []string              `json:"message_ids,omitempty"`
	EnableExtraction    bool                  `json:"enable_extraction"`
	EnableGraphBuilding bool                  `json:"enable_graph_building"`
	EnableAlgorithms    bool                  `json:"enable_algorithms"`
	EnableDeduplication bool                  `json:"enable_deduplication"`
	AlgorithmThreshold  float64               `json:"algorithm_threshold,omitempty"`
}
