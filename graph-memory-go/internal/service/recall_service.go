package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/example/graph-memory/internal/llm"
	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/internal/repository"
	"github.com/example/graph-memory/pkg/logger"
)

// RecallService 召回服务 - 实现双路径召回
type RecallService struct {
	nodeRepo   *repository.NodeRepository
	edgeRepo   *repository.EdgeRepository
	graphRepo  *repository.GraphRepository
	vectorRepo *repository.VectorRepository
	llmService *llm.Service
	config     *model.RecallConfig
	log        *logger.Logger
}

// NewRecallService 创建召回服务
func NewRecallService(
	nodeRepo *repository.NodeRepository,
	edgeRepo *repository.EdgeRepository,
	graphRepo *repository.GraphRepository,
	vectorRepo *repository.VectorRepository,
	llmService *llm.Service,
	config *model.RecallConfig,
) *RecallService {
	if config == nil {
		config = model.DefaultRecallConfig()
	}
	config.Validate()

	return &RecallService{
		nodeRepo:   nodeRepo,
		edgeRepo:   edgeRepo,
		graphRepo:  graphRepo,
		vectorRepo: vectorRepo,
		llmService: llmService,
		config:     config,
		log:        logger.NewLogger("info"),
	}
}

// Recall 执行召回
func (s *RecallService) Recall(ctx context.Context, req *model.RecallRequest) (*model.RecallResponse, error) {
	startTime := time.Now()

	// 设置默认值
	if req.Limit <= 0 {
		req.Limit = s.config.DefaultLimit
	}
	if req.PreciseWeight <= 0 {
		req.PreciseWeight = s.config.PreciseWeight
	}
	if req.MaxDepth <= 0 {
		req.MaxDepth = s.config.MaxTraversalDepth
	}

	// 默认启用双路径
	if !req.EnablePrecisePath && !req.EnableGeneralizedPath {
		req.EnablePrecisePath = true
		req.EnableGeneralizedPath = true
	}

	stats := &model.RecallStats{}
	allResults := make(map[string]*model.RecallResult) // 使用map去重

	// 1. 生成查询向量
	vectorStart := time.Now()
	queryVector, err := s.generateQueryVector(ctx, req.Query)
	if err != nil {
		s.log.Warn("Failed to generate query vector", "error", err)
		// 继续执行，使用关键词搜索
	}
	stats.VectorSearchDuration = time.Since(vectorStart).Milliseconds()

	// 2. 精确路径召回
	var preciseResults []*model.RecallResult
	if req.EnablePrecisePath && queryVector != nil {
		graphStart := time.Now()
		preciseResults, err = s.precisePathRecall(ctx, req, queryVector)
		if err != nil {
			s.log.Warn("Precise path recall failed", "error", err)
		}
		stats.GraphTraversalDuration += time.Since(graphStart).Milliseconds()
		stats.PreciseCount = len(preciseResults)

		// 合并结果
		for _, r := range preciseResults {
			r.Path = string(model.RecallPathPrecise)
			r.Score = r.Score * req.PreciseWeight
			if existing, ok := allResults[r.Node.ID]; ok {
				// 合并分数
				existing.Score = existing.Score + r.Score*(1-req.PreciseWeight)
				if existing.MatchReason == "" {
					existing.MatchReason = r.MatchReason
				}
			} else {
				allResults[r.Node.ID] = r
			}
		}
	}

	// 3. 泛化路径召回
	var generalizedResults []*model.RecallResult
	if req.EnableGeneralizedPath && queryVector != nil {
		graphStart := time.Now()
		generalizedResults, err = s.generalizedPathRecall(ctx, req, queryVector, preciseResults)
		if err != nil {
			s.log.Warn("Generalized path recall failed", "error", err)
		}
		stats.GraphTraversalDuration += time.Since(graphStart).Milliseconds()
		stats.GeneralizedCount = len(generalizedResults)

		// 合并结果
		generalizedWeight := 1 - req.PreciseWeight
		for _, r := range generalizedResults {
			r.Path = string(model.RecallPathGeneralized)
			r.Score = r.Score * generalizedWeight
			if existing, ok := allResults[r.Node.ID]; ok {
				// 合并分数
				existing.Score = existing.Score + r.Score
				if existing.MatchReason == "" {
					existing.MatchReason = r.MatchReason
				}
			} else {
				allResults[r.Node.ID] = r
			}
		}
	}

	// 4. 如果没有向量，使用关键词搜索
	if len(allResults) == 0 {
		keywordResults, err := s.keywordRecall(ctx, req)
		if err != nil {
			s.log.Warn("Keyword recall failed", "error", err)
		}
		for _, r := range keywordResults {
			allResults[r.Node.ID] = r
		}
	}

	// 5. 排序并截取
	results := s.sortAndLimitResults(allResults, req.Limit)
	stats.DeduplicatedCount = len(results)

	// 6. Token估算
	tokenEstimate := s.estimateTokens(results)

	// 7. 构建响应
	duration := time.Since(startTime).Milliseconds()
	return &model.RecallResponse{
		Results:       results,
		Total:         len(results),
		Duration:      duration,
		TokenEstimate: tokenEstimate,
		Stats:         stats,
	}, nil
}

// generateQueryVector 生成查询向量
func (s *RecallService) generateQueryVector(ctx context.Context, query string) ([]float64, error) {
	if s.llmService == nil {
		return nil, fmt.Errorf("LLM service not available")
	}

	embedding, err := s.llmService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	return embedding, nil
}

// precisePathRecall 精确路径召回
// 直接通过向量相似度匹配查询中的实体
func (s *RecallService) precisePathRecall(ctx context.Context, req *model.RecallRequest, queryVector []float64) ([]*model.RecallResult, error) {
	// 1. 向量搜索获取直接匹配的节点
	vectorReq := &model.VectorSearchRequest{
		Vector:   queryVector,
		Limit:    req.Limit * 2, // 获取更多候选
		MinScore: s.config.MinVectorSimilarity,
		Filters:  req.Filters,
	}

	matchedNodes, err := s.vectorRepo.VectorSearchWithNodes(ctx, vectorReq)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	results := make([]*model.RecallResult, 0, len(matchedNodes))

	// 2. 为每个匹配节点获取直接邻居
	for _, node := range matchedNodes {
		result := &model.RecallResult{
			Node:        node,
			Score:       node.PageRank, // 向量搜索分数存储在PageRank字段
			MatchReason: "向量相似度匹配",
			ContextPath: []*model.Node{node},
		}

		// 获取直接邻居
		neighbors, err := s.graphRepo.GetNeighbors(ctx, node.ID, 1)
		if err == nil && len(neighbors) > 0 {
			// 限制邻居数量
			if len(neighbors) > 5 {
				neighbors = neighbors[:5]
			}
			result.Neighbors = neighbors
		}

		results = append(results, result)
	}

	return results, nil
}

// generalizedPathRecall 泛化路径召回
// 通过图遍历扩展相关节点
func (s *RecallService) generalizedPathRecall(ctx context.Context, req *model.RecallRequest, queryVector []float64, preciseResults []*model.RecallResult) ([]*model.RecallResult, error) {
	results := make([]*model.RecallResult, 0)

	// 1. 收集精确路径召回的节点ID作为起点
	seedNodeIDs := make([]string, 0)
	for _, r := range preciseResults {
		seedNodeIDs = append(seedNodeIDs, r.Node.ID)
	}

	if len(seedNodeIDs) == 0 {
		return results, nil
	}

	// 2. 图遍历扩展
	visited := make(map[string]bool)
	for _, id := range seedNodeIDs {
		visited[id] = true
	}

	// BFS遍历
	currentLevel := seedNodeIDs
	for depth := 1; depth <= req.MaxDepth; depth++ {
		nextLevel := make([]string, 0)

		for _, nodeID := range currentLevel {
			// 获取邻居
			neighbors, err := s.graphRepo.GetNeighbors(ctx, nodeID, 1)
			if err != nil {
				continue
			}

			for _, neighbor := range neighbors {
				if visited[neighbor.ID] {
					continue
				}
				visited[neighbor.ID] = true

				// 计算分数 (基于深度衰减)
				score := 1.0 / float64(depth+1)

				// 如果有向量，计算与查询的相似度
				if len(neighbor.Embedding) > 0 && len(queryVector) > 0 {
					embedding := make([]float32, len(queryVector))
					for i, v := range queryVector {
						embedding[i] = float32(v)
					}
					similarity := model.CosineSimilarity(embedding, neighbor.Embedding)
					score = score*0.5 + similarity*0.5
				}

				// 检查过滤条件
				if !s.matchesFilters(neighbor, req.Filters) {
					continue
				}

				results = append(results, &model.RecallResult{
					Node:        neighbor,
					Score:       score,
					MatchReason: fmt.Sprintf("通过图遍历发现 (深度%d)", depth),
				})

				nextLevel = append(nextLevel, neighbor.ID)
			}
		}

		currentLevel = nextLevel
		if len(currentLevel) == 0 {
			break
		}
	}

	// 3. 社区扩展
	if s.config.EnableCommunityExpansion {
		communityResults := s.communityExpansion(ctx, seedNodeIDs, visited, req.Filters)
		results = append(results, communityResults...)
	}

	return results, nil
}

// communityExpansion 社区扩展
func (s *RecallService) communityExpansion(ctx context.Context, seedNodeIDs []string, visited map[string]bool, filters *model.RecallFilters) []*model.RecallResult {
	results := make([]*model.RecallResult, 0)

	// 获取种子节点的社区ID
	communityIDs := make(map[string]bool)
	for _, nodeID := range seedNodeIDs {
		node, err := s.nodeRepo.FindByID(ctx, nodeID)
		if err == nil && node.CommunityID != "" {
			communityIDs[node.CommunityID] = true
		}
	}

	// 获取同社区的高PageRank节点
	for communityID := range communityIDs {
		nodes, err := s.vectorRepo.GetNodesByCommunity(ctx, communityID)
		if err != nil {
			continue
		}

		for _, node := range nodes {
			if visited[node.ID] {
				continue
			}
			visited[node.ID] = true

			// 检查过滤条件
			if !s.matchesFilters(node, filters) {
				continue
			}

			// 分数基于PageRank
			score := 0.3
			if node.PageRank > 0 {
				score = 0.3 + node.PageRank*0.7
			}

			results = append(results, &model.RecallResult{
				Node:        node,
				Score:       score,
				MatchReason: "同社区关联节点",
			})
		}
	}

	return results
}

// keywordRecall 关键词召回 (当向量不可用时的备选方案)
func (s *RecallService) keywordRecall(ctx context.Context, req *model.RecallRequest) ([]*model.RecallResult, error) {
	// 使用关键词搜索
	nodes, err := s.nodeRepo.Search(ctx, req.Query, req.Limit*2)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}

	results := make([]*model.RecallResult, 0, len(nodes))
	for _, node := range nodes {
		// 计算简单的关键词匹配分数
		score := 0.5
		if strings.Contains(strings.ToLower(node.Name), strings.ToLower(req.Query)) {
			score = 0.8
		} else if strings.Contains(strings.ToLower(node.Description), strings.ToLower(req.Query)) {
			score = 0.6
		}

		// 检查过滤条件
		if !s.matchesFilters(node, req.Filters) {
			continue
		}

		results = append(results, &model.RecallResult{
			Node:        node,
			Score:       score,
			MatchReason: "关键词匹配",
		})
	}

	return results, nil
}

// matchesFilters 检查节点是否匹配过滤条件
func (s *RecallService) matchesFilters(node *model.Node, filters *model.RecallFilters) bool {
	if filters == nil {
		return true
	}

	// 检查节点类型
	if len(filters.NodeTypes) > 0 {
		found := false
		for _, t := range filters.NodeTypes {
			if node.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查状态
	if filters.Status != "" && node.Status != filters.Status {
		return false
	}

	// 检查社区ID
	if filters.CommunityID != "" && node.CommunityID != filters.CommunityID {
		return false
	}

	// 检查PageRank
	if filters.MinPageRank > 0 && node.PageRank < filters.MinPageRank {
		return false
	}

	return true
}

// sortAndLimitResults 排序并限制结果数量
func (s *RecallService) sortAndLimitResults(results map[string]*model.RecallResult, limit int) []*model.RecallResult {
	// 转换为切片
	slice := make([]*model.RecallResult, 0, len(results))
	for _, r := range results {
		slice = append(slice, r)
	}

	// 按分数排序 (降序)
	for i := 0; i < len(slice)-1; i++ {
		for j := i + 1; j < len(slice); j++ {
			if slice[i].Score < slice[j].Score {
				slice[i], slice[j] = slice[j], slice[i]
			}
		}
	}

	// 限制数量
	if len(slice) > limit {
		slice = slice[:limit]
	}

	return slice
}

// estimateTokens 估算Token数量
func (s *RecallService) estimateTokens(results []*model.RecallResult) *model.TokenEstimate {
	config := s.config.TokenEstimator
	if config == nil {
		config = model.DefaultTokenEstimatorConfig()
	}

	totalTokens := 0
	nodeTokens := 0
	contextTokens := 0
	relationTokens := 0

	for _, r := range results {
		// 节点描述Token
		nodeDesc := fmt.Sprintf("[%s] %s: %s", r.Node.Type, r.Node.Name, r.Node.Description)
		nodeTokenCount := int(float64(len(nodeDesc)) * config.CharsPerToken)
		nodeTokens += nodeTokenCount
		totalTokens += nodeTokenCount

		// 上下文路径Token
		for _, ctxNode := range r.ContextPath {
			ctxDesc := fmt.Sprintf("%s -> %s", ctxNode.Name, r.Node.Name)
			ctxTokenCount := int(float64(len(ctxDesc)) * config.CharsPerToken)
			contextTokens += ctxTokenCount
			totalTokens += ctxTokenCount
		}

		// 邻居节点关系Token
		for _, neighbor := range r.Neighbors {
			relDesc := fmt.Sprintf("%s 关联 %s", r.Node.Name, neighbor.Name)
			relTokenCount := int(float64(len(relDesc)) * config.CharsPerToken)
			relationTokens += relTokenCount
			totalTokens += relTokenCount
		}
	}

	return &model.TokenEstimate{
		TotalTokens:    totalTokens,
		NodeTokens:     nodeTokens,
		ContextTokens:  contextTokens,
		RelationTokens: relationTokens,
		ExceedsLimit:   totalTokens > config.TokenLimit,
		TokenLimit:     config.TokenLimit,
	}
}

// BuildContext 构建上下文
func (s *RecallService) BuildContext(ctx context.Context, req *model.ContextBuildRequest) (*model.ContextBuilder, error) {
	if len(req.Results) == 0 {
		return &model.ContextBuilder{
			Context:       "",
			Nodes:         []*model.Node{},
			Edges:         []*model.Edge{},
			TokenEstimate: &model.TokenEstimate{},
			BuildTime:     time.Now().UTC(),
		}, nil
	}

	// 设置默认值
	if req.MaxTokens <= 0 {
		req.MaxTokens = s.config.TokenEstimator.TokenLimit
	}
	if req.Format == "" {
		req.Format = "text"
	}

	var builder strings.Builder
	nodes := make([]*model.Node, 0)
	edges := make([]*model.Edge, 0)
	totalTokens := 0

	// 根据格式构建上下文
	switch req.Format {
	case "json":
		builder.WriteString("{\n  \"nodes\": [\n")
		for i, r := range req.Results {
			if totalTokens > req.MaxTokens {
				break
			}
			nodeJSON := fmt.Sprintf("    {\"name\": \"%s\", \"type\": \"%s\", \"description\": \"%s\"}",
				r.Node.Name, r.Node.Type, r.Node.Description)
			if i < len(req.Results)-1 {
				nodeJSON += ","
			}
			builder.WriteString(nodeJSON + "\n")
			nodes = append(nodes, r.Node)
			totalTokens += len(nodeJSON) / 2 // 粗略估算
		}
		builder.WriteString("  ]\n}")

	case "markdown":
		builder.WriteString("# 知识上下文\n\n")
		for _, r := range req.Results {
			if totalTokens > req.MaxTokens {
				break
			}
			section := fmt.Sprintf("## %s (%s)\n%s\n\n", r.Node.Name, r.Node.Type, r.Node.Description)
			builder.WriteString(section)
			nodes = append(nodes, r.Node)
			totalTokens += len(section) / 2
		}

	default: // text
		builder.WriteString("相关知识点：\n\n")
		for i, r := range req.Results {
			if totalTokens > req.MaxTokens {
				break
			}
			line := fmt.Sprintf("%d. [%s] %s: %s\n", i+1, r.Node.Type, r.Node.Name, r.Node.Description)
			builder.WriteString(line)
			nodes = append(nodes, r.Node)
			totalTokens += len(line) / 2

			// 添加邻居信息
			if req.IncludeNeighbors && len(r.Neighbors) > 0 {
				builder.WriteString("   相关节点：")
				for j, n := range r.Neighbors {
					if j > 0 {
						builder.WriteString(", ")
					}
					builder.WriteString(n.Name)
				}
				builder.WriteString("\n")
			}
		}
	}

	return &model.ContextBuilder{
		Context:       builder.String(),
		Nodes:         nodes,
		Edges:         edges,
		TokenEstimate: &model.TokenEstimate{TotalTokens: totalTokens},
		BuildTime:     time.Now(),
	}, nil
}

// GetRecallConfig 获取召回配置
func (s *RecallService) GetRecallConfig() *model.RecallConfig {
	return s.config
}

// SetRecallConfig 设置召回配置
func (s *RecallService) SetRecallConfig(config *model.RecallConfig) {
	config.Validate()
	s.config = config
}
