package algorithm

import (
	"context"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/repository"
)

// Algorithm 图算法接口
type Algorithm interface {
	// Name 算法名称
	Name() string
}

// PageRanker PageRank算法接口
type PageRanker interface {
	Algorithm
	// Compute 计算全局PageRank
	Compute(ctx context.Context, nodeIDs []string) (map[string]float64, error)
	// ComputePersonalized 计算个性化PageRank
	ComputePersonalized(ctx context.Context, seedIDs, candidateIDs []string) (map[string]float64, error)
}

// CommunityDetector 社区检测接口
type CommunityDetector interface {
	Algorithm
	// Detect 检测社区
	Detect(ctx context.Context, nodeIDs []string) (map[string]string, error)
}

// Deduplicator 去重接口
type Deduplicator interface {
	Algorithm
	// DetectDuplicates 检测重复节点
	DetectDuplicates(ctx context.Context, nodes []*model.Node, threshold float64) ([][]string, error)
	// MergeNodes 合并节点
	MergeNodes(ctx context.Context, primaryID string, duplicateIDs []string) error
}

// AlgorithmConfig 算法配置
type AlgorithmConfig struct {
	PageRank           PageRankConfig
	CommunityDetection CommunityDetectionConfig
	Deduplication      DeduplicationConfig
}

// PageRankConfig PageRank配置
type PageRankConfig struct {
	DampingFactor    float64
	MaxIterations    int
	ConvergenceDelta float64
}

// CommunityDetectionConfig 社区检测配置
type CommunityDetectionConfig struct {
	MaxIterations int
}

// DeduplicationConfig 去重配置
type DeduplicationConfig struct {
	SimilarityThreshold float64
}

// DefaultAlgorithmConfig 默认算法配置
func DefaultAlgorithmConfig() *AlgorithmConfig {
	return &AlgorithmConfig{
		PageRank: PageRankConfig{
			DampingFactor:    0.85,
			MaxIterations:    20,
			ConvergenceDelta: 1e-6,
		},
		CommunityDetection: CommunityDetectionConfig{
			MaxIterations: 10,
		},
		Deduplication: DeduplicationConfig{
			SimilarityThreshold: 0.95,
		},
	}
}

// AlgorithmFactory 算法工厂
type AlgorithmFactory struct {
	graphRepo *repository.GraphRepository
	config    *AlgorithmConfig
}

// NewAlgorithmFactory 创建算法工厂
func NewAlgorithmFactory(graphRepo *repository.GraphRepository, config *AlgorithmConfig) *AlgorithmFactory {
	if config == nil {
		config = DefaultAlgorithmConfig()
	}
	return &AlgorithmFactory{
		graphRepo: graphRepo,
		config:    config,
	}
}

// NewPageRanker 创建PageRank算法实例
func (f *AlgorithmFactory) NewPageRanker() PageRanker {
	return NewNeo4jPageRanker(f.graphRepo, f.config.PageRank)
}

// NewCommunityDetector 创建社区检测算法实例
func (f *AlgorithmFactory) NewCommunityDetector() CommunityDetector {
	return NewLabelPropagation(f.graphRepo, f.config.CommunityDetection)
}

// NewDeduplicator 创建去重算法实例
func (f *AlgorithmFactory) NewDeduplicator() Deduplicator {
	return NewVectorDeduplicator(f.graphRepo, f.config.Deduplication)
}
