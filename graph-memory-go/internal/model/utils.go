package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

// GenerateID 生成唯一ID
func GenerateID(prefix string) string {
	id := uuid.New().String()
	if prefix != "" {
		return fmt.Sprintf("%s-%s", prefix, id[:8])
	}
	return id
}

// ComputeEmbeddingHash 计算向量哈希
func ComputeEmbeddingHash(embedding []float32) string {
	if len(embedding) == 0 {
		return ""
	}

	// 将float32数组转换为字节
	data := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		// 简单的字节转换
		bits := uint32(v * 1000000) // 放大以保留精度
		data[i*4] = byte(bits >> 24)
		data[i*4+1] = byte(bits >> 16)
		data[i*4+2] = byte(bits >> 8)
		data[i*4+3] = byte(bits)
	}

	// 计算SHA256
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])[:16]
}

// CosineSimilarity 计算余弦相似度
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := 0; i < len(a); i++ {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (sqrt(normA) * sqrt(normB))
}

// sqrt 简单的平方根函数
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}

	// 牛顿法求平方根
	z := x
	for i := 0; i < 20; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// ContainsString 检查字符串切片是否包含某字符串
func ContainsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// RemoveString 从字符串切片中移除某字符串
func RemoveString(slice []string, s string) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if v != s {
			result = append(result, v)
		}
	}
	return result
}

// UniqueStrings 去重字符串切片
func UniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
