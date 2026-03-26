package service

import (
	"context"
	"fmt"
	"time"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/repository"
	"github.com/example/graph-memory/pkg/logger"
)

// NodeService 节点服务
type NodeService struct {
	nodeRepo  *repository.NodeRepository
	edgeRepo  *repository.EdgeRepository
	graphRepo *repository.GraphRepository
	log       *logger.Logger
}

// NewNodeService 创建节点服务
func NewNodeService(
	nodeRepo *repository.NodeRepository,
	edgeRepo *repository.EdgeRepository,
	graphRepo *repository.GraphRepository,
) *NodeService {
	return &NodeService{
		nodeRepo:  nodeRepo,
		edgeRepo:  edgeRepo,
		graphRepo: graphRepo,
		log:       logger.NewLogger("info"),
	}
}

// CreateNode 创建节点
func (s *NodeService) CreateNode(ctx context.Context, req *CreateNodeRequest) (*model.Node, error) {
	s.log.Debug("Creating node", "name", req.Name, "type", req.Type)

	node := model.NewNode(req.Name, req.Type, req.Description)
	
	// 设置元数据
	if req.Metadata != nil {
		node.Metadata = req.Metadata
	}

	// 验证
	if err := node.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 保存到数据库
	if err := s.nodeRepo.Create(ctx, node); err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	s.log.Info("Node created successfully", "id", node.ID, "name", node.Name)
	return node, nil
}

// GetNode 获取节点
func (s *NodeService) GetNode(ctx context.Context, id string) (*model.Node, error) {
	node, err := s.nodeRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}
	return node, nil
}

// UpdateNode 更新节点
func (s *NodeService) UpdateNode(ctx context.Context, id string, req *UpdateNodeRequest) (*model.Node, error) {
	s.log.Debug("Updating node", "id", id)

	node, err := s.nodeRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find node: %w", err)
	}

	// 更新字段
	if req.Name != "" {
		node.Name = req.Name
	}
	if req.Description != "" {
		node.Description = req.Description
	}
	if req.Status != "" {
		node.Status = req.Status
	}
	if req.Metadata != nil {
		node.Metadata = req.Metadata
	}
	node.UpdatedAt = time.Now().UTC() // 使用 UTC 时区，避免 Neo4j 时区错误

	// 验证
	if err := node.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 保存更新
	if err := s.nodeRepo.Update(ctx, node); err != nil {
		return nil, fmt.Errorf("failed to update node: %w", err)
	}

	s.log.Info("Node updated successfully", "id", id)
	return node, nil
}

// DeleteNode 删除节点
func (s *NodeService) DeleteNode(ctx context.Context, id string) error {
	s.log.Debug("Deleting node", "id", id)

	if err := s.nodeRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	s.log.Info("Node deleted successfully", "id", id)
	return nil
}

// ListNodes 列出节点
func (s *NodeService) ListNodes(ctx context.Context, req *ListNodesRequest) (*ListNodesResponse, error) {
	nodes, total, err := s.nodeRepo.List(ctx, req.Type, req.Status, req.Page, req.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	return &ListNodesResponse{
		Nodes:    nodes,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// SearchNodes 搜索节点
func (s *NodeService) SearchNodes(ctx context.Context, req *SearchNodesRequest) (*SearchNodesResponse, error) {
	s.log.Debug("Searching nodes", "query", req.Query)

	nodes, err := s.nodeRepo.Search(ctx, req.Query, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search nodes: %w", err)
	}

	return &SearchNodesResponse{
		Nodes: nodes,
		Total: len(nodes),
	}, nil
}

// CreateNodeRequest 创建节点请求
type CreateNodeRequest struct {
	Name        string                 `json:"name"`
	Type        model.NodeType         `json:"type"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateNodeRequest 更新节点请求
type UpdateNodeRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Status      model.NodeStatus       `json:"status,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ListNodesRequest 列出节点请求
type ListNodesRequest struct {
	Type     model.NodeType   `json:"type,omitempty"`
	Status   model.NodeStatus `json:"status,omitempty"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// ListNodesResponse 列出节点响应
type ListNodesResponse struct {
	Nodes    []*model.Node `json:"nodes"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// SearchNodesRequest 搜索节点请求
type SearchNodesRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// SearchNodesResponse 搜索节点响应
type SearchNodesResponse struct {
	Nodes []*model.Node `json:"nodes"`
	Total int           `json:"total"`
}
