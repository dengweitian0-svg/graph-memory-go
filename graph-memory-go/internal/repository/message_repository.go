package repository

import (
	"context"
	"fmt"

	"github.com/example/graph-memory/internal/model"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// MessageRepository 消息仓库
type MessageRepository struct {
	driver *Neo4jDriver
}

// NewMessageRepository 创建消息仓库
func NewMessageRepository(driver *Neo4jDriver) *MessageRepository {
	return &MessageRepository{driver: driver}
}

// Create 创建消息
func (r *MessageRepository) Create(ctx context.Context, message *model.Message) error {
	_, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, `
			CREATE (m:Message {
				id: $id,
				session_id: $session_id,
				turn_index: $turn_index,
				role: $role,
				content: $content,
				timestamp: $timestamp,
				extracted: $extracted,
				metadata: $metadata
			})
			RETURN m.id AS id
		`, map[string]interface{}{
			"id":         message.ID,
			"session_id": message.SessionID,
			"turn_index": message.TurnIndex,
			"role":       string(message.Role),
			"content":    message.Content,
			"timestamp":  message.Timestamp,
			"extracted":  message.Extracted,
			"metadata":   message.Metadata,
		})

		return nil, err
	})

	return err
}

// FindByID 根据ID查找消息
func (r *MessageRepository) FindByID(ctx context.Context, id string) (*model.Message, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (m:Message {id: $id})
			RETURN m
		`, map[string]interface{}{"id": id})

		if err != nil {
			return nil, err
		}

		if !record.Next(ctx) {
			return nil, model.ErrMessageNotFound
		}

		message, err := r.parseMessage(record.Record())
		if err != nil {
			return nil, err
		}

		return message, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*model.Message), nil
}

// FindBySessionID 根据会话ID查找消息
func (r *MessageRepository) FindBySessionID(ctx context.Context, sessionID string) ([]*model.Message, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (m:Message {session_id: $session_id})
			RETURN m
			ORDER BY m.turn_index ASC
		`, map[string]interface{}{"session_id": sessionID})

		if err != nil {
			return nil, err
		}

		messages := make([]*model.Message, 0)
		for record.Next(ctx) {
			message, err := r.parseMessage(record.Record())
			if err != nil {
				return nil, err
			}
			messages = append(messages, message)
		}

		return messages, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*model.Message), nil
}

// FindUnextracted 查找未提取的消息
func (r *MessageRepository) FindUnextracted(ctx context.Context, limit int) ([]*model.Message, error) {
	result, err := r.driver.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		record, err := tx.Run(ctx, `
			MATCH (m:Message {extracted: false})
			RETURN m
			ORDER BY m.timestamp ASC
			LIMIT $limit
		`, map[string]interface{}{"limit": limit})

		if err != nil {
			return nil, err
		}

		messages := make([]*model.Message, 0)
		for record.Next(ctx) {
			message, err := r.parseMessage(record.Record())
			if err != nil {
				return nil, err
			}
			messages = append(messages, message)
		}

		return messages, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]*model.Message), nil
}

// MarkExtracted 标记消息为已提取
func (r *MessageRepository) MarkExtracted(ctx context.Context, id string) error {
	_, err := r.driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, `
			MATCH (m:Message {id: $id})
			SET m.extracted = true
		`, map[string]interface{}{"id": id})

		return nil, err
	})

	return err
}

// parseMessage 解析消息
func (r *MessageRepository) parseMessage(record *neo4j.Record) (*model.Message, error) {
	messageValue, ok := record.Get("m")
	if !ok {
		return nil, fmt.Errorf("message not found in record")
	}

	messageProps, ok := messageValue.(neo4j.Node)
	if !ok {
		return nil, fmt.Errorf("invalid message type")
	}

	props := messageProps.Props
	message := &model.Message{
		ID:        getString(props, "id"),
		SessionID: getString(props, "session_id"),
		Role:      model.MessageRole(getString(props, "role")),
		Content:   getString(props, "content"),
		Extracted: false,
	}

	if v, ok := props["turn_index"].(int64); ok {
		message.TurnIndex = int(v)
	}
	if v, ok := props["extracted"].(bool); ok {
		message.Extracted = v
	}
	if v, ok := props["metadata"].(map[string]interface{}); ok {
		message.Metadata = v
	}

	return message, nil
}
