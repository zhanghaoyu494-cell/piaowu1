package embedding

import (
	"context"
	"fmt"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// Client 封装了火山引擎（Ark）提供的 Embedding API 调用能力
// 它是将“非结构化文本”转化为“计算机可理解向量”的桥梁
type Client struct {
	arkClient *arkruntime.Client // 火山引擎运行时 SDK 客户端
	model     string             // 使用的向量化模型 ID (例如: doubao-embedding-v1)
}

// NewClient 根据 APIKey 构造并初始化 Embedding 客户端
func NewClient(apiKey, modelName string) *Client {
	client := arkruntime.NewClientWithApiKey(apiKey)
	return &Client{
		arkClient: client,
		model:     modelName,
	}
}

// Embed 将单条文本字符串转变为对应的浮点向量
// 实现细节：内部调用了火山引擎的多模态文本向量化接口
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	// 构造多模态请求结构 (目前仅使用其中的文本类型)
	textPtr := text
	input := model.MultimodalEmbeddingInput{
		Type: model.MultiModalEmbeddingInputTypeText,
		Text: &textPtr,
	}

	req := model.MultiModalEmbeddingRequest{
		Model: c.model,
		Input: []model.MultimodalEmbeddingInput{input},
	}

	// 阻塞发起 API 调用，将文本点投影至高维空间
	resp, err := c.arkClient.CreateMultiModalEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("向量化请求解析异常: %w", err)
	}

	// 解析响应体中的向量数据
	embedding := resp.Data.Embedding
	if len(embedding) == 0 {
		return nil, fmt.Errorf("模型返回的向量内容为空，请检查模型可用性")
	}

	return embedding, nil
}

// EmbedBatch 针对多条文本执行批量初始化向量化处理
// 流程：通过循环遍历执行串行化请求 (火山引擎多模态 API 通常为单条或受限批量)
func (c *Client) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	result := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := c.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("批量处理在第 %d 条处失败: %w", i, err)
		}
		result[i] = vec
	}
	return result, nil
}

// GetDimension 探测当前模型生成的向量维度，用于初始化向量库集合
func (c *Client) GetDimension(ctx context.Context) (int, error) {
	// 通过一次测试调用来动态确认维度长度 (通常为 1024 或 2048)
	vec, err := c.Embed(ctx, "系统探测预检文本")
	if err != nil {
		return 0, err
	}
	return len(vec), nil
}
