package llm

import (
	"context"
	"fmt"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/pkg/logger"
)

// EnhancedExtractor 增强版知识提取器（使用LLM）
type EnhancedExtractor struct {
	llmService *Service
	log        *logger.Logger
}

// NewEnhancedExtractor 创建增强版提取器
func NewEnhancedExtractor(llmService *Service) *EnhancedExtractor {
	return &EnhancedExtractor{
		llmService: llmService,
		log:        logger.NewLogger("info"),
	}
}

// GraphExtractionResult 图提取结果（用于工作流）
type GraphExtractionResult struct {
	Nodes []*ExtractedNode `json:"nodes"`
	Edges []*ExtractedEdge `json:"edges"`
}

// ExtractedNode 提取的节点
type ExtractedNode struct {
	Name        string                 `json:"name"`
	Type        model.NodeType         `json:"type"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ExtractedEdge 提取的边
type ExtractedEdge struct {
	FromName string                 `json:"from_name"`
	ToName   string                 `json:"to_name"`
	Type     model.EdgeType         `json:"type"`
	Weight   float64                `json:"weight"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ExtractFromMessages 从消息中提取知识
func (e *EnhancedExtractor) ExtractFromMessages(ctx context.Context, messages []*model.Message) (*GraphExtractionResult, error) {
	e.log.Debug("Extracting knowledge from messages using LLM", "count", len(messages))

	// 合并消息内容
	var content string
	for _, msg := range messages {
		if msg.Role == model.MessageRoleUser || msg.Role == model.MessageRoleAssistant {
			content += msg.Content + " "
		}
	}

	if content == "" {
		return &GraphExtractionResult{
			Nodes: make([]*ExtractedNode, 0),
			Edges: make([]*ExtractedEdge, 0),
		}, nil
	}

	// 使用LLM提取知识
	knowledge, err := e.llmService.ExtractKnowledgeFromText(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("extract knowledge: %w", err)
	}

	// 转换为节点和边
	nodes := e.convertEntitiesToNodes(knowledge.Entities)
	edges := e.convertTriplesToEdges(knowledge.Triples, nodes)

	e.log.Info("LLM extraction completed",
		"nodes", len(nodes),
		"edges", len(edges),
	)

	return &GraphExtractionResult{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// ExtractFromText 从文本中提取知识
func (e *EnhancedExtractor) ExtractFromText(ctx context.Context, text string) (*GraphExtractionResult, error) {
	e.log.Debug("Extracting knowledge from text using LLM", "length", len(text))

	knowledge, err := e.llmService.ExtractKnowledgeFromText(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("extract knowledge: %w", err)
	}

	nodes := e.convertEntitiesToNodes(knowledge.Entities)
	edges := e.convertTriplesToEdges(knowledge.Triples, nodes)

	return &GraphExtractionResult{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// convertEntitiesToNodes 将实体转换为节点
func (e *EnhancedExtractor) convertEntitiesToNodes(entities []*ExtractedEntity) []*ExtractedNode {
	nodes := make([]*ExtractedNode, 0, len(entities))
	seen := make(map[string]bool)

	for _, entity := range entities {
		key := entity.Name + "_" + entity.Type
		if seen[key] {
			continue
		}
		seen[key] = true

		nodeType := model.NodeTypeConcept
		switch entity.Type {
		case "TASK":
			nodeType = model.NodeTypeTask
		case "SKILL":
			nodeType = model.NodeTypeSkill
		case "EVENT":
			nodeType = model.NodeTypeEvent
		}

		nodes = append(nodes, &ExtractedNode{
			Name:        entity.Name,
			Type:        nodeType,
			Description: entity.Description,
			Metadata: map[string]interface{}{
				"source":     "llm_extraction",
				"confidence": entity.Confidence,
			},
		})
	}

	return nodes
}

// convertTriplesToEdges 将三元组转换为边
func (e *EnhancedExtractor) convertTriplesToEdges(triples []*Triple, nodes []*ExtractedNode) []*ExtractedEdge {
	// 创建节点名称索引
	nodeNames := make(map[string]bool)
	for _, node := range nodes {
		nodeNames[node.Name] = true
	}

	edges := make([]*ExtractedEdge, 0, len(triples))

	for _, triple := range triples {
		// 只添加两个实体都存在的边
		if !nodeNames[triple.Subject] && !nodeNames[triple.Object] {
			continue
		}

		edgeType := e.mapPredicateToEdgeType(triple.Predicate)

		edges = append(edges, &ExtractedEdge{
			FromName: triple.Subject,
			ToName:   triple.Object,
			Type:     edgeType,
			Weight:   triple.Confidence,
			Metadata: map[string]interface{}{
				"predicate":  triple.Predicate,
				"confidence": triple.Confidence,
				"source":     "llm_extraction",
			},
		})
	}

	return edges
}

// mapPredicateToEdgeType 将谓词映射到边类型
func (e *EnhancedExtractor) mapPredicateToEdgeType(predicate string) model.EdgeType {
	switch predicate {
	case "需要", "requires", "依赖", "depends_on":
		return model.EdgeTypeRequires
	case "触发", "triggers", "导致", "causes":
		return model.EdgeTypeTriggered
	case "解决", "solves", "处理", "handles":
		return model.EdgeTypeSolves
	case "包含", "contains", "包括", "includes":
		return model.EdgeTypeContains
	case "跟随", "follows", "顺序", "sequenced":
		return model.EdgeTypeFollows
	default:
		return model.EdgeTypeRelated
	}
}
