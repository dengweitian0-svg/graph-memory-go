package workflow

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/example/graph-memory/internal/model"
	"github.com/example/graph-memory/pkg/logger"
)

// KnowledgeExtractor 知识提取器
type KnowledgeExtractor struct {
	log *logger.Logger
}

// NewKnowledgeExtractor 创建知识提取器
func NewKnowledgeExtractor() *KnowledgeExtractor {
	return &KnowledgeExtractor{
		log: logger.NewLogger("info"),
	}
}

// ExtractionResult 提取结果
type ExtractionResult struct {
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
func (e *KnowledgeExtractor) ExtractFromMessages(ctx context.Context, messages []*model.Message) (*ExtractionResult, error) {
	e.log.Debug("Extracting knowledge from messages", "count", len(messages))

	result := &ExtractionResult{
		Nodes: make([]*ExtractedNode, 0),
		Edges: make([]*ExtractedEdge, 0),
	}

	// 合并所有消息内容
	var content strings.Builder
	for _, msg := range messages {
		if msg.Role == model.MessageRoleUser || msg.Role == model.MessageRoleAssistant {
			content.WriteString(msg.Content)
			content.WriteString(" ")
		}
	}

	text := content.String()

	// 提取实体（简化实现）
	nodes := e.extractEntities(text)
	result.Nodes = append(result.Nodes, nodes...)

	// 提取关系
	edges := e.extractRelations(text, nodes)
	result.Edges = append(result.Edges, edges...)

	e.log.Info("Knowledge extraction completed",
		"nodes", len(result.Nodes),
		"edges", len(result.Edges),
	)

	return result, nil
}

// extractEntities 提取实体
func (e *KnowledgeExtractor) extractEntities(text string) []*ExtractedNode {
	nodes := make([]*ExtractedNode, 0)
	seen := make(map[string]bool)

	// 技能关键词
	skillPatterns := []string{
		`(?i)(\w+)\s+(?:编程|开发|框架|语言|技术)`,
		`(?i)(?:学习|掌握|使用|应用)\s+(\w+)`,
		`(?i)(\w+)\s+(?:技能|能力)`,
	}

	for _, pattern := range skillPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				name := strings.TrimSpace(match[1])
				if name != "" && len(name) > 1 && !seen[name] {
					seen[name] = true
					nodes = append(nodes, &ExtractedNode{
						Name:        name,
						Type:        model.NodeTypeSkill,
						Description: fmt.Sprintf("%s相关技能", name),
						Metadata:    map[string]interface{}{"source": "extracted"},
					})
				}
			}
		}
	}

	// 任务关键词
	taskPatterns := []string{
		`(?i)(\w+)\s+(?:任务|工作|项目)`,
		`(?i)(?:完成|实现|开发)\s+(\w+)`,
	}

	for _, pattern := range taskPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				name := strings.TrimSpace(match[1])
				if name != "" && len(name) > 1 && !seen[name] {
					seen[name] = true
					nodes = append(nodes, &ExtractedNode{
						Name:        name,
						Type:        model.NodeTypeTask,
						Description: fmt.Sprintf("%s相关任务", name),
						Metadata:    map[string]interface{}{"source": "extracted"},
					})
				}
			}
		}
	}

	// 概念关键词
	conceptPatterns := []string{
		`(?i)(\w+)\s+(?:概念|理论|原理|方法)`,
		`(?i)(?:理解|掌握)\s+(\w+)`,
	}

	for _, pattern := range conceptPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				name := strings.TrimSpace(match[1])
				if name != "" && len(name) > 1 && !seen[name] {
					seen[name] = true
					nodes = append(nodes, &ExtractedNode{
						Name:        name,
						Type:        model.NodeTypeConcept,
						Description: fmt.Sprintf("%s相关概念", name),
						Metadata:    map[string]interface{}{"source": "extracted"},
					})
				}
			}
		}
	}

	return nodes
}

// extractRelations 提取关系
func (e *KnowledgeExtractor) extractRelations(text string, nodes []*ExtractedNode) []*ExtractedEdge {
	edges := make([]*ExtractedEdge, 0)

	// 创建节点名称索引
	nodeNames := make(map[string]bool)
	for _, node := range nodes {
		nodeNames[node.Name] = true
	}

	// 关系模式
	relationPatterns := []struct {
		pattern string
		relType model.EdgeType
	}{
		{`(?i)(\w+)\s+(?:需要|依赖)\s+(\w+)`, model.EdgeTypeRequires},
		{`(?i)(\w+)\s+(?:触发|导致)\s+(\w+)`, model.EdgeTypeTriggered},
		{`(?i)(\w+)\s+(?:解决|处理)\s+(\w+)`, model.EdgeTypeSolves},
		{`(?i)(\w+)\s+(?:相关|类似)\s+(\w+)`, model.EdgeTypeRelated},
		{`(?i)(\w+)\s+(?:包含|包括)\s+(\w+)`, model.EdgeTypeContains},
		{`(?i)(\w+)\s+(?:依赖|基于)\s+(\w+)`, model.EdgeTypeDependsOn},
	}

	for _, rp := range relationPatterns {
		re := regexp.MustCompile(rp.pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 2 {
				fromName := strings.TrimSpace(match[1])
				toName := strings.TrimSpace(match[2])

				// 检查节点是否存在
				if nodeNames[fromName] || nodeNames[toName] {
					edges = append(edges, &ExtractedEdge{
						FromName: fromName,
						ToName:   toName,
						Type:     rp.relType,
						Weight:   1.0,
						Metadata: map[string]interface{}{"source": "extracted"},
					})
				}
			}
		}
	}

	return edges
}
