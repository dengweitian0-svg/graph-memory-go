package main

import (
	"context"
	"fmt"
	"log"

	"github.com/example/graph-memory/internal/config"
	"github.com/example/graph-memory/internal/repository"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 连接Neo4j
	driver, err := repository.NewNeo4jDriver(&cfg.Neo4j)
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer driver.Close()

	ctx := context.Background()

	// 初始化Schema
	if err := initSchema(ctx, driver); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	log.Println("Database initialization completed successfully!")
}

func initSchema(ctx context.Context, driver *repository.Neo4jDriver) error {
	// 创建约束
	constraints := []string{
		// Node约束
		`CREATE CONSTRAINT node_id_unique IF NOT EXISTS FOR (n:Node) REQUIRE n.id IS UNIQUE`,
		
		// Community约束
		`CREATE CONSTRAINT community_id_unique IF NOT EXISTS FOR (c:Community) REQUIRE c.id IS UNIQUE`,
		
		// Message约束
		`CREATE CONSTRAINT message_id_unique IF NOT EXISTS FOR (m:Message) REQUIRE m.id IS UNIQUE`,
		
		// Session约束
		`CREATE CONSTRAINT session_id_unique IF NOT EXISTS FOR (s:Session) REQUIRE s.id IS UNIQUE`,
	}

	// 创建索引
	indexes := []string{
		// Node索引
		`CREATE INDEX node_type_index IF NOT EXISTS FOR (n:Node) ON (n.type)`,
		`CREATE INDEX node_status_index IF NOT EXISTS FOR (n:Node) ON (n.status)`,
		`CREATE INDEX node_community_index IF NOT EXISTS FOR (n:Node) ON (n.community_id)`,
		`CREATE INDEX node_updated_index IF NOT EXISTS FOR (n:Node) ON (n.updated_at)`,
		`CREATE INDEX node_name_index IF NOT EXISTS FOR (n:Node) ON (n.name)`,
		
		// Message索引
		`CREATE INDEX message_session_index IF NOT EXISTS FOR (m:Message) ON (m.session_id)`,
		`CREATE INDEX message_turn_index IF NOT EXISTS FOR (m:Message) ON (m.turn_index)`,
	}

	// 创建向量索引（Neo4j 5.x+支持）
	vectorIndexes := []string{
		// Node向量索引
		`CALL db.index.vector.createNodeIndex('node_embedding_index', 'Node', 'embedding', 1536, 'cosine')`,
		
		// Community向量索引
		`CALL db.index.vector.createNodeIndex('community_embedding_index', 'Community', 'embedding', 1536, 'cosine')`,
	}

	// 创建全文索引
	fulltextIndexes := []string{
		`CALL db.index.fulltext.createNodeIndex('node_fulltext_index', ['Node'], ['name', 'description'])`,
	}

	// 执行约束创建
	fmt.Println("Creating constraints...")
	for _, constraint := range constraints {
		if err := executeCypher(ctx, driver, constraint); err != nil {
			log.Printf("Warning: Failed to create constraint: %v", err)
		} else {
			fmt.Printf("Created: %s\n", truncate(constraint, 60))
		}
	}

	// 执行索引创建
	fmt.Println("\nCreating indexes...")
	for _, index := range indexes {
		if err := executeCypher(ctx, driver, index); err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
		} else {
			fmt.Printf("Created: %s\n", truncate(index, 60))
		}
	}

	// 执行向量索引创建
	fmt.Println("\nCreating vector indexes...")
	for _, vIndex := range vectorIndexes {
		if err := executeCypher(ctx, driver, vIndex); err != nil {
			log.Printf("Warning: Failed to create vector index (may already exist): %v", err)
		} else {
			fmt.Printf("Created: %s\n", truncate(vIndex, 60))
		}
	}

	// 执行全文索引创建
	fmt.Println("\nCreating fulltext indexes...")
	for _, ftIndex := range fulltextIndexes {
		if err := executeCypher(ctx, driver, ftIndex); err != nil {
			log.Printf("Warning: Failed to create fulltext index (may already exist): %v", err)
		} else {
			fmt.Printf("Created: %s\n", truncate(ftIndex, 60))
		}
	}

	return nil
}

func executeCypher(ctx context.Context, driver *repository.Neo4jDriver, cypher string) error {
	_, err := driver.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		_, err := tx.Run(ctx, cypher, nil)
		return nil, err
	})
	return err
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
