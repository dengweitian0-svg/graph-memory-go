package service

import (
	"context"
	"fmt"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/repository"
	"github.com/example/graph-memory/pkg/logger"
)

// SessionService 会话服务
type SessionService struct {
	sessionRepo *repository.SessionRepository
	messageRepo *repository.MessageRepository
	log         *logger.Logger
}

// NewSessionService 创建会话服务
func NewSessionService(
	sessionRepo *repository.SessionRepository,
	messageRepo *repository.MessageRepository,
) *SessionService {
	return &SessionService{
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		log:         logger.NewLogger("info"),
	}
}

// CreateSession 创建会话
func (s *SessionService) CreateSession(ctx context.Context) (*model.Session, error) {
	session := model.NewSession()
	
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	s.log.Info("Session created", "id", session.ID)
	return session, nil
}

// GetSession 获取会话
func (s *SessionService) GetSession(ctx context.Context, id string) (*model.SessionWithMessages, error) {
	session, err := s.sessionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	messages, err := s.messageRepo.FindBySessionID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	return &model.SessionWithMessages{
		Session:  session,
		Messages: messages,
	}, nil
}

// ListSessions 列出会话
func (s *SessionService) ListSessions(ctx context.Context, req *ListSessionsRequest) (*ListSessionsResponse, error) {
	sessions, total, err := s.sessionRepo.List(ctx, req.Status, req.Page, req.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	return &ListSessionsResponse{
		Sessions: sessions,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// CompleteSession 完成会话
func (s *SessionService) CompleteSession(ctx context.Context, id string) error {
	session, err := s.sessionRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	session.Complete()
	
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return fmt.Errorf("failed to complete session: %w", err)
	}

	s.log.Info("Session completed", "id", id)
	return nil
}

// AddMessage 添加消息
func (s *SessionService) AddMessage(ctx context.Context, sessionID string, req *AddMessageRequest) (*model.Message, error) {
	// 获取会话
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// 创建消息
	message := model.NewMessage(sessionID, session.MessageCount, req.Role, req.Content)
	if err := message.Validate(); err != nil {
		return nil, fmt.Errorf("message validation failed: %w", err)
	}

	// 保存消息
	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// 更新会话
	session.IncrementMessageCount()
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		s.log.Error("Failed to update session message count", "error", err)
	}

	s.log.Info("Message added", "session_id", sessionID, "message_id", message.ID)
	return message, nil
}

// ListSessionsRequest 列出会话请求
type ListSessionsRequest struct {
	Status   model.SessionStatus `json:"status,omitempty"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

// ListSessionsResponse 列出会话响应
type ListSessionsResponse struct {
	Sessions []*model.Session `json:"sessions"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// AddMessageRequest 添加消息请求
type AddMessageRequest struct {
	Role    model.MessageRole `json:"role"`
	Content string            `json:"content"`
}
