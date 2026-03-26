package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/example/graph-memory/internal/algorithm"
	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/repository"
	"github.com/example/graph-memory/pkg/logger"
)

// Orchestrator 工作流编排器
type Orchestrator struct {
	sessionRepo  *repository.SessionRepository
	messageRepo  *repository.MessageRepository
	nodeRepo     *repository.NodeRepository
	edgeRepo     *repository.EdgeRepository
	graphRepo    *repository.GraphRepository
	algorithmSvc *algorithm.AlgorithmService

	extractor *KnowledgeExtractor
	builder   *GraphBuilder
	log       *logger.Logger

	mu       sync.Mutex
	running  map[string]*Workflow
}

// NewOrchestrator 创建工作流编排器
func NewOrchestrator(
	sessionRepo *repository.SessionRepository,
	messageRepo *repository.MessageRepository,
	nodeRepo *repository.NodeRepository,
	edgeRepo *repository.EdgeRepository,
	graphRepo *repository.GraphRepository,
	algorithmSvc *algorithm.AlgorithmService,
) *Orchestrator {
	return &Orchestrator{
		sessionRepo:  sessionRepo,
		messageRepo:  messageRepo,
		nodeRepo:     nodeRepo,
		edgeRepo:     edgeRepo,
		graphRepo:    graphRepo,
		algorithmSvc: algorithmSvc,
		extractor:    NewKnowledgeExtractor(),
		builder:      NewGraphBuilder(nodeRepo, edgeRepo, graphRepo),
		log:          logger.NewLogger("info"),
		running:      make(map[string]*Workflow),
	}
}

// ExecuteWorkflow 执行工作流
func (o *Orchestrator) ExecuteWorkflow(ctx context.Context, config *WorkflowConfig) (*WorkflowResult, error) {
	o.mu.Lock()
	
	// 创建工作流实例
	workflow := &Workflow{
		ID:        model.GenerateID("workflow"),
		Type:      config.Type,
		Status:    WorkflowStatusPending,
		CreatedAt: time.Now().UTC(), // 使用 UTC 时区，避免 Neo4j 时区错误
		Steps:     make([]*WorkflowStep, 0),
		Metadata:  make(map[string]interface{}),
	}

	// 检查是否已有相同类型的工作流在运行
	for _, w := range o.running {
		if w.Type == config.Type && w.Status == WorkflowStatusRunning {
			o.mu.Unlock()
			return nil, fmt.Errorf("workflow of type %s is already running", config.Type)
		}
	}

	o.running[workflow.ID] = workflow
	o.mu.Unlock()

	// 执行工作流
	result, err := o.execute(ctx, workflow, config)

	// 清理
	o.mu.Lock()
	delete(o.running, workflow.ID)
	o.mu.Unlock()

	return result, err
}

// execute 执行工作流
func (o *Orchestrator) execute(ctx context.Context, workflow *Workflow, config *WorkflowConfig) (*WorkflowResult, error) {
	startTime := time.Now().UTC()
	workflow.Status = WorkflowStatusRunning
	now := startTime
	workflow.StartedAt = &now

	o.log.Info("Starting workflow",
		"id", workflow.ID,
		"type", workflow.Type,
	)

	result := &WorkflowResult{
		Workflow:   workflow,
		Statistics: &Statistics{},
	}

	defer func() {
		endTime := time.Now().UTC()
		workflow.EndedAt = &endTime
		workflow.Duration = endTime.Sub(startTime)
		result.Statistics.Duration = workflow.Duration
	}()

	// 根据工作流类型执行不同的步骤
	switch config.Type {
	case WorkflowTypeIngestion:
		return o.executeIngestion(ctx, workflow, config, result)
	case WorkflowTypeExtraction:
		return o.executeExtraction(ctx, workflow, config, result)
	case WorkflowTypeUpdate:
		return o.executeUpdate(ctx, workflow, config, result)
	case WorkflowTypeMaintenance:
		return o.executeMaintenance(ctx, workflow, config, result)
	case WorkflowTypeFull:
		return o.executeFull(ctx, workflow, config, result)
	default:
		workflow.Status = WorkflowStatusFailed
		workflow.Error = fmt.Sprintf("unknown workflow type: %s", config.Type)
		return result, fmt.Errorf(workflow.Error)
	}
}

// executeIngestion 执行数据摄入工作流
func (o *Orchestrator) executeIngestion(ctx context.Context, workflow *Workflow, config *WorkflowConfig, result *WorkflowResult) (*WorkflowResult, error) {
	// 1. 加载会话消息
	messages, err := o.loadMessages(ctx, config)
	if err != nil {
		workflow.Status = WorkflowStatusFailed
		workflow.Error = err.Error()
		return result, err
	}

	result.Statistics.MessagesProcessed = len(messages)

	// 2. 提取知识
	if config.EnableExtraction {
		extraction, err := o.runExtraction(ctx, workflow, messages)
		if err != nil {
			workflow.Status = WorkflowStatusFailed
			workflow.Error = err.Error()
			return result, err
		}
		result.Extraction = extraction
	}

	// 3. 构建图
	if config.EnableGraphBuilding && result.Extraction != nil {
		buildResult, err := o.runGraphBuilding(ctx, workflow, result.Extraction, config.SessionID)
		if err != nil {
			workflow.Status = WorkflowStatusFailed
			workflow.Error = err.Error()
			return result, err
		}
		result.GraphBuilding = buildResult
		result.Statistics.NodesCreated = len(buildResult.Nodes)
		result.Statistics.EdgesCreated = len(buildResult.Edges)
	}

	workflow.Status = WorkflowStatusCompleted
	return result, nil
}

// executeExtraction 执行知识提取工作流
func (o *Orchestrator) executeExtraction(ctx context.Context, workflow *Workflow, config *WorkflowConfig, result *WorkflowResult) (*WorkflowResult, error) {
	messages, err := o.loadMessages(ctx, config)
	if err != nil {
		workflow.Status = WorkflowStatusFailed
		workflow.Error = err.Error()
		return result, err
	}

	extraction, err := o.runExtraction(ctx, workflow, messages)
	if err != nil {
		workflow.Status = WorkflowStatusFailed
		workflow.Error = err.Error()
		return result, err
	}

	result.Extraction = extraction
	workflow.Status = WorkflowStatusCompleted
	return result, nil
}

// executeUpdate 执行图更新工作流
func (o *Orchestrator) executeUpdate(ctx context.Context, workflow *Workflow, config *WorkflowConfig, result *WorkflowResult) (*WorkflowResult, error) {
	if !config.EnableAlgorithms {
		workflow.Status = WorkflowStatusCompleted
		return result, nil
	}

	// 运行算法更新
	algorithmResult, err := o.algorithmSvc.RunFullPipeline(ctx)
	if err != nil {
		workflow.Status = WorkflowStatusFailed
		workflow.Error = err.Error()
		return result, err
	}

	result.Algorithm = algorithmResult
	workflow.Status = WorkflowStatusCompleted
	return result, nil
}

// executeMaintenance 执行维护工作流
func (o *Orchestrator) executeMaintenance(ctx context.Context, workflow *Workflow, config *WorkflowConfig, result *WorkflowResult) (*WorkflowResult, error) {
	// 1. 运行去重
	if config.EnableDeduplication {
		nodes, err := o.graphRepo.GetAllNodes(ctx)
		if err != nil {
			workflow.Status = WorkflowStatusFailed
			workflow.Error = err.Error()
			return result, err
		}

		ddResult, err := o.algorithmSvc.RunDeduplication(ctx, nodes, config.AlgorithmThreshold)
		if err != nil {
			workflow.Status = WorkflowStatusFailed
			workflow.Error = err.Error()
			return result, err
		}

		result.Algorithm = ddResult
	}

	// 2. 更新算法
	if config.EnableAlgorithms {
		_, err := o.algorithmSvc.RunFullPipeline(ctx)
		if err != nil {
			workflow.Status = WorkflowStatusFailed
			workflow.Error = err.Error()
			return result, err
		}
	}

	workflow.Status = WorkflowStatusCompleted
	return result, nil
}

// executeFull 执行完整工作流
func (o *Orchestrator) executeFull(ctx context.Context, workflow *Workflow, config *WorkflowConfig, result *WorkflowResult) (*WorkflowResult, error) {
	// 完整工作流 = 摄入 + 更新 + 维护
	ingestionResult, err := o.executeIngestion(ctx, workflow, config, result)
	if err != nil {
		return ingestionResult, err
	}

	updateResult, err := o.executeUpdate(ctx, workflow, config, ingestionResult)
	if err != nil {
		return updateResult, err
	}

	maintenanceResult, err := o.executeMaintenance(ctx, workflow, config, updateResult)
	return maintenanceResult, err
}

// loadMessages 加载消息
func (o *Orchestrator) loadMessages(ctx context.Context, config *WorkflowConfig) ([]*model.Message, error) {
	if len(config.MessageIDs) > 0 {
		// 加载指定消息
		messages := make([]*model.Message, 0, len(config.MessageIDs))
		for _, id := range config.MessageIDs {
			msg, err := o.messageRepo.FindByID(ctx, id)
			if err != nil {
				o.log.Warn("Failed to load message", "id", id, "error", err)
				continue
			}
			messages = append(messages, msg)
		}
		return messages, nil
	}

	if config.SessionID != "" {
		// 加载会话消息
		return o.messageRepo.FindBySessionID(ctx, config.SessionID)
	}

	// 加载未处理的消息
	return o.messageRepo.FindUnextracted(ctx, config.BatchSize)
}

// runExtraction 运行知识提取
func (o *Orchestrator) runExtraction(ctx context.Context, workflow *Workflow, messages []*model.Message) (*ExtractionResult, error) {
	step := &WorkflowStep{
		Name:   "knowledge_extraction",
		Status: WorkflowStatusRunning,
	}
	now := time.Now().UTC()
	step.StartedAt = &now
	workflow.Steps = append(workflow.Steps, step)

	result, err := o.extractor.ExtractFromMessages(ctx, messages)
	if err != nil {
		step.Status = WorkflowStatusFailed
		step.Error = err.Error()
		return nil, err
	}

	endTime := time.Now().UTC()
	step.EndedAt = &endTime
	step.Duration = endTime.Sub(*step.StartedAt)
	step.Status = WorkflowStatusCompleted
	step.Output = result

	return result, nil
}

// runGraphBuilding 运行图构建
func (o *Orchestrator) runGraphBuilding(ctx context.Context, workflow *Workflow, extraction *ExtractionResult, sessionID string) (*BuildResult, error) {
	step := &WorkflowStep{
		Name:   "graph_building",
		Status: WorkflowStatusRunning,
	}
	now := time.Now().UTC()
	step.StartedAt = &now
	workflow.Steps = append(workflow.Steps, step)

	result, err := o.builder.BuildFromExtraction(ctx, extraction, sessionID)
	if err != nil {
		step.Status = WorkflowStatusFailed
		step.Error = err.Error()
		return nil, err
	}

	endTime := time.Now().UTC()
	step.EndedAt = &endTime
	step.Duration = endTime.Sub(*step.StartedAt)
	step.Status = WorkflowStatusCompleted
	step.Output = result

	return result, nil
}

// GetWorkflowStatus 获取工作流状态
func (o *Orchestrator) GetWorkflowStatus(workflowID string) (*Workflow, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	workflow, exists := o.running[workflowID]
	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	return workflow, nil
}

// CancelWorkflow 取消工作流
func (o *Orchestrator) CancelWorkflow(workflowID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	workflow, exists := o.running[workflowID]
	if !exists {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	workflow.Status = WorkflowStatusCancelled
	return nil
}
