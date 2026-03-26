package llm

import (
	"encoding/json"
	"regexp"
)

// ==================== JSON辅助函数 ====================

// jsonUnmarshal 安全的JSON解析
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// jsonMarshal JSON序列化
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// extractJSON 从文本中提取JSON内容
// 支持 markdown 代码块和原始JSON
func extractJSON(content string) string {
	// 1. 尝试提取 markdown 代码块中的 JSON
	codeBlockRegex := regexp.MustCompile("```(?:json)?\\s*\\n?([\\s\\S]*?)\\n?```")
	if matches := codeBlockRegex.FindStringSubmatch(content); len(matches) > 1 {
		return matches[1]
	}

	// 2. 尝试提取原始 JSON 对象
	jsonObjRegex := regexp.MustCompile("\\{[\\s\\S]*\\}")
	if match := jsonObjRegex.FindString(content); match != "" {
		return match
	}

	// 3. 返回原始内容
	return content
}

// ==================== 类型转换辅助函数 ====================

// triplesToPtr 转换三元组切片为指针切片
func triplesToPtr(triples []Triple) []*Triple {
	result := make([]*Triple, len(triples))
	for i := range triples {
		result[i] = &triples[i]
	}
	return result
}

// entitiesToPtr 转换实体切片为指针切片
func entitiesToPtr(entities []ExtractedEntity) []*ExtractedEntity {
	result := make([]*ExtractedEntity, len(entities))
	for i := range entities {
		result[i] = &entities[i]
	}
	return result
}

// ptrEntitiesToVal 转换实体指针切片为值切片
func ptrEntitiesToVal(entities []*ExtractedEntity) []ExtractedEntity {
	result := make([]ExtractedEntity, len(entities))
	for i, e := range entities {
		if e != nil {
			result[i] = *e
		}
	}
	return result
}

// ptrTriplesToVal 转换三元组指针切片为值切片
func ptrTriplesToVal(triples []*Triple) []Triple {
	result := make([]Triple, len(triples))
	for i, t := range triples {
		if t != nil {
			result[i] = *t
		}
	}
	return result
}

// float32To64 float32转float64
func float32To64(slice []float32) []float64 {
	result := make([]float64, len(slice))
	for i, v := range slice {
		result[i] = float64(v)
	}
	return result
}

// float64To32 float64转float32
func float64To32(slice []float64) []float32 {
	result := make([]float32, len(slice))
	for i, v := range slice {
		result[i] = float32(v)
	}
	return result
}

// float64To32Slice 2D float64转float32
func float64To32Slice(slice [][]float64) [][]float32 {
	result := make([][]float32, len(slice))
	for i, s := range slice {
		result[i] = float64To32(s)
	}
	return result
}
