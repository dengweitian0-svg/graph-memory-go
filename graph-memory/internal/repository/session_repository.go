package repository

import (
	"context"
	"fmt"

	"github.com/example/graph-memory/internal/model"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// SessionRepository 会话仓库
type SessionRepository struct {
	driver *Neo4jDriver
}

// NewSessionRepository 创建会话仓库
func NewSessionRepository(driver *Neo4jDriver) *SessionRepository {
	return &SessionRepository{driver: driver}
}

// Create 创建会话
func (r *SessionRepository) Create(ctx context.Context, session *model.Session) error {
	_, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, `
			CREATE (s:Session {
				id: $id,
				status: $status,
				started_at: $started_at,
				message_count: $message_count,
				metadata: $metadata
			})
			RETURN s.id AS id
		`, map[string]interface{}{
			"id":            session.ID,
			"status":        string(session.Status),
			"started_at":    session.StartedAt,
			"message_count": session.MessageCount,
			"metadata":      session.Metadata,
		})

		return nil, err
	})

	return err
}

// FindByID 根据ID查找会话
func (r *SessionRepository) FindByID(ctx context.Context, id string) (*model.Session, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (s:Session {id: $id})
			RETURN s
		`, map[string]interface{}{"id": id})

		if err != nil {
			return nil, err
		}

		if !record.Next(ctx) {
			return nil, model.ErrSessionNotFound
		}

		session, err := r.parseSession(record.Record())
		if err != nil {
			return nil, err
		}

		return session, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*model.Session), nil
}

// Update 更新会话
func (r *SessionRepository) Update(ctx context.Context, session *model.Session) error {
	_, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		params := map[string]interface{}{
			"id":            session.ID,
			"status":        string(session.Status),
			"message_count": session.MessageCount,
			"metadata":      session.Metadata,
		}

		if session.EndedAt != nil {
			params["ended_at"] = *session.EndedAt
		}

		_, err := tx.Run(ctx, `
			MATCH (s:Session {id: $id})
			SET s.status = $status,
			    s.message_count = $message_count,
			    s.metadata = $metadata
		`, params)

		return nil, err
	})

	return err
}

// List 列出会话
func (r *SessionRepository) List(ctx context.Context, status model.SessionStatus, page, pageSize int) ([]*model.Session, int, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		params := map[string]interface{}{
			"skip":  (page - 1) * pageSize,
			"limit": pageSize,
		}

		whereClause := ""
		if status != "" {
			whereClause = "WHERE s.status = $status"
			params["status"] = string(status)
		}

		// 查询总数
		countQuery := "MATCH (s:Session) " + whereClause + " RETURN count(s) AS count"
		countRecord, err := tx.Run(ctx, countQuery, params)
		if err != nil {
			return nil, err
		}

		var total int
		if countRecord.Next(ctx) {
			count, _ := countRecord.Record().Get("count")
			total = int(count.(int64))
		}

		// 查询会话
		query := `MATCH (s:Session) ` + whereClause + `
			RETURN s
			ORDER BY s.started_at DESC
			SKIP $skip
			LIMIT $limit`

		record, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		sessions := make([]*model.Session, 0)
		for record.Next(ctx) {
			session, err := r.parseSession(record.Record())
			if err != nil {
				return nil, err
			}
			sessions = append(sessions, session)
		}

		return map[string]interface{}{
			"sessions": sessions,
			"total":    total,
		}, nil
	})

	if err != nil {
		return nil, 0, err
	}

	data := result.(map[string]interface{})
	return data["sessions"].([]*model.Session), data["total"].(int), nil
}

// parseSession 解析会话
func (r *SessionRepository) parseSession(record *neo4j.Record) (*model.Session, error) {
	sessionValue, ok := record.Get("s")
	if !ok {
		return nil, fmt.Errorf("session not found in record")
	}

	sessionProps, ok := sessionValue.(neo4j.Node)
	if !ok {
		return nil, fmt.Errorf("invalid session type")
	}

	props := sessionProps.Props
	session := &model.Session{
		ID:     getString(props, "id"),
		Status: model.SessionStatus(getString(props, "status")),
	}

	if v, ok := props["message_count"].(int64); ok {
		session.MessageCount = int(v)
	}
	if v, ok := props["metadata"].(map[string]interface{}); ok {
		session.Metadata = v
	}

	return session, nil
}
