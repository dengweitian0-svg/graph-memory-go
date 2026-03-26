package algorithm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/repository"
	"github.com/example/graph-memory/pkg/logger"
)

// AlgorithmService 算法服务
type AlgorithmService struct {
	factory   *AlgorithmFactory
	graphRepo *repository.GraphRepository
	log       *logger.Logger
	config    *AlgorithmConfig
	mu        sync.RWMutex
	isRunning bool
	lastRun   time.Time
}

// NewAlgorithmService 创建算法服务
func NewAlgorithmService(graphRepo *repository.GraphRepository, config *AlgorithmConfig) *AlgorithmService {
	if config == nil {
		config = DefaultAlgorithmConfig()
	}
	return &AlgorithmService{
		factory:   NewAlgorithmFactory(graphRepo, config),
		graphRepo: graphRepo,
		log:       logger.NewLogger("info"),
		config:    config,
	}
}

// RunPageRank 运行PageRank算法
func (s *AlgorithmService) RunPageRank(ctx context.Context, nodeIDs []string) (*PageRankResult, error) {
	s.log.Info("Starting PageRank computation", "node_count", len(nodeIDs))

	startTime := time.Now()
	pr := s.factory.NewPageRanker()

	scores, err := pr.Compute(ctx, nodeIDs)
	if err != nil {
		return nil, fmt.Errorf("PageRank computation failed: %w", err)
	}

	result := &PageRankResult{
		Scores:    scores,
		NodeCount: len(scores),
		Duration:  time.Since(startTime),
	}

	s.log.Info("PageRank completed",
		"node_count", result.NodeCount,
		"duration", result.Duration,
	)

	return result, nil
}

// RunPersonalizedPageRank 运行个性化PageRank
func (s *AlgorithmService) RunPersonalizedPageRank(
	ctx context.Context,
	seedIDs []string,
	candidateIDs []string,
) (*PageRankResult, error) {
	s.log.Info("Starting Personalized PageRank",
		"seed_count", len(seedIDs),
		"candidate_count", len(candidateIDs),
	)

	startTime := time.Now()
	pr := s.factory.NewPageRanker()

	scores, err := pr.ComputePersonalized(ctx, seedIDs, candidateIDs)
	if err != nil {
		return nil, fmt.Errorf("personalized PageRank computation failed: %w", err)
	}

	result := &PageRankResult{
		Scores:    scores,
		NodeCount: len(scores),
		Duration:  time.Since(startTime),
	}

	s.log.Info("Personalized PageRank completed",
		"result_count", result.NodeCount,
		"duration", result.Duration,
	)

	return result, nil
}

// RunCommunityDetection 运行社区检测
func (s *AlgorithmService) RunCommunityDetection(ctx context.Context, nodeIDs []string) (*CommunityResult, error) {
	s.log.Info("Starting community detection", "node_count", len(nodeIDs))

	startTime := time.Now()
	cd := s.factory.NewCommunityDetector()

	assignments, err := cd.Detect(ctx, nodeIDs)
	if err != nil {
		return nil, fmt.Errorf("community detection failed: %w", err)
	}

	// 构建社区统计
	communities := s.buildCommunityStats(assignments)

	result := &CommunityResult{
		Assignments:    assignments,
		Communities:    communities,
		NodeCount:      len(assignments),
		CommunityCount: len(communities),
		Duration:       time.Since(startTime),
	}

	s.log.Info("Community detection completed",
		"node_count", result.NodeCount,
		"community_count", result.CommunityCount,
		"duration", result.Duration,
	)

	return result, nil
}

// RunDeduplication 运行去重检测
func (s *AlgorithmService) RunDeduplication(
	ctx context.Context,
	nodes []*model.Node,
	threshold float64,
) (*DeduplicationResult, error) {
	s.log.Info("Starting deduplication", "node_count", len(nodes), "threshold", threshold)

	startTime := time.Now()
	dd := s.factory.NewDeduplicator()

	groups, err := dd.DetectDuplicates(ctx, nodes, threshold)
	if err != nil {
		return nil, fmt.Errorf("deduplication failed: %w", err)
	}

	result := &DeduplicationResult{
		DuplicateGroups: groups,
		GroupCount:      len(groups),
		Duration:        time.Since(startTime),
	}

	s.log.Info("Deduplication completed",
		"group_count", result.GroupCount,
		"duration", result.Duration,
	)

	return result, nil
}

// RunFullPipeline 运行完整算法流水线
func (s *AlgorithmService) RunFullPipeline(ctx context.Context) (*PipelineResult, error) {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return nil, fmt.Errorf("algorithm pipeline is already running")
	}
	s.isRunning = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isRunning = false
		s.lastRun = time.Now().UTC()
		s.mu.Unlock()
	}()

	s.log.Info("Starting full algorithm pipeline")

	startTime := time.Now()
	result := &PipelineResult{}

	// 1. 加载所有节点
	nodes, err := s.graphRepo.GetAllNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load nodes: %w", err)
	}

	if len(nodes) == 0 {
		return &PipelineResult{
			PageRank:      &PageRankResult{},
			Community:     &CommunityResult{},
			Deduplication: &DeduplicationResult{},
			Duration:      time.Since(startTime),
		}, nil
	}

	nodeIDs := make([]string, len(nodes))
	for i, node := range nodes {
		nodeIDs[i] = node.ID
	}

	// 2. 运行PageRank
	prResult, err := s.RunPageRank(ctx, nodeIDs)
	if err != nil {
		s.log.Error("PageRank failed", "error", err)
		result.Errors = append(result.Errors, fmt.Sprintf("PageRank: %v", err))
	} else {
		result.PageRank = prResult
	}

	// 3. 运行社区检测
	cdResult, err := s.RunCommunityDetection(ctx, nodeIDs)
	if err != nil {
		s.log.Error("Community detection failed", "error", err)
		result.Errors = append(result.Errors, fmt.Sprintf("CommunityDetection: %v", err))
	} else {
		result.Community = cdResult
	}

	// 4. 运行去重检测
	ddResult, err := s.RunDeduplication(ctx, nodes, s.config.Deduplication.SimilarityThreshold)
	if err != nil {
		s.log.Error("Deduplication failed", "error", err)
		result.Errors = append(result.Errors, fmt.Sprintf("Deduplication: %v", err))
	} else {
		result.Deduplication = ddResult
	}

	result.Duration = time.Since(startTime)

	s.log.Info("Full pipeline completed",
		"total_duration", result.Duration,
		"errors", len(result.Errors),
	)

	return result, nil
}

// GetStatus 获取算法服务状态
func (s *AlgorithmService) GetStatus() *ServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &ServiceStatus{
		IsRunning: s.isRunning,
		LastRun:   s.lastRun,
		Config:    s.config,
	}
}

// buildCommunityStats 构建社区统计
func (s *AlgorithmService) buildCommunityStats(assignments map[string]string) map[string]*CommunityStats {
	communities := make(map[string]*CommunityStats)

	for nodeID, communityID := range assignments {
		if _, exists := communities[communityID]; !exists {
			communities[communityID] = &CommunityStats{
				ID:      communityID,
				NodeIDs: make([]string, 0),
			}
		}
		communities[communityID].NodeIDs = append(communities[communityID].NodeIDs, nodeID)
		communities[communityID].Size++
	}

	return communities
}

// PageRankResult PageRank结果
type PageRankResult struct {
	Scores    map[string]float64 `json:"scores"`
	NodeCount int                `json:"node_count"`
	Duration  time.Duration      `json:"duration"`
}

// CommunityResult 社区检测结果
type CommunityResult struct {
	Assignments    map[string]string          `json:"assignments"`
	Communities    map[string]*CommunityStats `json:"communities"`
	NodeCount      int                        `json:"node_count"`
	CommunityCount int                        `json:"community_count"`
	Duration       time.Duration              `json:"duration"`
}

// CommunityStats 社区统计
type CommunityStats struct {
	ID      string   `json:"id"`
	Size    int      `json:"size"`
	NodeIDs []string `json:"node_ids"`
}

// DeduplicationResult 去重结果
type DeduplicationResult struct {
	DuplicateGroups [][]string    `json:"duplicate_groups"`
	GroupCount      int           `json:"group_count"`
	Duration        time.Duration `json:"duration"`
}

// PipelineResult 流水线结果
type PipelineResult struct {
	PageRank      *PageRankResult      `json:"page_rank,omitempty"`
	Community     *CommunityResult     `json:"community,omitempty"`
	Deduplication *DeduplicationResult `json:"deduplication,omitempty"`
	Errors        []string             `json:"errors,omitempty"`
	Duration      time.Duration        `json:"duration"`
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	IsRunning bool             `json:"is_running"`
	LastRun   time.Time        `json:"last_run"`
	Config    *AlgorithmConfig `json:"config"`
}
