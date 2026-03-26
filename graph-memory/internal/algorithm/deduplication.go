package algorithm

import (
	"context"
	"math"
	"sync"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/repository"
	"github.com/example/graph-memory/pkg/logger"
)

// VectorDeduplicator 向量去重算法
type VectorDeduplicator struct {
	graphRepo *repository.GraphRepository
	config    DeduplicationConfig
	log       *logger.Logger
}

// NewVectorDeduplicator 创建向量去重实例
func NewVectorDeduplicator(graphRepo *repository.GraphRepository, config DeduplicationConfig) *VectorDeduplicator {
	return &VectorDeduplicator{
		graphRepo: graphRepo,
		config:    config,
		log:       logger.NewLogger("info"),
	}
}

// Name 返回算法名称
func (d *VectorDeduplicator) Name() string {
	return "VectorDeduplication"
}

// DuplicateGroup 重复节点组
type DuplicateGroup struct {
	PrimaryID    string   // 主要节点ID
	DuplicateIDs []string // 重复节点ID列表
	Similarity   float64  // 相似度
}

// DetectDuplicates 检测重复节点
func (d *VectorDeduplicator) DetectDuplicates(ctx context.Context, nodes []*model.Node, threshold float64) ([][]string, error) {
	if len(nodes) < 2 {
		return [][]string{}, nil
	}

	if threshold <= 0 {
		threshold = d.config.SimilarityThreshold
	}

	d.log.Debug("Starting duplicate detection",
		"node_count", len(nodes),
		"threshold", threshold,
	)

	// 过滤有向量的节点
	nodesWithEmbedding := make([]*model.Node, 0, len(nodes))
	for _, node := range nodes {
		if len(node.Embedding) > 0 {
			nodesWithEmbedding = append(nodesWithEmbedding, node)
		}
	}

	if len(nodesWithEmbedding) < 2 {
		return [][]string{}, nil
	}

	d.log.Info("Nodes with embeddings", "count", len(nodesWithEmbedding))

	// 并行计算相似度矩阵
	similarityMatrix := d.computeSimilarityMatrix(nodesWithEmbedding)

	// 构建重复组
	groups := d.buildDuplicateGroups(nodesWithEmbedding, similarityMatrix, threshold)

	d.log.Info("Duplicate detection completed",
		"total_nodes", len(nodesWithEmbedding),
		"duplicate_groups", len(groups),
	)

	return groups, nil
}

// MergeNodes 合并节点
func (d *VectorDeduplicator) MergeNodes(ctx context.Context, primaryID string, duplicateIDs []string) error {
	if len(duplicateIDs) == 0 {
		return nil
	}

	d.log.Info("Merging nodes",
		"primary_id", primaryID,
		"duplicate_count", len(duplicateIDs),
	)

	// 合并逻辑：
	// 1. 将重复节点的边迁移到主节点
	// 2. 合并 source_sessions
	// 3. 累加 validated_count
	// 4. 删除重复节点

	// TODO: 实现实际的数据库合并操作
	// 这里需要通过 graphRepo 执行 Cypher 事务

	return nil
}

// computeSimilarityMatrix 计算相似度矩阵
func (d *VectorDeduplicator) computeSimilarityMatrix(nodes []*model.Node) [][]float64 {
	n := len(nodes)
	matrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
		matrix[i][i] = 1.0 // 自己与自己的相似度为1
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// 并行计算上三角矩阵
	type pair struct {
		i, j int
		sim  float64
	}

	pairChan := make(chan pair, n*n/2)
	workerCount := 4

	// 启动worker
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range pairChan {
				sim := d.cosineSimilarity(nodes[p.i].Embedding, nodes[p.j].Embedding)
				mu.Lock()
				matrix[p.i][p.j] = sim
				matrix[p.j][p.i] = sim // 对称矩阵
				mu.Unlock()
			}
		}()
	}

	// 发送计算任务
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			pairChan <- pair{i: i, j: j}
		}
	}
	close(pairChan)

	wg.Wait()
	return matrix
}

// cosineSimilarity 计算余弦相似度
func (d *VectorDeduplicator) cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := 0; i < len(a); i++ {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// buildDuplicateGroups 构建重复组
func (d *VectorDeduplicator) buildDuplicateGroups(nodes []*model.Node, similarityMatrix [][]float64, threshold float64) [][]string {
	n := len(nodes)
	visited := make([]bool, n)
	groups := make([][]string, 0)

	for i := 0; i < n; i++ {
		if visited[i] {
			continue
		}

		group := []string{nodes[i].ID}
		visited[i] = true

		// 找到所有与节点i相似的节点
		for j := i + 1; j < n; j++ {
			if visited[j] {
				continue
			}

			if similarityMatrix[i][j] >= threshold {
				group = append(group, nodes[j].ID)
				visited[j] = true
			}
		}

		// 只有当组内有多于1个节点时才添加
		if len(group) > 1 {
			groups = append(groups, group)
		}
	}

	return groups
}

// FindSimilarNodes 查找与目标节点相似的节点
func (d *VectorDeduplicator) FindSimilarNodes(target *model.Node, candidates []*model.Node, threshold float64) []*model.Node {
	if len(target.Embedding) == 0 || len(candidates) == 0 {
		return []*model.Node{}
	}

	if threshold <= 0 {
		threshold = d.config.SimilarityThreshold
	}

	similar := make([]*model.Node, 0)
	for _, candidate := range candidates {
		if candidate.ID == target.ID {
			continue
		}
		if len(candidate.Embedding) == 0 {
			continue
		}

		sim := d.cosineSimilarity(target.Embedding, candidate.Embedding)
		if sim >= threshold {
			similar = append(similar, candidate)
		}
	}

	return similar
}

// BatchSimilarity 批量计算相似度
func (d *VectorDeduplicator) BatchSimilarity(target *model.Node, candidates []*model.Node) map[string]float64 {
	result := make(map[string]float64)

	if len(target.Embedding) == 0 {
		return result
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, candidate := range candidates {
		if candidate.ID == target.ID || len(candidate.Embedding) == 0 {
			continue
		}

		wg.Add(1)
		go func(c *model.Node) {
			defer wg.Done()
			sim := d.cosineSimilarity(target.Embedding, c.Embedding)
			mu.Lock()
			result[c.ID] = sim
			mu.Unlock()
		}(candidate)
	}

	wg.Wait()
	return result
}
