package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/example/graph-memory/internal/model"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// NodeRepository 节点仓库
type NodeRepository struct {
	driver *Neo4jDriver
}

// NewNodeRepository 创建节点仓库
func NewNodeRepository(driver *Neo4jDriver) *NodeRepository {
	return &NodeRepository{driver: driver}
}

// Create 创建节点
func (r *NodeRepository) Create(ctx context.Context, node *model.Node) error {
	if err := node.Validate(); err != nil {
		return err
	}

	_, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// 将 metadata 序列化为 JSON 字符串，Neo4j 不支持嵌套 map
		var metadataJSON string
		if len(node.Metadata) > 0 {
			jsonBytes, err := json.Marshal(node.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metadata: %w", err)
			}
			metadataJSON = string(jsonBytes)
		}

		params := map[string]interface{}{
			"id":              node.ID,
			"name":            node.Name,
			"type":            string(node.Type),
			"description":     node.Description,
			"status":          string(node.Status),
			"validated_count": node.ValidatedCount,
			"pagerank":        node.PageRank,
			"community_id":    node.CommunityID,
			"created_at":      node.CreatedAt,
			"updated_at":      node.UpdatedAt,
			"source_sessions": node.SourceSessions,
			"embedding_hash":  node.EmbeddingHash,
			"embedding":       node.Embedding,
			"metadata":        metadataJSON,
		}

		_, err := tx.Run(ctx, `
			CREATE (n:Node {
				id: $id,
				name: $name,
				type: $type,
				description: $description,
				status: $status,
				validated_count: $validated_count,
				pagerank: $pagerank,
				community_id: $community_id,
				created_at: $created_at,
				updated_at: $updated_at,
				source_sessions: $source_sessions,
				embedding_hash: $embedding_hash,
				embedding: $embedding,
				metadata: $metadata
			})
			RETURN n.id AS id
		`, params)

		return nil, err
	})

	return err
}

// FindByID 根据ID查找节点
func (r *NodeRepository) FindByID(ctx context.Context, id string) (*model.Node, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (n:Node {id: $id})
			RETURN n
		`, map[string]interface{}{"id": id})

		if err != nil {
			return nil, err
		}

		if !record.Next(ctx) {
			return nil, model.ErrNodeNotFound
		}

		node, err := r.parseNode(record.Record())
		if err != nil {
			return nil, err
		}

		return node, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*model.Node), nil
}

// FindByIDs 根据ID列表查找节点
func (r *NodeRepository) FindByIDs(ctx context.Context, ids []string) ([]*model.Node, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (n:Node)
			WHERE n.id IN $ids
			RETURN n
		`, map[string]interface{}{"ids": ids})

		if err != nil {
			return nil, err
		}

		nodes := make([]*model.Node, 0)
		for record.Next(ctx) {
			node, err := r.parseNode(record.Record())
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}

		return nodes, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*model.Node), nil
}

// FindByType 根据类型查找节点
func (r *NodeRepository) FindByType(ctx context.Context, nodeType model.NodeType, limit int) ([]*model.Node, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (n:Node {type: $type})
			RETURN n
			ORDER BY n.created_at DESC
			LIMIT $limit
		`, map[string]interface{}{
			"type":  string(nodeType),
			"limit": limit,
		})

		if err != nil {
			return nil, err
		}

		nodes := make([]*model.Node, 0)
		for record.Next(ctx) {
			node, err := r.parseNode(record.Record())
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}

		return nodes, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*model.Node), nil
}

// FindByCommunityID 根据社区ID查找节点
func (r *NodeRepository) FindByCommunityID(ctx context.Context, communityID string) ([]*model.Node, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (n:Node {community_id: $community_id})
			RETURN n
			ORDER BY n.pagerank DESC
		`, map[string]interface{}{"community_id": communityID})

		if err != nil {
			return nil, err
		}

		nodes := make([]*model.Node, 0)
		for record.Next(ctx) {
			node, err := r.parseNode(record.Record())
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}

		return nodes, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*model.Node), nil
}

// Update 更新节点
func (r *NodeRepository) Update(ctx context.Context, node *model.Node) error {
	_, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// 将 metadata 序列化为 JSON 字符串，Neo4j 不支持嵌套 map
		var metadataJSON string
		if len(node.Metadata) > 0 {
			jsonBytes, err := json.Marshal(node.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metadata: %w", err)
			}
			metadataJSON = string(jsonBytes)
		}

		params := map[string]interface{}{
			"id":              node.ID,
			"name":            node.Name,
			"description":     node.Description,
			"status":          string(node.Status),
			"validated_count": node.ValidatedCount,
			"pagerank":        node.PageRank,
			"community_id":    node.CommunityID,
			"updated_at":      time.Now().UTC(), // 使用 UTC 时区，避免 Neo4j 时区错误
			"source_sessions": node.SourceSessions,
			"embedding_hash":  node.EmbeddingHash,
			"embedding":       node.Embedding,
			"metadata":        metadataJSON,
		}

		_, err := tx.Run(ctx, `
			MATCH (n:Node {id: $id})
			SET n.name = $name,
			    n.description = $description,
			    n.status = $status,
			    n.validated_count = $validated_count,
			    n.pagerank = $pagerank,
			    n.community_id = $community_id,
			    n.updated_at = $updated_at,
			    n.source_sessions = $source_sessions,
			    n.embedding_hash = $embedding_hash,
			    n.embedding = $embedding,
			    n.metadata = $metadata
		`, params)

		return nil, err
	})

	return err
}

// Delete 删除节点
func (r *NodeRepository) Delete(ctx context.Context, id string) error {
	_, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, `
			MATCH (n:Node {id: $id})
			DETACH DELETE n
		`, map[string]interface{}{"id": id})

		return nil, err
	})

	return err
}

// SearchByName 按名称搜索
func (r *NodeRepository) SearchByName(ctx context.Context, name string, limit int) ([]*model.Node, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (n:Node)
			WHERE n.name CONTAINS $name
			RETURN n
			ORDER BY n.pagerank DESC
			LIMIT $limit
		`, map[string]interface{}{
			"name":  name,
			"limit": limit,
		})

		if err != nil {
			return nil, err
		}

		nodes := make([]*model.Node, 0)
		for record.Next(ctx) {
			node, err := r.parseNode(record.Record())
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}

		return nodes, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*model.Node), nil
}

// List 列出节点
func (r *NodeRepository) List(ctx context.Context, nodeType model.NodeType, status model.NodeStatus, page, pageSize int) ([]*model.Node, int, error) {
	// Defensive validation: ensure valid pagination parameters
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// 构建查询条件
		whereClause := ""
		params := map[string]interface{}{
			"skip":  (page - 1) * pageSize,
			"limit": pageSize,
		}

		if nodeType != "" {
			whereClause = "WHERE n.type = $type"
			params["type"] = string(nodeType)
		}
		if status != "" {
			if whereClause != "" {
				whereClause += " AND n.status = $status"
			} else {
				whereClause = "WHERE n.status = $status"
			}
			params["status"] = string(status)
		}

		// 查询总数
		countQuery := "MATCH (n:Node) " + whereClause + " RETURN count(n) AS count"
		countRecord, err := tx.Run(ctx, countQuery, params)
		if err != nil {
			return nil, err
		}

		var total int
		if countRecord.Next(ctx) {
			count, _ := countRecord.Record().Get("count")
			total = int(count.(int64))
		}

		// 查询节点
		query := `MATCH (n:Node) ` + whereClause + `
			RETURN n
			ORDER BY n.created_at DESC
			SKIP $skip
			LIMIT $limit`

		record, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		nodes := make([]*model.Node, 0)
		for record.Next(ctx) {
			node, err := r.parseNode(record.Record())
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}

		return map[string]interface{}{
			"nodes": nodes,
			"total": total,
		}, nil
	})

	if err != nil {
		return nil, 0, err
	}

	data := result.(map[string]interface{})
	return data["nodes"].([]*model.Node), data["total"].(int), nil
}

// Search 搜索节点（支持名称和描述搜索）
func (r *NodeRepository) Search(ctx context.Context, query string, limit int) ([]*model.Node, error) {
	// Defensive validation: ensure valid limit parameter
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (n:Node)
			WHERE n.name CONTAINS $query OR n.description CONTAINS $query
			RETURN n
			ORDER BY n.pagerank DESC
			LIMIT $limit
		`, map[string]interface{}{
			"query": query,
			"limit": limit,
		})

		if err != nil {
			return nil, err
		}

		nodes := make([]*model.Node, 0)
		for record.Next(ctx) {
			node, err := r.parseNode(record.Record())
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}

		return nodes, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*model.Node), nil
}

// Count 统计节点数量
func (r *NodeRepository) Count(ctx context.Context) (int64, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (n:Node)
			RETURN count(n) AS count
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

// parseNode 解析节点
func (r *NodeRepository) parseNode(record *neo4j.Record) (*model.Node, error) {
	nodeValue, ok := record.Get("n")
	if !ok {
		return nil, fmt.Errorf("node not found in record")
	}

	nodeProps, ok := nodeValue.(neo4j.Node)
	if !ok {
		return nil, fmt.Errorf("invalid node type")
	}

	props := nodeProps.Props

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
	if v, ok := props["source_sessions"].([]interface{}); ok {
		node.SourceSessions = make([]string, 0, len(v))
		for _, s := range v {
			if str, ok := s.(string); ok {
				node.SourceSessions = append(node.SourceSessions, str)
			}
		}
	}
	node.EmbeddingHash = getString(props, "embedding_hash")
	if v, ok := props["embedding"].([]interface{}); ok {
		node.Embedding = make([]float32, 0, len(v))
		for _, f := range v {
			if val, ok := f.(float64); ok {
				node.Embedding = append(node.Embedding, float32(val))
			}
		}
	}
	
	// 反序列化 metadata JSON 字符串
	if metadataStr := getString(props, "metadata"); metadataStr != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
			node.Metadata = metadata
		}
	}
	if node.Metadata == nil {
		node.Metadata = make(map[string]interface{})
	}

	return node, nil
}

// getString 从map中获取字符串
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
