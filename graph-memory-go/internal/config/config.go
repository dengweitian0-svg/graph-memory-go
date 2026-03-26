package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Neo4j      Neo4jConfig      `mapstructure:"neo4j"`
	Redis      RedisConfig      `mapstructure:"redis"`
	LLM        LLMConfig        `mapstructure:"llm"`
	Algorithms AlgorithmsConfig `mapstructure:"algorithms"`
	Recall     RecallConfig     `mapstructure:"recall"`
	Cache      CacheConfig      `mapstructure:"cache"`
	Logging    LoggingConfig    `mapstructure:"logging"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
}

// ServerConfig 服务配置
type ServerConfig struct {
	HTTP      HTTPConfig      `mapstructure:"http"`
	GRPC      GRPCConfig      `mapstructure:"grpc"`
	WebSocket WebSocketConfig `mapstructure:"websocket"`
}

// HTTPConfig HTTP服务配置
type HTTPConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// GRPCConfig gRPC服务配置
type GRPCConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// WebSocketConfig WebSocket服务配置
type WebSocketConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// Neo4jConfig Neo4j配置
type Neo4jConfig struct {
	URI                     string        `mapstructure:"uri"`
	Username                string        `mapstructure:"username"`
	Password                string        `mapstructure:"password"`
	MaxConnectionPoolSize   int           `mapstructure:"max_connection_pool_size"`
	ConnectionTimeout       time.Duration `mapstructure:"connection_timeout"`
	MaxTransactionRetryTime time.Duration `mapstructure:"max_transaction_retry_time"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// LLMConfig LLM配置 - 基于eino框架
type LLMConfig struct {
	// API配置
	APIKey  string `mapstructure:"api_key"`   // API密钥
	BaseURL string `mapstructure:"base_url"`  // API地址

	// 模型配置
	ChatModel     string  `mapstructure:"chat_model"`      // 聊天模型ID
	EmbeddingModel string `mapstructure:"embedding_model"` // 嵌入模型ID

	// 生成参数
	Temperature float64 `mapstructure:"temperature"` // 温度参数
	MaxTokens   int     `mapstructure:"max_tokens"`   // 最大Token数

	// 超时设置
	Timeout time.Duration `mapstructure:"timeout"`

	// 功能开关
	Enabled bool `mapstructure:"enabled"`
}

// AlgorithmsConfig 算法配置
type AlgorithmsConfig struct {
	PageRank          PageRankConfig          `mapstructure:"pagerank"`
	CommunityDetection CommunityDetectionConfig `mapstructure:"community_detection"`
	Deduplication     DeduplicationConfig     `mapstructure:"deduplication"`
}

// PageRankConfig PageRank算法配置
type PageRankConfig struct {
	DampingFactor    float64 `mapstructure:"damping_factor"`
	MaxIterations    int     `mapstructure:"max_iterations"`
	ConvergenceDelta float64 `mapstructure:"convergence_delta"`
}

// CommunityDetectionConfig 社区检测配置
type CommunityDetectionConfig struct {
	MaxIterations int `mapstructure:"max_iterations"`
}

// DeduplicationConfig 去重配置
type DeduplicationConfig struct {
	SimilarityThreshold float64 `mapstructure:"similarity_threshold"`
}

// RecallConfig 召回配置
type RecallConfig struct {
	DualPath DualPathConfig `mapstructure:"dual_path"`
}

// DualPathConfig 双路径召回配置
type DualPathConfig struct {
	PreciseWeight     float64 `mapstructure:"precise_weight"`
	GeneralizedWeight float64 `mapstructure:"generalized_weight"`
	Limit             int     `mapstructure:"limit"`
	MaxDepth          int     `mapstructure:"max_depth"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	L1 CacheLayerConfig `mapstructure:"l1"`
	L2 CacheLayerConfig `mapstructure:"l2"`
}

// CacheLayerConfig 缓存层配置
type CacheLayerConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	TTL     time.Duration `mapstructure:"ttl"`
	MaxSize int           `mapstructure:"max_size"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	OutputPath string `mapstructure:"output_path"`
	MaxSize    int64  `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	Enabled    bool             `mapstructure:"enabled"`
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
	Tracing    TracingConfig    `mapstructure:"tracing"`
}

// PrometheusConfig Prometheus配置
type PrometheusConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

// TracingConfig 追踪配置
type TracingConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Endpoint string `mapstructure:"endpoint"`
}

// Load 加载配置
func Load() (*Config, error) {
	return LoadWithPath("")
}

// LoadWithPath 从指定路径加载配置
// 支持多路径搜索，兼容 Windows 和 Linux
func LoadWithPath(configPath string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 配置文件路径搜索策略
	if configPath != "" {
		// 1. 显式指定的配置文件路径（最高优先级）
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")

		// 2. 按优先级添加搜索路径
		configPaths := getConfigSearchPaths()
		for _, path := range configPaths {
			v.AddConfigPath(path)
		}
	}

	// 环境变量
	v.SetEnvPrefix("GRAPH_MEMORY")
	v.AutomaticEnv()

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// 配置文件不存在时使用默认值，输出警告
		fmt.Fprintf(os.Stderr, "[WARN] Config file not found, using default values\n")
	} else {
		// 输出配置文件路径便于调试
		fmt.Fprintf(os.Stderr, "[INFO] Using config file: %s\n", v.ConfigFileUsed())
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// getConfigSearchPaths 获取配置文件搜索路径
// 按优先级从高到低返回，兼容 Windows 和 Linux
func getConfigSearchPaths() []string {
	var paths []string

	// 1. 当前工作目录下的 config 子目录
	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, "config"))
		paths = append(paths, wd) // 也搜索当前目录
	}

	// 2. 可执行文件所在目录下的 config 子目录
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		paths = append(paths, filepath.Join(exeDir, "config"))
		paths = append(paths, exeDir)
	}

	// 3. 项目根目录（向上查找 go.mod）
	if projectRoot := findProjectRoot(); projectRoot != "" {
		paths = append(paths, filepath.Join(projectRoot, "config"))
		paths = append(paths, projectRoot)
	}

	// 4. 系统级配置目录（Linux/macOS）
	if runtime.GOOS != "windows" {
		paths = append(paths, "/etc/graph-memory")
		paths = append(paths, "/usr/local/etc/graph-memory")
	} else {
		// Windows: 使用 APPDATA 目录
		if appData := os.Getenv("APPDATA"); appData != "" {
			paths = append(paths, filepath.Join(appData, "graph-memory"))
		}
		// Windows: 使用 ProgramData 目录（系统级）
		if programData := os.Getenv("ProgramData"); programData != "" {
			paths = append(paths, filepath.Join(programData, "graph-memory"))
		}
	}

	return paths
}

// findProjectRoot 向上查找项目根目录（通过 go.mod 文件）
func findProjectRoot() string {
	// 从当前目录开始向上查找
	startDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := startDir
	for {
		// 检查是否存在 go.mod
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}

		// 向上一级目录
		parent := filepath.Dir(dir)
		if parent == dir {
			// 已到达根目录，停止搜索
			break
		}
		dir = parent
	}

	// 也从可执行文件目录开始查找
	if exePath, err := os.Executable(); err == nil {
		dir = filepath.Dir(exePath)
		for {
			goModPath := filepath.Join(dir, "go.mod")
			if _, err := os.Stat(goModPath); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return ""
}

// setDefaults 设置默认值
func setDefaults(v *viper.Viper) {
	// Server
	v.SetDefault("server.http.host", "0.0.0.0")
	v.SetDefault("server.http.port", 8080)
	v.SetDefault("server.grpc.host", "0.0.0.0")
	v.SetDefault("server.grpc.port", 9090)
	v.SetDefault("server.websocket.host", "0.0.0.0")
	v.SetDefault("server.websocket.port", 8081)

	// Neo4j
	v.SetDefault("neo4j.uri", "bolt://localhost:7687")
	v.SetDefault("neo4j.username", "neo4j")
	v.SetDefault("neo4j.password", "password")
	v.SetDefault("neo4j.max_connection_pool_size", 50)
	v.SetDefault("neo4j.connection_timeout", "30s")
	v.SetDefault("neo4j.max_transaction_retry_time", "30s")

	// Redis
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 20)

	// LLM - eino框架配置
	v.SetDefault("llm.api_key", "")
	v.SetDefault("llm.base_url", "https://ark.cn-beijing.volces.com/api/v3")
	v.SetDefault("llm.chat_model", "deepseek-v3-2-251201")
	v.SetDefault("llm.embedding_model", "doubao-embedding-vision-251215")
	v.SetDefault("llm.temperature", 0.7)
	v.SetDefault("llm.max_tokens", 4096)
	v.SetDefault("llm.timeout", "60s")
	v.SetDefault("llm.enabled", true)

	// Algorithms
	v.SetDefault("algorithms.pagerank.damping_factor", 0.85)
	v.SetDefault("algorithms.pagerank.max_iterations", 20)
	v.SetDefault("algorithms.pagerank.convergence_delta", 1e-6)
	v.SetDefault("algorithms.community_detection.max_iterations", 10)
	v.SetDefault("algorithms.deduplication.similarity_threshold", 0.95)

	// Recall
	v.SetDefault("recall.dual_path.precise_weight", 0.6)
	v.SetDefault("recall.dual_path.generalized_weight", 0.4)
	v.SetDefault("recall.dual_path.limit", 20)
	v.SetDefault("recall.dual_path.max_depth", 3)

	// Cache
	v.SetDefault("cache.l1.enabled", true)
	v.SetDefault("cache.l1.ttl", "30s")
	v.SetDefault("cache.l1.max_size", 10000)
	v.SetDefault("cache.l2.enabled", true)
	v.SetDefault("cache.l2.ttl", "5m")

	// Logging
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.output_path", "./log/app.log")
	v.SetDefault("logging.max_size", 104857600) // 100MB
	v.SetDefault("logging.max_backups", 10)
	v.SetDefault("logging.max_age", 30)
	v.SetDefault("logging.compress", true)

	// Monitoring
	v.SetDefault("monitoring.enabled", true)
	v.SetDefault("monitoring.prometheus.enabled", true)
	v.SetDefault("monitoring.prometheus.port", 9090)
	v.SetDefault("monitoring.tracing.enabled", true)
	v.SetDefault("monitoring.tracing.endpoint", "localhost:4317")
}
