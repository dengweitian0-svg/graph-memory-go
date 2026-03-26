package repository

import (
	"context"
	"fmt"

	"github.com/example/graph-memory/internal/model"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// GraphRepository 图查询仓库
type GraphRepository struct {
	driver *Neo4jDriver
}

// NewGraphRepository 创建图查询仓库
func NewGraphRepository(driver *Neo4jDriver) *GraphRepository {
	return &GraphRepository{driver: driver}
}

// GraphStructure 图结构
type GraphStructure struct {
	NodeSet map[string]bool
	AdjList map[string][]string
}

// GetSubgraph 获取子图
func (r *GraphRepository) GetSubgraph(ctx context.Context, nodeIDs []string, depth int) (*model.Subgraph, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH path = (n:Node)-[r*1..`+fmt.Sprintf("%d", depth)+`]-(m:Node)
			WHERE n.id IN $node_ids
			RETURN n, r, m, path
			LIMIT 1000
		`, map[string]interface{}{
			"node_ids": nodeIDs,
		})

		if err != nil {
			return nil, err
		}

		subgraph := &model.Subgraph{
			Nodes: make([]*model.Node, 0),
			Edges: make([]*model.Edge, 0),
		}

		nodeSet := make(map[string]bool)
		edgeSet := make(map[string]bool)

		for record.Next(ctx) {
			// 解析节点
			if nodeValue, ok := record.Record().Get("n"); ok {
				if node, ok := nodeValue.(neo4j.Node); ok {
					if !nodeSet[node.Props["id"].(string)] {
						nodeSet[node.Props["id"].(string)] = true
						n, _ := r.parseNodeFromProps(node.Props)
						subgraph.Nodes = append(subgraph.Nodes, n)
					}
				}
			}

			if mValue, ok := record.Record().Get("m"); ok {
				if mNode, ok := mValue.(neo4j.Node); ok {
					if !nodeSet[mNode.Props["id"].(string)] {
						nodeSet[mNode.Props["id"].(string)] = true
						n, _ := r.parseNodeFromProps(mNode.Props)
						subgraph.Nodes = append(subgraph.Nodes, n)
					}
				}
			}

			// 解析边
			if rValue, ok := record.Record().Get("r"); ok {
				if rels, ok := rValue.([]interface{}); ok {
					for _, rel := range rels {
						if relationship, ok := rel.(neo4j.Relationship); ok {
							edgeID := relationship.Props["id"].(string)
							if !edgeSet[edgeID] {
								edgeSet[edgeID] = true
								e := &model.Edge{
									ID:     edgeID,
									Type:   model.EdgeType(relationship.Type),
									Weight: relationship.Props["weight"].(float64),
								}
								subgraph.Edges = append(subgraph.Edges, e)
							}
						}
					}
				}
			}
		}

		return subgraph, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*model.Subgraph), nil
}

// GetNeighbors 获取邻居节点
func (r *GraphRepository) GetNeighbors(ctx context.Context, nodeID string, depth int) ([]*model.Node, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (n:Node {id: $node_id})-[*1..`+fmt.Sprintf("%d", depth)+`]-(neighbor:Node)
			RETURN DISTINCT neighbor
			LIMIT 100
		`, map[string]interface{}{
			"node_id": nodeID,
		})

		if err != nil {
			return nil, err
		}

		nodes := make([]*model.Node, 0)
		for record.Next(ctx) {
			if neighborValue, ok := record.Record().Get("neighbor"); ok {
				if neighbor, ok := neighborValue.(neo4j.Node); ok {
					n, _ := r.parseNodeFromProps(neighbor.Props)
					nodes = append(nodes, n)
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

// LoadGraphStructure 加载图结构（用于算法）
func (r *GraphRepository) LoadGraphStructure(ctx context.Context, nodeIDs []string) (*GraphStructure, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// 获取节点
		nodeRecord, err := tx.Run(ctx, `
			MATCH (n:Node)
			WHERE n.id IN $node_ids OR n.community_id IN (
				SELECT DISTINCT community_id FROM Node WHERE id IN $node_ids AND community_id <> ''
			)
			RETURN n.id AS id
		`, map[string]interface{}{
			"node_ids": nodeIDs,
		})

		if err != nil {
			return nil, err
		}

		structure := &GraphStructure{
			NodeSet: make(map[string]bool),
			AdjList: make(map[string][]string),
		}

		// 收集节点
		for nodeRecord.Next(ctx) {
			id, _ := nodeRecord.Record().Get("id")
			structure.NodeSet[id.(string)] = true
		}

		// 获取边
		edgeRecord, err := tx.Run(ctx, `
			MATCH (from:Node)-[r]->(to:Node)
			WHERE from.id IN $node_ids OR from.community_id IN (
				SELECT DISTINCT community_id FROM Node WHERE id IN $node_ids AND community_id <> ''
			)
			RETURN from.id AS from_id, to.id AS to_id
		`, map[string]interface{}{
			"node_ids": nodeIDs,
		})

		if err != nil {
			return nil, err
		}

		// 构建邻接表
		for edgeRecord.Next(ctx) {
			fromID, _ := edgeRecord.Record().Get("from_id")
			toID, _ := edgeRecord.Record().Get("to_id")

			from := fromID.(string)
			to := toID.(string)

			if !structure.NodeSet[from] {
				structure.NodeSet[from] = true
			}
			if !structure.NodeSet[to] {
				structure.NodeSet[to] = true
			}

			structure.AdjList[from] = append(structure.AdjList[from], to)
		}

		return structure, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*GraphStructure), nil
}

// FindShortestPath 查找最短路径
func (r *GraphRepository) FindShortestPath(ctx context.Context, fromID, toID string, maxDepth int) ([]*model.Node, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH path = shortestPath(
				(from:Node {id: $from_id})-[*1..`+fmt.Sprintf("%d", maxDepth)+`]-(to:Node {id: $to_id})
			)
			RETURN nodes(path) AS nodes
		`, map[string]interface{}{
			"from_id": fromID,
			"to_id":   toID,
		})

		if err != nil {
			return nil, err
		}

		if !record.Next(ctx) {
			return []*model.Node{}, nil
		}

		nodesValue, _ := record.Record().Get("nodes")
		nodes := make([]*model.Node, 0)

		if nodesSlice, ok := nodesValue.([]interface{}); ok {
			for _, n := range nodesSlice {
				if node, ok := n.(neo4j.Node); ok {
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

// FindPath 查找路径（返回完整的路径对象）
func (r *GraphRepository) FindPath(ctx context.Context, fromID, toID string, maxDepth int) (*model.Path, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH path = shortestPath(
				(from:Node {id: $from_id})-[*1..`+fmt.Sprintf("%d", maxDepth)+`]-(to:Node {id: $to_id})
			)
			RETURN nodes(path) AS nodes, relationships(path) AS rels
		`, map[string]interface{}{
			"from_id": fromID,
			"to_id":   toID,
		})

		if err != nil {
			return nil, err
		}

		if !record.Next(ctx) {
			return nil, nil
		}

		path := &model.Path{
			Nodes: make([]*model.Node, 0),
			Edges: make([]*model.Edge, 0),
		}

		// 解析节点
		if nodesValue, ok := record.Record().Get("nodes"); ok {
			if nodesSlice, ok := nodesValue.([]interface{}); ok {
				for _, n := range nodesSlice {
					if node, ok := n.(neo4j.Node); ok {
						parsed, _ := r.parseNodeFromProps(node.Props)
						path.Nodes = append(path.Nodes, parsed)
					}
				}
			}
		}

		// 解析边
		if relsValue, ok := record.Record().Get("rels"); ok {
			if relsSlice, ok := relsValue.([]interface{}); ok {
				for _, rel := range relsSlice {
					if relationship, ok := rel.(neo4j.Relationship); ok {
						edge := &model.Edge{
							ID:     getString(relationship.Props, "id"),
							Type:   model.EdgeType(relationship.Type),
							Weight: getFloat64(relationship.Props, "weight"),
						}
						path.Edges = append(path.Edges, edge)
					}
				}
			}
		}

		path.Length = len(path.Edges)
		return path, nil
	})

	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	return result.(*model.Path), nil
}

// getFloat64 从map中获取float64
func getFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0.0
}

// GetAllNodes 获取所有节点（用于PageRank等算法）
func (r *GraphRepository) GetAllNodes(ctx context.Context) ([]*model.Node, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (n:Node)
			RETURN n
			ORDER BY n.created_at DESC
		`, nil)

		if err != nil {
			return nil, err
		}

		nodes := make([]*model.Node, 0)
		for record.Next(ctx) {
			if nodeValue, ok := record.Record().Get("n"); ok {
				if node, ok := nodeValue.(neo4j.Node); ok {
					n, _ := r.parseNodeFromProps(node.Props)
					nodes = append(nodes, n)
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

// UpdatePageRankScores 批量更新PageRank分数
func (r *GraphRepository) UpdatePageRankScores(ctx context.Context, scores map[string]float64) error {
	_, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for nodeID, score := range scores {
			_, err := tx.Run(ctx, `
				MATCH (n:Node {id: $id})
				SET n.pagerank = $score
			`, map[string]interface{}{
				"id":    nodeID,
				"score": score,
			})
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

	return err
}

// UpdateCommunityIDs 批量更新社区ID
func (r *GraphRepository) UpdateCommunityIDs(ctx context.Context, assignments map[string]string) error {
	_, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for nodeID, communityID := range assignments {
			_, err := tx.Run(ctx, `
				MATCH (n:Node {id: $id})
				SET n.community_id = $community_id
			`, map[string]interface{}{
				"id":          nodeID,
				"community_id": communityID,
			})
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

	return err
}

// parseNodeFromProps 从属性解析节点
func (r *GraphRepository) parseNodeFromProps(props map[string]interface{}) (*model.Node, error) {
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
