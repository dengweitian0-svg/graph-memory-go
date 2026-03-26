package model

import (
	"errors"
)

// 节点相关错误
var (
	ErrNodeNotFound          = errors.New("node not found")
	ErrNodeNameRequired      = errors.New("node name is required")
	ErrNodeTypeRequired      = errors.New("node type is required")
	ErrNodeDescriptionRequired = errors.New("node description is required")
	ErrNodeAlreadyExists     = errors.New("node already exists")
)

// 边相关错误
var (
	ErrEdgeNotFound      = errors.New("edge not found")
	ErrEdgeFromIDRequired = errors.New("edge from_id is required")
	ErrEdgeToIDRequired   = errors.New("edge to_id is required")
	ErrEdgeTypeRequired   = errors.New("edge type is required")
	ErrEdgeSelfLoop       = errors.New("self-loop edge is not allowed")
	ErrEdgeAlreadyExists  = errors.New("edge already exists")
)

// 消息相关错误
var (
	ErrMessageNotFound         = errors.New("message not found")
	ErrMessageSessionIDRequired = errors.New("message session_id is required")
	ErrMessageRoleRequired     = errors.New("message role is required")
	ErrMessageContentRequired  = errors.New("message content is required")
)

// 会话相关错误
var (
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionIDRequired   = errors.New("session id is required")
	ErrSessionAlreadyEnded = errors.New("session already ended")
)

// 社区相关错误
var (
	ErrCommunityNotFound = errors.New("community not found")
)

// 图相关错误
var (
	ErrGraphQueryFailed   = errors.New("graph query failed")
	ErrTransactionFailed  = errors.New("transaction failed")
	ErrConnectionFailed   = errors.New("connection failed")
)

// 工作流相关错误
var (
	ErrWorkflowNotFound     = errors.New("workflow not found")
	ErrWorkflowIDRequired   = errors.New("workflow id is required")
	ErrWorkflowAlreadyRunning = errors.New("workflow already running")
)

// 算法相关错误
var (
	ErrInvalidSeedNodes    = errors.New("invalid seed nodes for pagerank")
	ErrInvalidCandidateNodes = errors.New("invalid candidate nodes for pagerank")
	ErrConvergenceNotReached = errors.New("pagerank convergence not reached")
)

// 向量相关错误
var (
	ErrInvalidEmbedding     = errors.New("invalid embedding")
	ErrEmbeddingNotSet      = errors.New("embedding not set")
	ErrEmbeddingDimension   = errors.New("embedding dimension mismatch")
)

// 通用错误
var (
	ErrInvalidInput    = errors.New("invalid input")
	ErrInternalError   = errors.New("internal error")
	ErrTimeout         = errors.New("operation timeout")
	ErrNotImplemented  = errors.New("not implemented")
)
