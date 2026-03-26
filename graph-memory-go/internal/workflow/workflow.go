package workflow

import (
	"context"
	"time"
)

// WorkflowType 工作流类型
type WorkflowType string

const (
	WorkflowTypeIngestion   WorkflowType = "ingestion"   // 数据摄入工作流
	WorkflowTypeExtraction  WorkflowType = "extraction"  // 知识提取工作流
	WorkflowTypeUpdate      WorkflowType = "update"      // 图更新工作流
	WorkflowTypeMaintenance WorkflowType = "maintenance" // 维护工作流
	WorkflowTypeFull        WorkflowType = "full"        // 完整工作流
)

// WorkflowStatus 工作流状态
type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)

// Workflow 工作流定义
type Workflow struct {
	ID        string         `json:"id"`
	Type      WorkflowType   `json:"type"`
	Status    WorkflowStatus `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	StartedAt *time.Time     `json:"started_at,omitempty"`
	EndedAt   *time.Time     `json:"ended_at,omitempty"`
	Duration  time.Duration  `json:"duration,omitempty"`
	Error     string         `json:"error,omitempty"`
	Steps     []*WorkflowStep `json:"steps,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// WorkflowStep 工作流步骤
type WorkflowStep struct {
	Name      string         `json:"name"`
	Status    WorkflowStatus `json:"status"`
	StartedAt *time.Time     `json:"started_at,omitempty"`
	EndedAt   *time.Time     `json:"ended_at,omitempty"`
	Duration  time.Duration  `json:"duration,omitempty"`
	Error     string         `json:"error,omitempty"`
	Output    interface{}    `json:"output,omitempty"`
}

// WorkflowConfig 工作流配置
type WorkflowConfig struct {
	Type                    WorkflowType
	SessionID               string
	MessageIDs              []string
	EnableExtraction        bool
	EnableGraphBuilding     bool
	EnableAlgorithms        bool
	EnableDeduplication     bool
	AlgorithmThreshold      float64
	BatchSize               int
}

// DefaultWorkflowConfig 默认工作流配置
func DefaultWorkflowConfig() *WorkflowConfig {
	return &WorkflowConfig{
		Type:                WorkflowTypeFull,
		EnableExtraction:    true,
		EnableGraphBuilding: true,
		EnableAlgorithms:    true,
		EnableDeduplication: false,
		AlgorithmThreshold:  0.95,
		BatchSize:           100,
	}
}

// WorkflowExecutor 工作流执行器接口
type WorkflowExecutor interface {
	// Execute 执行工作流
	Execute(ctx context.Context, config *WorkflowConfig) (*WorkflowResult, error)
	// GetType 获取工作流类型
	GetType() WorkflowType
	// GetName 获取工作流名称
	GetName() string
}

// WorkflowResult 工作流结果
type WorkflowResult struct {
	Workflow      *Workflow        `json:"workflow"`
	Extraction    *ExtractionResult `json:"extraction,omitempty"`
	GraphBuilding *BuildResult      `json:"graph_building,omitempty"`
	Algorithm     interface{}       `json:"algorithm,omitempty"`
	Statistics    *Statistics       `json:"statistics,omitempty"`
}

// Statistics 统计信息
type Statistics struct {
	NodesCreated     int            `json:"nodes_created"`
	NodesUpdated     int            `json:"nodes_updated"`
	EdgesCreated     int            `json:"edges_created"`
	EdgesUpdated     int            `json:"edges_updated"`
	MessagesProcessed int           `json:"messages_processed"`
	Duration         time.Duration  `json:"duration"`
	Errors           []string       `json:"errors,omitempty"`
}
