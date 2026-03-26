package server

import (
	"context"
	"net/http"
	"time"

	"github.com/example/graph-memory/internal/handler"
	"github.com/example/graph-memory/internal/llm"
	"github.com/example/graph-memory/internal/middleware"
	"github.com/example/graph-memory/internal/service"
	"github.com/example/graph-memory/internal/workflow"
	"github.com/example/graph-memory/pkg/logger"
	"github.com/gin-gonic/gin"
)

// HTTPServer HTTP服务器
type HTTPServer struct {
	server *http.Server
	router *gin.Engine
	log    *logger.Logger
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(
	addr string,
	nodeService *service.NodeService,
	edgeService *service.EdgeService,
	graphService *service.GraphService,
	sessionService *service.SessionService,
	orchestrator *workflow.Orchestrator,
	llmService *llm.Service,
	recallService *service.RecallService,
	log *logger.Logger,
) *HTTPServer {
	// 设置Gin为release模式
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// 创建处理器
	nodeHandler := handler.NewNodeHandler(nodeService)
	edgeHandler := handler.NewEdgeHandler(edgeService)
	graphHandler := handler.NewGraphHandler(graphService)
	workflowHandler := handler.NewWorkflowHandler(orchestrator, sessionService)

	// 创建LLM处理器
	var llmHandler *handler.LLMHandler
	if llmService != nil {
		llmHandler = handler.NewLLMHandler(llmService)
	} else {
		// 即使LLM服务禁用，也创建一个禁用状态的处理器来注册路由
		llmHandler = handler.NewDisabledLLMHandler()
	}

	// 创建召回处理器
	var recallHandler *handler.RecallHandler
	if recallService != nil {
		recallHandler = handler.NewRecallHandler(recallService)
	}

	// 添加中间件
	router.Use(middleware.Recovery(log))
	router.Use(middleware.Logger(log))
	router.Use(middleware.CORS([]string{"*"}))

	// 健康检查
	router.GET("/health", handler.HealthCheck)
	router.GET("/ready", handler.ReadinessCheck)

	// API路由组
	api := router.Group("/api/v1")
	{
		// 节点路由
		nodes := api.Group("/nodes")
		{
			nodes.POST("", nodeHandler.CreateNode)
			nodes.GET("", nodeHandler.ListNodes)
			nodes.GET("/search", nodeHandler.SearchNodes)
			nodes.GET("/:id", nodeHandler.GetNode)
			nodes.PUT("/:id", nodeHandler.UpdateNode)
			nodes.DELETE("/:id", nodeHandler.DeleteNode)
		}

		// 边路由
		edges := api.Group("/edges")
		{
			edges.POST("", edgeHandler.CreateEdge)
			edges.GET("", edgeHandler.ListEdges)
			edges.GET("/:id", edgeHandler.GetEdge)
			edges.PUT("/:id", edgeHandler.UpdateEdge)
			edges.DELETE("/:id", edgeHandler.DeleteEdge)
		}

		// 图路由
		graph := api.Group("/graph")
		{
			graph.POST("/subgraph", graphHandler.GetSubgraph)
			graph.POST("/path", graphHandler.FindPath)
			graph.GET("/nodes/:id/neighbors", graphHandler.GetNeighbors)
		}

		// 算法路由
		algorithms := api.Group("/algorithms")
		{
			algorithms.POST("/pagerank", graphHandler.RunPageRank)
			algorithms.POST("/community", graphHandler.RunCommunityDetection)
			algorithms.POST("/deduplication", graphHandler.RunDeduplication)
			algorithms.POST("/pipeline", graphHandler.RunFullPipeline)
		}

		// 工作流路由
		workflows := api.Group("/workflows")
		{
			workflows.POST("/execute", workflowHandler.ExecuteWorkflow)
			workflows.GET("/:id/status", workflowHandler.GetWorkflowStatus)
			workflows.POST("/:id/cancel", workflowHandler.CancelWorkflow)
		}

		// 会话路由
		sessions := api.Group("/sessions")
		{
			sessions.POST("", workflowHandler.CreateSession)
			sessions.GET("", workflowHandler.ListSessions)
			sessions.GET("/:id", workflowHandler.GetSession)
			sessions.POST("/:id/complete", workflowHandler.CompleteSession)
			sessions.POST("/:id/messages", workflowHandler.AddMessage)
		}

		// LLM服务路由 (始终注册，禁用时返回友好错误)
		llm := api.Group("/llm")
		{
			llm.POST("/extract", llmHandler.ExtractTriples)
			llm.POST("/embeddings", llmHandler.GenerateEmbeddings)
			llm.POST("/summarize", llmHandler.SummarizeCommunity)
			llm.POST("/entities", llmHandler.RecognizeEntities)
			llm.GET("/health", llmHandler.LLMHealthCheck)
		}

		// 召回服务路由
		if recallHandler != nil {
			recall := api.Group("/recall")
			{
				recall.POST("", recallHandler.Recall)
				recall.GET("/quick", recallHandler.QuickRecall)
				recall.POST("/context", recallHandler.BuildContext)
				recall.POST("/vector", recallHandler.VectorSearch)
				recall.GET("/config", recallHandler.GetRecallConfig)
				recall.PUT("/config", recallHandler.UpdateRecallConfig)
			}
		}
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &HTTPServer{
		server: server,
		router: router,
		log:    log,
	}
}

// Start 启动服务器
func (s *HTTPServer) Start() error {
	s.log.Info("HTTP server starting", "addr", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown 关闭服务器
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.log.Info("HTTP server shutting down")
	return s.server.Shutdown(ctx)
}

// GetRouter 获取路由器
func (s *HTTPServer) GetRouter() *gin.Engine {
	return s.router
}
