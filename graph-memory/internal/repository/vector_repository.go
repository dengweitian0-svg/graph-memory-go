package repository

import (
	"context"
	"fmt"
	"sort"

	"github.com/example/graph-memory/internal/model"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// VectorRepository 向量仓库 - 提供向量相似度搜索功能
type VectorRepository struct {
	driver *Neo4jDriver
}

// NewVectorRepository 创建向量仓库
func NewVectorRepository(driver *Neo4jDriver) *VectorRepository {
	return &VectorRepository{driver: driver}
}

// VectorSearch 向量相似度搜索
// 使用余弦相似度进行搜索
func (r *VectorRepository) VectorSearch(ctx context.Context, req *model.VectorSearchRequest) ([]*model.VectorSearchResult, error) {
	if len(req.Vector) == 0 {
		return []*model.VectorSearchResult{}, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	minScore := req.MinScore
	if minScore <= 0 {
		minScore = 0.5
	}

	// 从Neo4j获取所有有向量的节点
	nodes, err := r.getNodesWithEmbeddings(ctx, req.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes with embeddings: %w", err)
	}

	// 计算相似度并排序
	results := make([]*model.VectorSearchResult, 0)
	queryVector := make([]float32, len(req.Vector))
	for i, v := range req.Vector {
		queryVector[i] = float32(v)
	}

	for _, node := range nodes {
		if len(node.Embedding) == 0 {
			continue
		}

		similarity := model.CosineSimilarity(queryVector, node.Embedding)
		if similarity >= minScore {
			results = append(results, &model.VectorSearchResult{
				NodeID:   node.ID,
				Score:    similarity,
				Distance: 1 - similarity, // 余弦距离
			})
		}
	}

	// 按分数降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// 限制返回数量
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// VectorSearchWithNodes 向量搜索并返回节点详情
func (r *VectorRepository) VectorSearchWithNodes(ctx context.Context, req *model.VectorSearchRequest) ([]*model.Node, error) {
	results, err := r.VectorSearch(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return []*model.Node{}, nil
	}

	// 获取节点ID列表
	nodeIDs := make([]string, len(results))
	for i, r := range results {
		nodeIDs[i] = r.NodeID
	}

	// 批量获取节点
	nodeRepo := NewNodeRepository(r.driver)
	nodes, err := nodeRepo.FindByIDs(ctx, nodeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	// 按相似度分数排序
	nodeMap := make(map[string]*model.Node)
	for _, node := range nodes {
		nodeMap[node.ID] = node
	}

	sortedNodes := make([]*model.Node, 0, len(results))
	for _, result := range results {
		if node, ok := nodeMap[result.NodeID]; ok {
			// 将相似度分数存储在节点的PageRank字段中（临时使用）
			node.PageRank = result.Score
			sortedNodes = append(sortedNodes, node)
		}
	}

	return sortedNodes, nil
}

// FindSimilarNodes 查找与指定节点相似的节点
func (r *VectorRepository) FindSimilarNodes(ctx context.Context, nodeID string, limit int, minScore float64) ([]*model.VectorSearchResult, error) {
	// 获取节点
	nodeRepo := NewNodeRepository(r.driver)
	node, err := nodeRepo.FindByID(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to find node: %w", err)
	}

	if len(node.Embedding) == 0 {
		return []*model.VectorSearchResult{}, nil
	}

	// 转换为float64
	embedding := make([]float64, len(node.Embedding))
	for i, v := range node.Embedding {
		embedding[i] = float64(v)
	}

	// 执行向量搜索
	return r.VectorSearch(ctx, &model.VectorSearchRequest{
		Vector:   embedding,
		Limit:    limit + 1, // +1 因为会包含自身
		MinScore: minScore,
	})
}

// UpdateNodeEmbedding 更新节点的向量嵌入
func (r *VectorRepository) UpdateNodeEmbedding(ctx context.Context, nodeID string, embedding []float64) error {
	_, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		embeddingHash := model.ComputeEmbeddingHash(toFloat32Slice(embedding))

		_, err := tx.Run(ctx, `
			MATCH (n:Node {id: $id})
			SET n.embedding = $embedding,
			    n.embedding_hash = $embedding_hash
		`, map[string]interface{}{
			"id":             nodeID,
			"embedding":      embedding,
			"embedding_hash": embeddingHash,
		})

		return nil, err
	})

	return err
}

// BatchUpdateEmbeddings 批量更新节点嵌入
func (r *VectorRepository) BatchUpdateEmbeddings(ctx context.Context, embeddings map[string][]float64) error {
	for nodeID, embedding := range embeddings {
		if err := r.UpdateNodeEmbedding(ctx, nodeID, embedding); err != nil {
			return fmt.Errorf("failed to update embedding for node %s: %w", nodeID, err)
		}
	}
	return nil
}

// getNodesWithEmbeddings 获取有嵌入向量的节点
func (r *VectorRepository) getNodesWithEmbeddings(ctx context.Context, filters *model.RecallFilters) ([]*model.Node, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// 构建查询条件
		whereClause := "WHERE n.embedding IS NOT NULL AND size(n.embedding) > 0"
		params := make(map[string]interface{})

		if filters != nil {
			if len(filters.NodeTypes) > 0 {
				types := make([]string, len(filters.NodeTypes))
				for i, t := range filters.NodeTypes {
					types[i] = string(t)
				}
				whereClause += " AND n.type IN $types"
				params["types"] = types
			}
			if filters.Status != "" {
				whereClause += " AND n.status = $status"
				params["status"] = string(filters.Status)
			}
			if filters.CommunityID != "" {
				whereClause += " AND n.community_id = $community_id"
				params["community_id"] = filters.CommunityID
			}
			if filters.MinPageRank > 0 {
				whereClause += " AND n.pagerank >= $min_pagerank"
				params["min_pagerank"] = filters.MinPageRank
			}
		}

		query := fmt.Sprintf(`
			MATCH (n:Node)
			%s
			RETURN n
		`, whereClause)

		record, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		nodes := make([]*model.Node, 0)
		for record.Next(ctx) {
			if nodeValue, ok := record.Record().Get("n"); ok {
				if node, ok := nodeValue.(neo4j.Node); ok {
					parsed, err := r.parseNodeFromProps(node.Props)
					if err != nil {
						continue
					}
					nodes = append(nodes, parsed)
				}
			}
		}

		return nodes, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*model.Node), nil
}

// parseNodeFromProps 从属性解析节点
func (r *VectorRepository) parseNodeFromProps(props map[string]interface{}) (*model.Node, error) {
	node := &model.Node{
		ID:          getString(props, "id"),
		Name:        getString(props, "name"),
		Type:        model.NodeType(getString(props, "type")),
		Description: getString(props, "description"),
		Status:      model.NodeStatus(getString(props, "status")),
	}

	if v, ok := props["validated_count"].(int64); ok {
		node.ValidatedCount = int(v)
	}
	if v, ok := props["pagerank"].(float64); ok {
		node.PageRank = v
	}
	node.CommunityID = getString(props, "community_id")
	node.EmbeddingHash = getString(props, "embedding_hash")
	if v, ok := props["embedding"].([]interface{}); ok {
		node.Embedding = make([]float32, 0, len(v))
		for _, f := range v {
			if val, ok := f.(float64); ok {
				node.Embedding = append(node.Embedding, float32(val))
			}
		}
	}
	if v, ok := props["metadata"].(map[string]interface{}); ok {
		node.Metadata = v
	}

	return node, nil
}

// toFloat32Slice 将float64切片转换为float32切片
func toFloat32Slice(src []float64) []float32 {
	dst := make([]float32, len(src))
	for i, v := range src {
		dst[i] = float32(v)
	}
	return dst
}

// GetNodesByCommunity 根据社区ID获取节点
func (r *VectorRepository) GetNodesByCommunity(ctx context.Context, communityID string) ([]*model.Node, error) {
	nodeRepo := NewNodeRepository(r.driver)
	return nodeRepo.FindByCommunityID(ctx, communityID)
}

// GetHighPageRankNodes 获取高PageRank节点
func (r *VectorRepository) GetHighPageRankNodes(ctx context.Context, minPageRank float64, limit int) ([]*model.Node, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (n:Node)
			WHERE n.pagerank >= $min_pagerank
			RETURN n
			ORDER BY n.pagerank DESC
			LIMIT $limit
		`, map[string]interface{}{
			"min_pagerank": minPageRank,
			"limit":        limit,
		})

		if err != nil {
			return nil, err
		}

		nodes := make([]*model.Node, 0)
		for record.Next(ctx) {
			if nodeValue, ok := record.Record().Get("n"); ok {
				if node, ok := nodeValue.(neo4j.Node); ok {
					parsed, _ := r.parseNodeFromProps(node.Props)
					nodes = append(nodes, parsed)
				}
			}
		}

		return nodes, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*model.Node), nil
}
