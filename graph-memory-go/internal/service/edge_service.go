package service

import (
	"context"
	"fmt"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/repository"
	"github.com/example/graph-memory/pkg/logger"
)

// EdgeService 边服务
type EdgeService struct {
	edgeRepo  *repository.EdgeRepository
	graphRepo *repository.GraphRepository
	log       *logger.Logger
}

// NewEdgeService 创建边服务
func NewEdgeService(
	edgeRepo *repository.EdgeRepository,
	graphRepo *repository.GraphRepository,
) *EdgeService {
	return &EdgeService{
		edgeRepo:  edgeRepo,
		graphRepo: graphRepo,
		log:       logger.NewLogger("info"),
	}
}

// CreateEdge 创建边
func (s *EdgeService) CreateEdge(ctx context.Context, req *CreateEdgeRequest) (*model.Edge, error) {
	s.log.Debug("Creating edge", "from", req.FromID, "to", req.ToID, "type", req.Type)

	edge := &model.Edge{
		ID:           model.GenerateID("edge"),
		FromID:       req.FromID,
		ToID:         req.ToID,
		Type:         req.Type,
		Weight:       req.Weight,
		SourceSession: req.SourceSession,
		Metadata:     req.Metadata,
	}

	// 验证
	if err := edge.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 保存到数据库
	if err := s.edgeRepo.Create(ctx, edge); err != nil {
		return nil, fmt.Errorf("failed to create edge: %w", err)
	}

	s.log.Info("Edge created successfully", "id", edge.ID)
	return edge, nil
}

// GetEdge 获取边
func (s *EdgeService) GetEdge(ctx context.Context, id string) (*model.Edge, error) {
	edge, err := s.edgeRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get edge: %w", err)
	}
	return edge, nil
}

// UpdateEdge 更新边
func (s *EdgeService) UpdateEdge(ctx context.Context, id string, req *UpdateEdgeRequest) (*model.Edge, error) {
	s.log.Debug("Updating edge", "id", id)

	edge, err := s.edgeRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find edge: %w", err)
	}

	// 更新字段
	if req.Weight > 0 {
		edge.Weight = req.Weight
	}
	if req.SourceSession != "" {
		edge.SourceSession = req.SourceSession
	}
	if req.Metadata != nil {
		edge.Metadata = req.Metadata
	}

	// 验证
	if err := edge.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 保存更新
	if err := s.edgeRepo.Update(ctx, edge); err != nil {
		return nil, fmt.Errorf("failed to update edge: %w", err)
	}

	s.log.Info("Edge updated successfully", "id", id)
	return edge, nil
}

// DeleteEdge 删除边
func (s *EdgeService) DeleteEdge(ctx context.Context, id string) error {
	s.log.Debug("Deleting edge", "id", id)

	if err := s.edgeRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete edge: %w", err)
	}

	s.log.Info("Edge deleted successfully", "id", id)
	return nil
}

// ListEdges 列出边
func (s *EdgeService) ListEdges(ctx context.Context, req *ListEdgesRequest) (*ListEdgesResponse, error) {
	edges, total, err := s.edgeRepo.List(ctx, req.NodeID, req.Type, req.Page, req.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list edges: %w", err)
	}

	return &ListEdgesResponse{
		Edges:    edges,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// CreateEdgeRequest 创建边请求
type CreateEdgeRequest struct {
	FromID        string                 `json:"from_id"`
	ToID          string                 `json:"to_id"`
	Type          model.EdgeType         `json:"type"`
	Weight        float64                `json:"weight"`
	SourceSession string                 `json:"source_session,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateEdgeRequest 更新边请求
type UpdateEdgeRequest struct {
	Weight        float64                `json:"weight,omitempty"`
	SourceSession string                 `json:"source_session,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ListEdgesRequest 列出边请求
type ListEdgesRequest struct {
	NodeID   string         `json:"node_id,omitempty"`
	Type     model.EdgeType `json:"type,omitempty"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// ListEdgesResponse 列出边响应
type ListEdgesResponse struct {
	Edges    []*model.Edge `json:"edges"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}
