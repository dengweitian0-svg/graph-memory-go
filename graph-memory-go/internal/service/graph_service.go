package service

import (
	"context"
	"fmt"

	"github.com/example/graph-memory/internal/algorithm"
	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/repository"
	"github.com/example/graph-memory/pkg/logger"
)

// GraphService 图服务
type GraphService struct {
	graphRepo     *repository.GraphRepository
	nodeRepo      *repository.NodeRepository
	edgeRepo      *repository.EdgeRepository
	algorithmSvc  *algorithm.AlgorithmService
	log           *logger.Logger
}

// NewGraphService 创建图服务
func NewGraphService(
	graphRepo *repository.GraphRepository,
	nodeRepo *repository.NodeRepository,
	edgeRepo *repository.EdgeRepository,
	algorithmSvc *algorithm.AlgorithmService,
) *GraphService {
	return &GraphService{
		graphRepo:    graphRepo,
		nodeRepo:     nodeRepo,
		edgeRepo:     edgeRepo,
		algorithmSvc: algorithmSvc,
		log:          logger.NewLogger("info"),
	}
}

// GetSubgraph 获取子图
func (s *GraphService) GetSubgraph(ctx context.Context, req *GetSubgraphRequest) (*model.Subgraph, error) {
	s.log.Debug("Getting subgraph", "node_ids", req.NodeIDs, "depth", req.Depth)

	subgraph, err := s.graphRepo.GetSubgraph(ctx, req.NodeIDs, req.Depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get subgraph: %w", err)
	}

	return subgraph, nil
}

// GetNeighbors 获取邻居节点
func (s *GraphService) GetNeighbors(ctx context.Context, req *GetNeighborsRequest) (*GetNeighborsResponse, error) {
	s.log.Debug("Getting neighbors", "node_id", req.NodeID, "depth", req.Depth)

	nodes, err := s.graphRepo.GetNeighbors(ctx, req.NodeID, req.Depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get neighbors: %w", err)
	}

	return &GetNeighborsResponse{
		Nodes: nodes,
		Total: len(nodes),
	}, nil
}

// FindPath 查找路径
func (s *GraphService) FindPath(ctx context.Context, req *FindPathRequest) (*FindPathResponse, error) {
	s.log.Debug("Finding path", "from", req.FromID, "to", req.ToID)

	path, err := s.graphRepo.FindPath(ctx, req.FromID, req.ToID, req.MaxDepth)
	if err != nil {
		return nil, fmt.Errorf("failed to find path: %w", err)
	}

	return &FindPathResponse{
		Path: path,
	}, nil
}

// RunPageRank 运行PageRank算法
func (s *GraphService) RunPageRank(ctx context.Context, req *RunPageRankRequest) (*algorithm.PageRankResult, error) {
	s.log.Debug("Running PageRank", "node_count", len(req.NodeIDs))

	result, err := s.algorithmSvc.RunPageRank(ctx, req.NodeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to run PageRank: %w", err)
	}

	return result, nil
}

// RunCommunityDetection 运行社区检测
func (s *GraphService) RunCommunityDetection(ctx context.Context, req *RunCommunityDetectionRequest) (*algorithm.CommunityResult, error) {
	s.log.Debug("Running community detection", "node_count", len(req.NodeIDs))

	result, err := s.algorithmSvc.RunCommunityDetection(ctx, req.NodeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to run community detection: %w", err)
	}

	return result, nil
}

// RunDeduplication 运行去重检测
func (s *GraphService) RunDeduplication(ctx context.Context, req *RunDeduplicationRequest) (*algorithm.DeduplicationResult, error) {
	s.log.Debug("Running deduplication", "threshold", req.Threshold)

	// 加载所有节点
	nodes, err := s.graphRepo.GetAllNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load nodes: %w", err)
	}

	result, err := s.algorithmSvc.RunDeduplication(ctx, nodes, req.Threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to run deduplication: %w", err)
	}

	return result, nil
}

// RunFullPipeline 运行完整算法流水线
func (s *GraphService) RunFullPipeline(ctx context.Context) (*algorithm.PipelineResult, error) {
	s.log.Debug("Running full algorithm pipeline")

	result, err := s.algorithmSvc.RunFullPipeline(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run full pipeline: %w", err)
	}

	return result, nil
}

// GetSubgraphRequest 获取子图请求
type GetSubgraphRequest struct {
	NodeIDs []string `json:"node_ids"`
	Depth   int      `json:"depth"`
}

// GetNeighborsRequest 获取邻居请求
type GetNeighborsRequest struct {
	NodeID string `json:"node_id"`
	Depth  int    `json:"depth"`
}

// GetNeighborsResponse 获取邻居响应
type GetNeighborsResponse struct {
	Nodes []*model.Node `json:"nodes"`
	Total int           `json:"total"`
}

// FindPathRequest 查找路径请求
type FindPathRequest struct {
	FromID   string `json:"from_id"`
	ToID     string `json:"to_id"`
	MaxDepth int    `json:"max_depth"`
}

// FindPathResponse 查找路径响应
type FindPathResponse struct {
	Path *model.Path `json:"path"`
}

// RunPageRankRequest 运行PageRank请求
type RunPageRankRequest struct {
	NodeIDs []string `json:"node_ids,omitempty"`
}

// RunCommunityDetectionRequest 运行社区检测请求
type RunCommunityDetectionRequest struct {
	NodeIDs []string `json:"node_ids,omitempty"`
}

// RunDeduplicationRequest 运行去重请求
type RunDeduplicationRequest struct {
	Threshold float64 `json:"threshold"`
}
