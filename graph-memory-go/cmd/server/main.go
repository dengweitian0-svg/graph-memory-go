package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/graph-memory/internal/algorithm"
	"github.com/example/graph-memory/internal/config"
	"github.com/example/graph-memory/internal/llm"
	"github.com/example/graph-memory/internal/repository"
	"github.com/example/graph-memory/internal/server"
	"github.com/example/graph-memory/internal/service"
	"github.com/example/graph-memory/internal/workflow"
	"github.com/example/graph-memory/pkg/logger"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// 初始化配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	log := logger.NewLoggerWithConfig(&logger.Config{
		Level:      cfg.Logging.Level,
		OutputPath: cfg.Logging.OutputPath,
		MaxSize:    cfg.Logging.MaxSize,
		MaxBackups: cfg.Logging.MaxBackups,
		MaxAge:     cfg.Logging.MaxAge,
		Compress:   cfg.Logging.Compress,
	})
	defer log.Close()

	log.Info("Starting Graph Memory Service",
		"version", Version,
		"build_time", BuildTime,
		"log_level", cfg.Logging.Level,
	)

	// 初始化Neo4j连接
	neo4jDriver, err := repository.NewNeo4jDriver(&cfg.Neo4j)
	if err != nil {
		log.Error("Failed to create Neo4j driver", "error", err)
		os.Exit(1)
	}
	defer neo4jDriver.Close()

	log.Info("Neo4j driver created", "uri", cfg.Neo4j.URI)

	// 初始化Repository
	nodeRepo := repository.NewNodeRepository(neo4jDriver)
	edgeRepo := repository.NewEdgeRepository(neo4jDriver)
	graphRepo := repository.NewGraphRepository(neo4jDriver)
	sessionRepo := repository.NewSessionRepository(neo4jDriver)
	messageRepo := repository.NewMessageRepository(neo4jDriver)
	vectorRepo := repository.NewVectorRepository(neo4jDriver)

	// 健康检查（非阻塞，允许服务启动后再连接）
	ctx := context.Background()
	if err := neo4jDriver.HealthCheck(ctx); err != nil {
		log.Warn("Neo4j health check failed, service will retry connection on demand",
			"error", err,
			"uri", cfg.Neo4j.URI,
		)
		// 不再强制退出，允许服务启动，后续操作时会自动重试连接
	} else {
		log.Info("Neo4j health check passed")
	}

	// 初始化LLM服务（使用eino框架）
	var llmService *llm.Service
	if cfg.LLM.Enabled && cfg.LLM.APIKey != "" {
		llmService, err = llm.NewService(&llm.ServiceConfig{
			EinoConfig: &llm.EinoConfig{
				APIKey:         cfg.LLM.APIKey,
				BaseURL:        cfg.LLM.BaseURL,
				Model:          cfg.LLM.ChatModel,
				EmbeddingModel: cfg.LLM.EmbeddingModel,
				Temperature:    cfg.LLM.Temperature,
				MaxTokens:      cfg.LLM.MaxTokens,
			},
		})
		if err != nil {
			log.Error("Failed to initialize LLM service", "error", err)
			os.Exit(1)
		}
		log.Info("LLM service initialized",
			"chat_model", cfg.LLM.ChatModel,
			"embedding_model", cfg.LLM.EmbeddingModel,
		)
	} else {
		log.Warn("LLM service disabled or API key not configured")
	}

	// 初始化算法服务
	algorithmConfig := &algorithm.AlgorithmConfig{
		PageRank: algorithm.PageRankConfig{
			DampingFactor:    cfg.Algorithms.PageRank.DampingFactor,
			MaxIterations:    cfg.Algorithms.PageRank.MaxIterations,
			ConvergenceDelta: cfg.Algorithms.PageRank.ConvergenceDelta,
		},
		CommunityDetection: algorithm.CommunityDetectionConfig{
			MaxIterations: cfg.Algorithms.CommunityDetection.MaxIterations,
		},
		Deduplication: algorithm.DeduplicationConfig{
			SimilarityThreshold: cfg.Algorithms.Deduplication.SimilarityThreshold,
		},
	}
	algorithmSvc := algorithm.NewAlgorithmService(graphRepo, algorithmConfig)

	// 初始化业务服务
	nodeService := service.NewNodeService(nodeRepo, edgeRepo, graphRepo)
	edgeService := service.NewEdgeService(edgeRepo, graphRepo)
	graphService := service.NewGraphService(graphRepo, nodeRepo, edgeRepo, algorithmSvc)
	sessionService := service.NewSessionService(sessionRepo, messageRepo)

	// 初始化召回服务
	recallService := service.NewRecallService(nodeRepo, edgeRepo, graphRepo, vectorRepo, llmService, nil)

	// 初始化工作流编排器
	orchestrator := workflow.NewOrchestrator(sessionRepo, messageRepo, nodeRepo, edgeRepo, graphRepo, algorithmSvc)

	// 初始化HTTP服务器
	httpAddr := fmt.Sprintf("%s:%d", cfg.Server.HTTP.Host, cfg.Server.HTTP.Port)
	httpServer := server.NewHTTPServer(httpAddr, nodeService, edgeService, graphService, sessionService, orchestrator, llmService, recallService, log)

	// 启动HTTP服务器
	go func() {
		log.Info("HTTP server listening", "addr", httpAddr)
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	log.Info("All services initialized successfully")

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down Graph Memory Service...")

	// 优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", "error", err)
	}

	select {
	case <-shutdownCtx.Done():
		log.Warn("Shutdown timeout exceeded")
	default:
	}

	log.Info("Graph Memory Service stopped")
}
