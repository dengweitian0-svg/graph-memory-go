package model

// NodeQueryBuilder Node查询构建器
type NodeQueryBuilder struct {
	filters map[string]interface{}
	limit   int
	offset  int
	orderBy string
	order   string // "ASC" or "DESC"
}

// NewNodeQueryBuilder 创建Node查询构建器
func NewNodeQueryBuilder() *NodeQueryBuilder {
	return &NodeQueryBuilder{
		filters: make(map[string]interface{}),
		limit:   100,
		offset:  0,
		orderBy: "created_at",
		order:   "DESC",
	}
}

// ByType 按类型过滤
func (b *NodeQueryBuilder) ByType(nodeType NodeType) *NodeQueryBuilder {
	b.filters["type"] = string(nodeType)
	return b
}

// ByStatus 按状态过滤
func (b *NodeQueryBuilder) ByStatus(status NodeStatus) *NodeQueryBuilder {
	b.filters["status"] = string(status)
	return b
}

// ByCommunityID 按社区ID过滤
func (b *NodeQueryBuilder) ByCommunityID(communityID string) *NodeQueryBuilder {
	b.filters["community_id"] = communityID
	return b
}

// ByName 按名称模糊匹配
func (b *NodeQueryBuilder) ByName(name string) *NodeQueryBuilder {
	b.filters["name"] = name
	return b
}

// WithLimit 设置限制
func (b *NodeQueryBuilder) WithLimit(limit int) *NodeQueryBuilder {
	b.limit = limit
	return b
}

// WithOffset 设置偏移
func (b *NodeQueryBuilder) WithOffset(offset int) *NodeQueryBuilder {
	b.offset = offset
	return b
}

// OrderByPageRank 按PageRank排序
func (b *NodeQueryBuilder) OrderByPageRank(desc bool) *NodeQueryBuilder {
	b.orderBy = "pagerank"
	if desc {
		b.order = "DESC"
	} else {
		b.order = "ASC"
	}
	return b
}

// OrderByCreatedAt 按创建时间排序
func (b *NodeQueryBuilder) OrderByCreatedAt(desc bool) *NodeQueryBuilder {
	b.orderBy = "created_at"
	if desc {
		b.order = "DESC"
	} else {
		b.order = "ASC"
	}
	return b
}

// Build 构建查询参数
func (b *NodeQueryBuilder) Build() map[string]interface{} {
	params := make(map[string]interface{})
	for k, v := range b.filters {
		params[k] = v
	}
	params["limit"] = b.limit
	params["offset"] = b.offset
	params["order_by"] = b.orderBy
	params["order"] = b.order
	return params
}

// EdgeQueryBuilder Edge查询构建器
type EdgeQueryBuilder struct {
	fromID  string
	toID    string
	edgeType EdgeType
	limit   int
}

// NewEdgeQueryBuilder 创建Edge查询构建器
func NewEdgeQueryBuilder() *EdgeQueryBuilder {
	return &EdgeQueryBuilder{
		limit: 100,
	}
}

// FromNode 设置起始节点
func (b *EdgeQueryBuilder) FromNode(nodeID string) *EdgeQueryBuilder {
	b.fromID = nodeID
	return b
}

// ToNode 设置目标节点
func (b *EdgeQueryBuilder) ToNode(nodeID string) *EdgeQueryBuilder {
	b.toID = nodeID
	return b
}

// ByType 按类型过滤
func (b *EdgeQueryBuilder) ByType(edgeType EdgeType) *EdgeQueryBuilder {
	b.edgeType = edgeType
	return b
}

// WithLimit 设置限制
func (b *EdgeQueryBuilder) WithLimit(limit int) *EdgeQueryBuilder {
	b.limit = limit
	return b
}

// Build 构建查询参数
func (b *EdgeQueryBuilder) Build() map[string]interface{} {
	params := make(map[string]interface{})
	if b.fromID != "" {
		params["from_id"] = b.fromID
	}
	if b.toID != "" {
		params["to_id"] = b.toID
	}
	if b.edgeType != "" {
		params["type"] = string(b.edgeType)
	}
	params["limit"] = b.limit
	return params
}

// GraphTraversalQuery 图遍历查询
type GraphTraversalQuery struct {
	StartNodeID string
	Direction   string // "outgoing", "incoming", "both"
	MaxDepth    int
	EdgeTypes   []EdgeType
	Limit       int
}

// NewGraphTraversalQuery 创建图遍历查询
func NewGraphTraversalQuery(startNodeID string) *GraphTraversalQuery {
	return &GraphTraversalQuery{
		StartNodeID: startNodeID,
		Direction:   "outgoing",
		MaxDepth:    3,
		EdgeTypes:   []EdgeType{},
		Limit:       100,
	}
}

// Outgoing 设置出边方向
func (q *GraphTraversalQuery) Outgoing() *GraphTraversalQuery {
	q.Direction = "outgoing"
	return q
}

// Incoming 设置入边方向
func (q *GraphTraversalQuery) Incoming() *GraphTraversalQuery {
	q.Direction = "incoming"
	return q
}

// Both 设置双向
func (q *GraphTraversalQuery) Both() *GraphTraversalQuery {
	q.Direction = "both"
	return q
}

// WithMaxDepth 设置最大深度
func (q *GraphTraversalQuery) WithMaxDepth(depth int) *GraphTraversalQuery {
	q.MaxDepth = depth
	return q
}

// WithEdgeTypes 设置边类型
func (q *GraphTraversalQuery) WithEdgeTypes(types ...EdgeType) *GraphTraversalQuery {
	q.EdgeTypes = types
	return q
}

// WithLimit 设置限制
func (q *GraphTraversalQuery) WithLimit(limit int) *GraphTraversalQuery {
	q.Limit = limit
	return q
}
