# Graph Memory

[![Go Version](https://img.shields.io/badge/Go-1.22-blue.svg)](https://golang.org)
[![Neo4j](https://img.shields.io/badge/Neo4j-5.15-green.svg)](https://neo4j.com)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Graph Memory 是一个基于知识图谱的上下文引擎，帮助 AI 代理解决上下文爆炸、跨会话遗忘和技能孤岛问题。

## 特性

- 🧠 **知识图谱存储**: 使用 Neo4j 存储和管理知识节点及其关系
- 🔍 **双路径召回**: 结合精确路径和泛化路径的上下文检索
- 📊 **图算法支持**: PageRank、社区检测、向量去重等核心算法
- 🚀 **高性能**: 多级缓存、并行计算、查询优化
- 🔌 **多协议支持**: gRPC、HTTP REST、WebSocket

## 快速开始

### 前置条件

- Go 1.23+
- Docker

### 启动服务

```bash
# 克隆项目
git clone https://github.com/example/graph-memory.git
cd graph-memory

# 启动依赖服务（Neo4j、Redis、Prometheus、Grafana）
docker-compose up -d neo4j redis prometheus grafana

# 等待 Neo4j 启动（约30秒）
# 初始化数据库
make db-init

# 编译并运行
make run
```

### 访问服务

| 服务 | 地址 |
|------|------|
| HTTP API | http://localhost:8080 |
| gRPC(待完善) | localhost:9090 |
| Neo4j Browser | http://localhost:7474 |
| Grafana | http://localhost:3000 (admin/admin) |
| Prometheus | http://localhost:9091 |

## 项目结构

```
graph-memory/
├── cmd/                    # 应用入口
│   ├── server/            # 服务启动
│   └── cli/               # CLI工具
├── internal/              # 内部包
│   ├── config/            # 配置管理
│   ├── model/             # 数据模型
│   ├── repository/        # 数据访问层
│   └── service/           # 业务逻辑层
├── pkg/                   # 公共包
│   ├── cache/             # 缓存
│   └── logger/            # 日志
├── config/                # 配置文件
├── scripts/               # 脚本
└── docs/                  # 文档
```

## 开发指南

### 构建

```bash
# 下载依赖
make deps

# 编译
make build

# 运行测试
make test
```

### 代码检查

```bash
# 格式化代码
make fmt

# 运行代码检查
make lint
```

### 测试覆盖率

```bash
# 运行测试并生成覆盖率报告
make test-coverage
```

## API 示例

### 创建节点

```bash
curl -X POST http://localhost:8080/api/v1/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "name": "docker-deploy",
    "type": "SKILL",
    "description": "Deploy Docker containers to production"
  }'
```

### 创建关系

```bash
curl -X POST http://localhost:8080/api/v1/edges \
  -H "Content-Type: application/json" \
  -d '{
    "from_id": "node-1",
    "to_id": "node-2",
    "type": "REQUIRES"
  }'
```

### 召回知识

```bash
curl -X POST http://localhost:8080/api/v1/recall \
  -H "Content-Type: application/json" \
  -d '{
    "query": "docker deployment",
    "limit": 10
  }'
```

## 文档

详细文档请参阅 [docs/](./docs/) 目录：

- [架构设计](./docs/01-架构设计.md) - 系统架构、模块设计、数据流
- [API接口](./docs/API.md) - RESTful API 完整文档

## 贡献

欢迎贡献代码！请阅读 [CONTRIBUTING.md](./CONTRIBUTING.md) 了解详情。

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](./LICENSE) 文件。
