package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/example/graph-memory/internal/config"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jDriver Neo4j驱动包装
type Neo4jDriver struct {
	driver neo4j.DriverWithContext
	config *config.Neo4jConfig
}

// NewNeo4jDriver 创建Neo4j驱动
func NewNeo4jDriver(cfg *config.Neo4jConfig) (*Neo4jDriver, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.URI,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	return &Neo4jDriver{
		driver: driver,
		config: cfg,
	}, nil
}

// Close 关闭连接
func (d *Neo4jDriver) Close() error {
	ctx := context.Background()
	return d.driver.Close(ctx)
}

// NewSession 创建新会话
func (d *Neo4jDriver) NewSession(ctx context.Context, mode neo4j.AccessMode) neo4j.SessionWithContext {
	return d.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode: mode,
	})
}

// ExecuteRead 执行读事务
func (d *Neo4jDriver) ExecuteRead(ctx context.Context, fn func(tx neo4j.ManagedTransaction) (interface{}, error)) (interface{}, error) {
	session := d.NewSession(ctx, neo4j.AccessModeRead)
	defer session.Close(ctx)

	return session.ExecuteRead(ctx, fn)
}

// ExecuteWrite 执行写事务
func (d *Neo4jDriver) ExecuteWrite(ctx context.Context, fn func(tx neo4j.ManagedTransaction) (interface{}, error)) (interface{}, error) {
	session := d.NewSession(ctx, neo4j.AccessModeWrite)
	defer session.Close(ctx)

	return session.ExecuteWrite(ctx, fn)
}

// HealthCheck 健康检查
func (d *Neo4jDriver) HealthCheck(ctx context.Context) error {
	session := d.NewSession(ctx, neo4j.AccessModeRead)
	defer session.Close(ctx)

	result, err := session.Run(ctx, "RETURN 1 AS test", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if !result.Next(ctx) {
		return fmt.Errorf("health check returned no result")
	}

	return nil
}

// RunQuery 执行查询
func (d *Neo4jDriver) RunQuery(ctx context.Context, cypher string, params map[string]interface{}) (neo4j.ResultWithContext, error) {
	session := d.NewSession(ctx, neo4j.AccessModeRead)
	defer session.Close(ctx)

	return session.Run(ctx, cypher, params)
}

// GetMetrics 获取连接池指标
func (d *Neo4jDriver) GetMetrics() map[string]interface{} {
	// 返回连接池状态
	return map[string]interface{}{
		"uri":       d.config.URI,
		"pool_size": d.config.MaxConnectionPoolSize,
	}
}

// BatchOperation 批量操作
func (d *Neo4jDriver) BatchOperation(ctx context.Context, operations []func(tx neo4j.ManagedTransaction) error) error {
	_, err := d.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for _, op := range operations {
			if err := op(tx); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}

// WithRetry 带重试的操作
func (d *Neo4jDriver) WithRetry(ctx context.Context, maxRetries int, fn func() error) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		// 指数退避
		backoff := time.Duration(1<<uint(i)) * 100 * time.Millisecond
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			continue
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
