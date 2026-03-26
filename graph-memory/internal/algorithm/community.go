package algorithm

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/repository"
	"github.com/example/graph-memory/pkg/logger"
)

// LabelPropagation 标签传播社区检测算法
type LabelPropagation struct {
	graphRepo *repository.GraphRepository
	config    CommunityDetectionConfig
	log       *logger.Logger
}

// NewLabelPropagation 创建标签传播实例
func NewLabelPropagation(graphRepo *repository.GraphRepository, config CommunityDetectionConfig) *LabelPropagation {
	return &LabelPropagation{
		graphRepo: graphRepo,
		config:    config,
		log:       logger.NewLogger("info"),
	}
}

// Name 返回算法名称
func (l *LabelPropagation) Name() string {
	return "LabelPropagation"
}

// Detect 执行社区检测
func (l *LabelPropagation) Detect(ctx context.Context, nodeIDs []string) (map[string]string, error) {
	if len(nodeIDs) == 0 {
		return make(map[string]string), nil
	}

	l.log.Debug("Starting community detection", "node_count", len(nodeIDs))

	// 加载图结构
	graph, err := l.graphRepo.LoadGraphStructure(ctx, nodeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load graph structure: %w", err)
	}

	// 初始化：每个节点的标签是其自身
	labels := make(map[string]string, len(graph.NodeSet))
	for nodeID := range graph.NodeSet {
		labels[nodeID] = nodeID
	}

	// 迭代传播标签
	for iter := 0; iter < l.config.MaxIterations; iter++ {
		changed := l.propagate(labels, graph)

		l.log.Debug("Label propagation iteration",
			"iteration", iter+1,
			"changed", changed,
		)

		if changed == 0 {
			l.log.Info("Label propagation converged", "iterations", iter+1)
			break
		}
	}

	// 合并小社区并规范化标签
	labels = l.normalizeLabels(labels)

	l.log.Info("Community detection completed",
		"node_count", len(labels),
		"community_count", l.countCommunities(labels),
	)

	return labels, nil
}

// propagate 执行一次标签传播
func (l *LabelPropagation) propagate(labels map[string]string, graph *repository.GraphStructure) int {
	changed := 0
	var mu sync.Mutex

	// 获取所有节点ID并随机打乱顺序
	nodeList := make([]string, 0, len(graph.NodeSet))
	for nodeID := range graph.NodeSet {
		nodeList = append(nodeList, nodeID)
	}

	// 简单随机：交替顺序
	if len(nodeList) > 1 {
		for i := 0; i < len(nodeList)/2; i++ {
			j := len(nodeList) - 1 - i
			nodeList[i], nodeList[j] = nodeList[j], nodeList[i]
		}
	}

	// 并行处理
	var wg sync.WaitGroup
	batchSize := 50

	for i := 0; i < len(nodeList); i += batchSize {
		end := i + batchSize
		if end > len(nodeList) {
			end = len(nodeList)
		}

		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()
			localChanged := 0

			for _, nodeID := range batch {
				// 获取邻居节点的标签频率
				labelFreq := make(map[string]int)
				neighbors := l.getNeighbors(graph, nodeID)

				for _, neighborID := range neighbors {
					neighborLabel := labels[neighborID]
					labelFreq[neighborLabel]++
				}

				// 选择频率最高的标签
				if len(labelFreq) > 0 {
					newLabel := l.selectBestLabel(labelFreq, labels[nodeID])
					if newLabel != labels[nodeID] {
						labels[nodeID] = newLabel
						localChanged++
					}
				}
			}

			mu.Lock()
			changed += localChanged
			mu.Unlock()
		}(nodeList[i:end])
	}

	wg.Wait()
	return changed
}

// getNeighbors 获取所有邻居节点（入边+出边）
func (l *LabelPropagation) getNeighbors(graph *repository.GraphStructure, nodeID string) []string {
	neighborSet := make(map[string]bool)

	// 出边邻居
	for _, toID := range graph.AdjList[nodeID] {
		neighborSet[toID] = true
	}

	// 入边邻居
	for fromID, neighbors := range graph.AdjList {
		for _, toID := range neighbors {
			if toID == nodeID {
				neighborSet[fromID] = true
			}
		}
	}

	neighbors := make([]string, 0, len(neighborSet))
	for id := range neighborSet {
		neighbors = append(neighbors, id)
	}
	return neighbors
}

// selectBestLabel 选择最佳标签
func (l *LabelPropagation) selectBestLabel(freq map[string]int, currentLabel string) string {
	// 找到最大频率
	maxFreq := 0
	for _, f := range freq {
		if f > maxFreq {
			maxFreq = f
		}
	}

	// 收集所有具有最大频率的标签
	candidates := make([]string, 0)
	for label, f := range freq {
		if f == maxFreq {
			candidates = append(candidates, label)
		}
	}

	// 如果当前标签在候选中，优先保持
	for _, label := range candidates {
		if label == currentLabel {
			return currentLabel
		}
	}

	// 否则选择字典序最小的标签（确保确定性）
	sort.Strings(candidates)
	return candidates[0]
}

// normalizeLabels 规范化标签
func (l *LabelPropagation) normalizeLabels(labels map[string]string) map[string]string {
	// 构建社区映射
	communityMembers := make(map[string][]string)
	for nodeID, label := range labels {
		communityMembers[label] = append(communityMembers[label], nodeID)
	}

	// 为每个社区分配规范化的标签
	result := make(map[string]string, len(labels))
	for _, members := range communityMembers {
		if len(members) > 0 {
			// 使用社区中字典序最小的节点ID作为标签
			sort.Strings(members)
			communityLabel := model.GenerateID("community")
			for _, nodeID := range members {
				result[nodeID] = communityLabel
			}
		}
	}

	return result
}

// countCommunities 统计社区数量
func (l *LabelPropagation) countCommunities(labels map[string]string) int {
	communities := make(map[string]bool)
	for _, label := range labels {
		communities[label] = true
	}
	return len(communities)
}
