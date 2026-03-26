package workflow

import (
	"context"
	"sync"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/repository"
	"github.com/example/graph-memory/pkg/logger"
)

// GraphBuilder 图构建器
type GraphBuilder struct {
	nodeRepo  *repository.NodeRepository
	edgeRepo  *repository.EdgeRepository
	graphRepo *repository.GraphRepository
	log       *logger.Logger
	mu        sync.Mutex
}

// NewGraphBuilder 创建图构建器
func NewGraphBuilder(
	nodeRepo *repository.NodeRepository,
	edgeRepo *repository.EdgeRepository,
	graphRepo *repository.GraphRepository,
) *GraphBuilder {
	return &GraphBuilder{
		nodeRepo:  nodeRepo,
		edgeRepo:  edgeRepo,
		graphRepo: graphRepo,
		log:       logger.NewLogger("info"),
	}
}

// BuildFromExtraction 从提取结果构建图
func (b *GraphBuilder) BuildFromExtraction(ctx context.Context, extraction *ExtractionResult, sessionID string) (*BuildResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.log.Debug("Building graph from extraction",
		"nodes", len(extraction.Nodes),
		"edges", len(extraction.Edges),
		"session_id", sessionID,
	)

	result := &BuildResult{
		Nodes: make([]*model.Node, 0),
		Edges: make([]*model.Edge, 0),
	}

	// 创建节点映射（名称 -> 节点ID）
	nodeMap := make(map[string]string)

	// 1. 创建或更新节点
	for _, extractedNode := range extraction.Nodes {
		node, err := b.createOrUpdateNode(ctx, extractedNode, sessionID)
		if err != nil {
			b.log.Error("Failed to create/update node", "name", extractedNode.Name, "error", err)
			continue
		}
		result.Nodes = append(result.Nodes, node)
		nodeMap[extractedNode.Name] = node.ID
	}

	// 2. 创建或更新边
	for _, extractedEdge := range extraction.Edges {
		fromID, fromOK := nodeMap[extractedEdge.FromName]
		toID, toOK := nodeMap[extractedEdge.ToName]

		// 如果节点不存在，尝试查找已有节点
		if !fromOK {
			fromID = b.findNodeByName(ctx, extractedEdge.FromName)
			if fromID != "" {
				nodeMap[extractedEdge.FromName] = fromID
			}
		}
		if !toOK {
			toID = b.findNodeByName(ctx, extractedEdge.ToName)
			if toID != "" {
				nodeMap[extractedEdge.ToName] = toID
			}
		}

		// 如果节点都存在，创建边
		if fromID != "" && toID != "" {
			edge, err := b.createOrUpdateEdge(ctx, extractedEdge, fromID, toID, sessionID)
			if err != nil {
				b.log.Error("Failed to create/update edge",
					"from", extractedEdge.FromName,
					"to", extractedEdge.ToName,
					"error", err,
				)
				continue
			}
			result.Edges = append(result.Edges, edge)
		}
	}

	b.log.Info("Graph building completed",
		"nodes_created", len(result.Nodes),
		"edges_created", len(result.Edges),
	)

	return result, nil
}

// createOrUpdateNode 创建或更新节点
func (b *GraphBuilder) createOrUpdateNode(ctx context.Context, extracted *ExtractedNode, sessionID string) (*model.Node, error) {
	// 查找现有节点
	existingNodes, err := b.nodeRepo.SearchByName(ctx, extracted.Name, 1)
	if err == nil && len(existingNodes) > 0 {
		// 更新现有节点
		node := existingNodes[0]
		node.AddSourceSession(sessionID)
		node.IncrementValidatedCount()
		
		if err := b.nodeRepo.Update(ctx, node); err != nil {
			return nil, err
		}
		
		return node, nil
	}

	// 创建新节点
	node := model.NewNode(extracted.Name, extracted.Type, extracted.Description)
	node.AddSourceSession(sessionID)
	node.Metadata = extracted.Metadata

	if err := b.nodeRepo.Create(ctx, node); err != nil {
		return nil, err
	}

	return node, nil
}

// createOrUpdateEdge 创建或更新边
func (b *GraphBuilder) createOrUpdateEdge(ctx context.Context, extracted *ExtractedEdge, fromID, toID string, sessionID string) (*model.Edge, error) {
	// 查找现有边
	existingEdges, err := b.edgeRepo.FindBetweenNodes(ctx, fromID, toID)
	if err == nil && len(existingEdges) > 0 {
		// 更新现有边（找到同类型的边）
		for _, edge := range existingEdges {
			if edge.Type == extracted.Type {
				edge.Weight += 0.1 // 增加权重
				edge.SourceSession = sessionID
				if err := b.edgeRepo.Update(ctx, edge); err != nil {
					return nil, err
				}
				return edge, nil
			}
		}
	}

	// 创建新边
	edge := model.NewEdge(extracted.Type, fromID, toID)
	edge.Weight = extracted.Weight
	edge.SourceSession = sessionID
	edge.Metadata = extracted.Metadata

	if err := b.edgeRepo.Create(ctx, edge); err != nil {
		return nil, err
	}

	return edge, nil
}

// findNodeByName 根据名称查找节点
func (b *GraphBuilder) findNodeByName(ctx context.Context, name string) string {
	nodes, err := b.nodeRepo.SearchByName(ctx, name, 1)
	if err != nil || len(nodes) == 0 {
		return ""
	}
	return nodes[0].ID
}

// BuildResult 构建结果
type BuildResult struct {
	Nodes []*model.Node `json:"nodes"`
	Edges []*model.Edge `json:"edges"`
}

// MergeNodes 合并节点
func (b *GraphBuilder) MergeNodes(ctx context.Context, primaryID string, duplicateIDs []string) error {
	b.log.Info("Merging nodes", "primary", primaryID, "duplicates", duplicateIDs)

	// 实现节点合并逻辑
	// 1. 将重复节点的边迁移到主节点
	// 2. 合并 source_sessions
	// 3. 累加 validated_count
	// 4. 删除重复节点

	// TODO: 实现完整的合并逻辑

	return nil
}
