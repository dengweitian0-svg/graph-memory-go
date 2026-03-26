package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/example/graph-memory/internal/model"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// neo4jRelTypeRegex 验证 Neo4j 关系类型名称的正则表达式
// 关系类型必须以字母开头，只能包含字母、数字和下划线
var neo4jRelTypeRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// validateRelType 验证并返回有效的关系类型名称
func validateRelType(relType string) (string, error) {
	relType = strings.TrimSpace(relType)
	if relType == "" {
		return "", fmt.Errorf("relationship type cannot be empty")
	}
	if !neo4jRelTypeRegex.MatchString(relType) {
		return "", fmt.Errorf("invalid relationship type '%s': must start with a letter and contain only letters, numbers, and underscores", relType)
	}
	return relType, nil
}

// EdgeRepository 边仓库
type EdgeRepository struct {
	driver *Neo4jDriver
}

// NewEdgeRepository 创建边仓库
func NewEdgeRepository(driver *Neo4jDriver) *EdgeRepository {
	return &EdgeRepository{driver: driver}
}

// Create 创建边
func (r *EdgeRepository) Create(ctx context.Context, edge *model.Edge) error {
	if err := edge.Validate(); err != nil {
		return err
	}

	// 验证并获取安全的关系类型名称
	relType, err := validateRelType(string(edge.Type))
	if err != nil {
		return fmt.Errorf("invalid edge type: %w", err)
	}

	// 将 metadata 序列化为 JSON 字符串，避免 Neo4j Map 类型错误
	var metadataJSON string
	if edge.Metadata != nil && len(edge.Metadata) > 0 {
		bytes, err := json.Marshal(edge.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(bytes)
	}

	_, err = r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// 注意：Neo4j 不支持参数化关系类型，必须使用字符串拼接
		// 但我们已经通过 validateRelType 确保了类型名称的安全性
		query := fmt.Sprintf(`
			MATCH (from:Node {id: $from_id})
			MATCH (to:Node {id: $to_id})
			CREATE (from)-[r:%s {
				id: $id,
				weight: $weight,
				created_at: $created_at,
				updated_at: $updated_at,
				source_session: $source_session,
				metadata: $metadata
			}]->(to)
			RETURN r.id AS id
		`, relType)

		_, err := tx.Run(ctx, query, map[string]interface{}{
			"id":             edge.ID,
			"from_id":        edge.FromID,
			"to_id":          edge.ToID,
			"weight":         edge.Weight,
			"created_at":     edge.CreatedAt,
			"updated_at":     edge.UpdatedAt,
			"source_session": edge.SourceSession,
			"metadata":       metadataJSON,
		})

		return nil, err
	})

	return err
}

// FindByID 根据ID查找边
func (r *EdgeRepository) FindByID(ctx context.Context, id string) (*model.Edge, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH ()-[r]->()
			WHERE r.id = $id
			RETURN r, startNode(r) AS from_node, endNode(r) AS to_node
		`, map[string]interface{}{"id": id})

		if err != nil {
			return nil, err
		}

		if !record.Next(ctx) {
			return nil, model.ErrEdgeNotFound
		}

		edge, err := r.parseEdgeWithNodes(record.Record())
		if err != nil {
			return nil, err
		}

		return edge, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*model.Edge), nil
}

// FindByFromNode 根据起始节点查找边
func (r *EdgeRepository) FindByFromNode(ctx context.Context, nodeID string, limit int) ([]*model.EdgeQueryResult, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (from:Node {id: $from_id})-[r]->(to:Node)
			RETURN r, from, to
			ORDER BY r.weight DESC
			LIMIT $limit
		`, map[string]interface{}{
			"from_id": nodeID,
			"limit":   limit,
		})

		if err != nil {
			return nil, err
		}

		edges := make([]*model.EdgeQueryResult, 0)
		for record.Next(ctx) {
			edge, err := r.parseEdgeWithNodes(record.Record())
			if err != nil {
				return nil, err
			}
			edges = append(edges, edge)
		}

		return edges, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*model.EdgeQueryResult), nil
}

// FindByToNode 根据目标节点查找边
func (r *EdgeRepository) FindByToNode(ctx context.Context, nodeID string, limit int) ([]*model.EdgeQueryResult, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (from:Node)-[r]->(to:Node {id: $to_id})
			RETURN r, from, to
			ORDER BY r.weight DESC
			LIMIT $limit
		`, map[string]interface{}{
			"to_id": nodeID,
			"limit": limit,
		})

		if err != nil {
			return nil, err
		}

		edges := make([]*model.EdgeQueryResult, 0)
		for record.Next(ctx) {
			edge, err := r.parseEdgeWithNodes(record.Record())
			if err != nil {
				return nil, err
			}
			edges = append(edges, edge)
		}

		return edges, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*model.EdgeQueryResult), nil
}

// FindBetweenNodes 查找两个节点之间的边
func (r *EdgeRepository) FindBetweenNodes(ctx context.Context, fromID, toID string) ([]*model.Edge, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (from:Node {id: $from_id})-[r]->(to:Node {id: $to_id})
			RETURN r
		`, map[string]interface{}{
			"from_id": fromID,
			"to_id":   toID,
		})

		if err != nil {
			return nil, err
		}

		edges := make([]*model.Edge, 0)
		for record.Next(ctx) {
			edge, err := r.parseEdge(record.Record())
			if err != nil {
				return nil, err
			}
			edges = append(edges, edge)
		}

		return edges, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*model.Edge), nil
}

// Update 更新边
func (r *EdgeRepository) Update(ctx context.Context, edge *model.Edge) error {
	// 将 metadata 序列化为 JSON 字符串，避免 Neo4j Map 类型错误
	var metadataJSON string
	if edge.Metadata != nil && len(edge.Metadata) > 0 {
		bytes, err := json.Marshal(edge.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(bytes)
	}

	_, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, `
			MATCH ()-[r]->()
			WHERE r.id = $id
			SET r.weight = $weight,
			    r.updated_at = $updated_at,
			    r.source_session = $source_session,
			    r.metadata = $metadata
		`, map[string]interface{}{
			"id":             edge.ID,
			"weight":         edge.Weight,
			"updated_at":     edge.UpdatedAt,
			"source_session": edge.SourceSession,
			"metadata":       metadataJSON,
		})

		return nil, err
	})

	return err
}

// Delete 删除边
func (r *EdgeRepository) Delete(ctx context.Context, id string) error {
	_, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, `
			MATCH ()-[r]->()
			WHERE r.id = $id
			DELETE r
		`, map[string]interface{}{"id": id})

		return nil, err
	})

	return err
}

// DeleteBetweenNodes 删除两个节点之间的所有边
func (r *EdgeRepository) DeleteBetweenNodes(ctx context.Context, fromID, toID string) error {
	_, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, `
			MATCH (from:Node {id: $from_id})-[r]->(to:Node {id: $to_id})
			DELETE r
		`, map[string]interface{}{
			"from_id": fromID,
			"to_id":   toID,
		})

		return nil, err
	})

	return err
}

// Count 统计边数量
func (r *EdgeRepository) Count(ctx context.Context) (int64, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH ()-[r]->()
			RETURN count(r) AS count
		`, nil)

		if err != nil {
			return nil, err
		}

		if !record.Next(ctx) {
			return int64(0), nil
		}

		count, _ := record.Record().Get("count")
		return count.(int64), nil
	})

	if err != nil {
		return 0, err
	}

	return result.(int64), nil
}

// List 列出边
func (r *EdgeRepository) List(ctx context.Context, nodeID string, edgeType model.EdgeType, page, pageSize int) ([]*model.Edge, int, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// 构建查询条件
		var whereClause string
		var matchClause string
		params := map[string]interface{}{
			"skip":  (page - 1) * pageSize,
			"limit": pageSize,
		}

		if nodeID != "" {
			matchClause = "MATCH (n:Node {id: $node_id})-[r]->(to:Node)"
			params["node_id"] = nodeID
		} else {
			matchClause = "MATCH ()-[r]->()"
		}

		if edgeType != "" {
			whereClause = "WHERE type(r) = $edge_type"
			params["edge_type"] = string(edgeType)
		}

		// 查询总数
		countQuery := matchClause + " " + whereClause + " RETURN count(r) AS count"
		countRecord, err := tx.Run(ctx, countQuery, params)
		if err != nil {
			return nil, err
		}

		var total int
		if countRecord.Next(ctx) {
			count, _ := countRecord.Record().Get("count")
			total = int(count.(int64))
		}

		// 查询边
		query := matchClause + " " + whereClause + `
			RETURN r
			ORDER BY r.created_at DESC
			SKIP $skip
			LIMIT $limit`

		record, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		edges := make([]*model.Edge, 0)
		for record.Next(ctx) {
			edge, err := r.parseEdge(record.Record())
			if err != nil {
				return nil, err
			}
			edges = append(edges, edge)
		}

		return map[string]interface{}{
			"edges": edges,
			"total": total,
		}, nil
	})

	if err != nil {
		return nil, 0, err
	}

	data := result.(map[string]interface{})
	return data["edges"].([]*model.Edge), data["total"].(int), nil
}

// parseEdge 解析边
func (r *EdgeRepository) parseEdge(record *neo4j.Record) (*model.Edge, error) {
	relValue, ok := record.Get("r")
	if !ok {
		return nil, fmt.Errorf("edge not found in record")
	}

	rel, ok := relValue.(neo4j.Relationship)
	if !ok {
		return nil, fmt.Errorf("invalid relationship type")
	}

	props := rel.Props
	edge := &model.Edge{
		ID:     getString(props, "id"),
		Type:   model.EdgeType(rel.Type),
		FromID: getString(props, "from_id"),
		ToID:   getString(props, "to_id"),
	}

	if v, ok := props["weight"].(float64); ok {
		edge.Weight = v
	}
	edge.SourceSession = getString(props, "source_session")
	// 解析 metadata JSON 字符串
	if v := getString(props, "metadata"); v != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(v), &metadata); err == nil {
			edge.Metadata = metadata
		}
	}

	return edge, nil
}

// parseEdgeWithNodes 解析边和节点
func (r *EdgeRepository) parseEdgeWithNodes(record *neo4j.Record) (*model.EdgeQueryResult, error) {
	// 解析边
	edge, err := r.parseEdge(record)
	if err != nil {
		return nil, err
	}

	result := &model.EdgeQueryResult{
		Edge: edge,
	}

	// 解析起始节点
	if fromValue, ok := record.Get("from"); ok {
		if fromNode, ok := fromValue.(neo4j.Node); ok {
			result.FromNode, _ = r.parseNodeFromProps(fromNode.Props)
		}
	}

	// 解析目标节点
	if toValue, ok := record.Get("to"); ok {
		if toNode, ok := toValue.(neo4j.Node); ok {
			result.ToNode, _ = r.parseNodeFromProps(toNode.Props)
		}
	}

	// 从关系中获取起始和目标节点ID
	relValue, _ := record.Get("r")
	if rel, ok := relValue.(neo4j.Relationship); ok {
		edge.FromID = fmt.Sprintf("%d", rel.StartId)
		edge.ToID = fmt.Sprintf("%d", rel.EndId)
	}

	return result, nil
}

// parseNodeFromProps 从属性解析节点
func (r *EdgeRepository) parseNodeFromProps(props map[string]interface{}) (*model.Node, error) {
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

	return node, nil
}
