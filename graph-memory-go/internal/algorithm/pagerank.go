package algorithm

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/example/graph-memory/internal/repository"
	"github.com/example/graph-memory/pkg/logger"
)

// Neo4jPageRanker Neo4j实现的PageRank算法
type Neo4jPageRanker struct {
	graphRepo *repository.GraphRepository
	config    PageRankConfig
	log       *logger.Logger
}

// NewNeo4jPageRanker 创建Neo4j PageRank实例
func NewNeo4jPageRanker(graphRepo *repository.GraphRepository, config PageRankConfig) *Neo4jPageRanker {
	return &Neo4jPageRanker{
		graphRepo: graphRepo,
		config:    config,
		log:       logger.NewLogger("info"),
	}
}

// Name 返回算法名称
func (p *Neo4jPageRanker) Name() string {
	return "PageRank"
}

// Compute 计算全局PageRank
func (p *Neo4jPageRanker) Compute(ctx context.Context, nodeIDs []string) (map[string]float64, error) {
	if len(nodeIDs) == 0 {
		return make(map[string]float64), nil
	}

	p.log.Debug("Starting global PageRank computation", "node_count", len(nodeIDs))

	// 加载图结构
	graph, err := p.graphRepo.LoadGraphStructure(ctx, nodeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load graph structure: %w", err)
	}

	// 初始化分数
	n := len(graph.NodeSet)
	if n == 0 {
		return make(map[string]float64), nil
	}

	initialScore := 1.0 / float64(n)
	scores := make(map[string]float64, n)
	for nodeID := range graph.NodeSet {
		scores[nodeID] = initialScore
	}

	// 迭代计算
	dampingFactor := p.config.DampingFactor
	teleportProb := (1.0 - dampingFactor) / float64(n)

	for iter := 0; iter < p.config.MaxIterations; iter++ {
		newScores := p.iterate(scores, graph, dampingFactor, teleportProb)

		// 检查收敛
		delta := p.computeDelta(scores, newScores)
		scores = newScores

		p.log.Debug("PageRank iteration",
			"iteration", iter+1,
			"delta", delta,
		)

		if delta < p.config.ConvergenceDelta {
			p.log.Info("PageRank converged", "iterations", iter+1, "delta", delta)
			break
		}
	}

	// 归一化
	p.normalize(scores)

	p.log.Info("PageRank computation completed", "node_count", len(scores))
	return scores, nil
}

// ComputePersonalized 计算个性化PageRank
func (p *Neo4jPageRanker) ComputePersonalized(ctx context.Context, seedIDs, candidateIDs []string) (map[string]float64, error) {
	if len(seedIDs) == 0 {
		return nil, fmt.Errorf("seed nodes cannot be empty")
	}
	if len(candidateIDs) == 0 {
		return make(map[string]float64), nil
	}

	p.log.Debug("Starting personalized PageRank",
		"seed_count", len(seedIDs),
		"candidate_count", len(candidateIDs),
	)

	// 加载图结构
	graph, err := p.graphRepo.LoadGraphStructure(ctx, append(seedIDs, candidateIDs...))
	if err != nil {
		return nil, fmt.Errorf("failed to load graph structure: %w", err)
	}

	// 初始化分数 - 只在种子节点上分配初始分数
	seedProb := 1.0 / float64(len(seedIDs))
	scores := make(map[string]float64, len(graph.NodeSet))
	seedSet := make(map[string]bool, len(seedIDs))
	for _, seedID := range seedIDs {
		seedSet[seedID] = true
		scores[seedID] = seedProb
	}

	// 其他节点初始分数为0
	for nodeID := range graph.NodeSet {
		if !seedSet[nodeID] {
			scores[nodeID] = 0
		}
	}

	// 迭代计算
	dampingFactor := p.config.DampingFactor
	teleportProb := (1.0 - dampingFactor) / float64(len(seedIDs))

	for iter := 0; iter < p.config.MaxIterations; iter++ {
		newScores := p.iteratePersonalized(scores, graph, seedSet, dampingFactor, teleportProb)

		// 检查收敛
		delta := p.computeDelta(scores, newScores)
		scores = newScores

		if delta < p.config.ConvergenceDelta {
			p.log.Debug("Personalized PageRank converged",
				"iterations", iter+1,
				"delta", delta,
			)
			break
		}
	}

	// 只返回候选节点的分数
	result := make(map[string]float64, len(candidateIDs))
	candidateSet := make(map[string]bool, len(candidateIDs))
	for _, id := range candidateIDs {
		candidateSet[id] = true
	}

	for nodeID, score := range scores {
		if candidateSet[nodeID] || seedSet[nodeID] {
			result[nodeID] = score
		}
	}

	// 归一化
	p.normalize(result)

	p.log.Info("Personalized PageRank completed",
		"seed_count", len(seedIDs),
		"result_count", len(result),
	)

	return result, nil
}

// iterate 执行一次迭代（全局PageRank）
func (p *Neo4jPageRanker) iterate(scores map[string]float64, graph *repository.GraphStructure, dampingFactor, teleportProb float64) map[string]float64 {
	newScores := make(map[string]float64, len(graph.NodeSet))
	var mu sync.Mutex

	// 并行计算
	var wg sync.WaitGroup
	nodeList := make([]string, 0, len(graph.NodeSet))
	for nodeID := range graph.NodeSet {
		nodeList = append(nodeList, nodeID)
	}

	batchSize := 100
	for i := 0; i < len(nodeList); i += batchSize {
		end := i + batchSize
		if end > len(nodeList) {
			end = len(nodeList)
		}

		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()
			for _, nodeID := range batch {
				score := teleportProb

				// 计算来自入边的贡献
				for _, fromID := range p.getIncomingNodes(graph, nodeID) {
					outDegree := p.getOutDegree(graph, fromID)
					if outDegree > 0 {
						score += dampingFactor * scores[fromID] / float64(outDegree)
					}
				}

				mu.Lock()
				newScores[nodeID] = score
				mu.Unlock()
			}
		}(nodeList[i:end])
	}

	wg.Wait()
	return newScores
}

// iteratePersonalized 执行一次迭代（个性化PageRank）
func (p *Neo4jPageRanker) iteratePersonalized(
	scores map[string]float64,
	graph *repository.GraphStructure,
	seedSet map[string]bool,
	dampingFactor, teleportProb float64,
) map[string]float64 {
	newScores := make(map[string]float64, len(graph.NodeSet))

	// 计算来自入边的贡献
	for nodeID := range graph.NodeSet {
		score := 0.0

		// 如果是种子节点，添加teleport概率
		if seedSet[nodeID] {
			score = teleportProb
		}

		// 计算来自入边的贡献
		for _, fromID := range p.getIncomingNodes(graph, nodeID) {
			outDegree := p.getOutDegree(graph, fromID)
			if outDegree > 0 {
				score += dampingFactor * scores[fromID] / float64(outDegree)
			}
		}

		newScores[nodeID] = score
	}

	return newScores
}

// getIncomingNodes 获取入边节点列表
func (p *Neo4jPageRanker) getIncomingNodes(graph *repository.GraphStructure, nodeID string) []string {
	incoming := make([]string, 0)
	for fromID, neighbors := range graph.AdjList {
		for _, toID := range neighbors {
			if toID == nodeID {
				incoming = append(incoming, fromID)
			}
		}
	}
	return incoming
}

// getOutDegree 获取出度
func (p *Neo4jPageRanker) getOutDegree(graph *repository.GraphStructure, nodeID string) int {
	return len(graph.AdjList[nodeID])
}

// computeDelta 计算分数变化量
func (p *Neo4jPageRanker) computeDelta(oldScores, newScores map[string]float64) float64 {
	delta := 0.0
	for nodeID, newScore := range newScores {
		oldScore := oldScores[nodeID]
		delta += math.Abs(newScore - oldScore)
	}
	return delta
}

// normalize 归一化分数
func (p *Neo4jPageRanker) normalize(scores map[string]float64) {
	sum := 0.0
	for _, score := range scores {
		sum += score
	}
	if sum > 0 {
		for nodeID := range scores {
			scores[nodeID] /= sum
		}
	}
}
