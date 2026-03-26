package model

// Subgraph 子图
type Subgraph struct {
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
}

// Path 路径
type Path struct {
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
	Length int    `json:"length"`
}

