package main

import (
	"context"
	"fmt"
	"os"

	"github.com/example/graph-memory/internal/config"
	"github.com/example/graph-memory/internal/repository"
)

func main() {
	fmt.Println("=== Neo4j Connection Test ===")
	
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📋 Configuration loaded:\n")
	fmt.Printf("   URI: %s\n", cfg.Neo4j.URI)
	fmt.Printf("   Username: %s\n", cfg.Neo4j.Username)
	fmt.Printf("   Password: %s\n", maskPassword(cfg.Neo4j.Password))
	fmt.Println()

	// 尝试连接
	fmt.Println("🔌 Attempting to connect to Neo4j...")
	driver, err := repository.NewNeo4jDriver(&cfg.Neo4j)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create driver: %v\n", err)
		os.Exit(1)
	}
	defer driver.Close()

	fmt.Println("✅ Driver created successfully")

	// 健康检查
	fmt.Println("🏥 Running health check...")
	ctx := context.Background()
	if err := driver.HealthCheck(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Health check failed: %v\n", err)
		fmt.Println("\n💡 Troubleshooting tips:")
		fmt.Println("   1. Verify Neo4j is running: docker ps | grep neo4j")
		fmt.Println("   2. Check credentials in config/config.yaml")
		fmt.Println("   3. Try connecting with cypher-shell:")
		fmt.Printf("      cypher-shell -a %s -u %s -p <password>\n", cfg.Neo4j.URI, cfg.Neo4j.Username)
		fmt.Println("   4. Check Neo4j logs for authentication errors")
		os.Exit(1)
	}

	fmt.Println("✅ Health check passed!")
	fmt.Println("\n🎉 Neo4j connection is working correctly!")
}

func maskPassword(password string) string {
	if len(password) <= 2 {
		return "***"
	}
	return password[:2] + "***" + password[len(password)-2:]
}
