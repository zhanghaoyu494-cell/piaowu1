package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// MilvusClient 封装了高效的 Milvus 向量数据库操作
// 它是 RAG (检索增强生成) 流程中的关键组件
type MilvusClient struct {
	client     client.Client // 原始 Milvus SDK 客户端
	collection string        // 处理的集合名称
	dimension  int           // 向量维度 (需与 Embedding 模型对齐)
}

// SearchResult 描述了从向量库检索出的原始背景数据块
type SearchResult struct {
	ChunkID int64   `json:"chunk_id"` // 数据块唯一 ID
	Text    string  `json:"text"`     // 知识点文本内容
	Source  string  `json:"source"`   // 知识来源 (如文件名、URL)
	Score   float32 `json:"score"`    // 相似度得分 (L2 距离)
}

// NewMilvusClient 创建并验证 Milvus 数据库的长连接
func NewMilvusClient(host string, port int, collection string, dimension int) (*MilvusClient, error) {
	ctx := context.Background()
	address := fmt.Sprintf("%s:%d", host, port)

	// 初始化客户端配置
	c, err := client.NewClient(ctx, client.Config{
		Address: address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to milvus at %s: %w", address, err)
	}

	log.Printf("✅ 已建立 Milvus 向量数据库连接: %s", address)

	return &MilvusClient{
		client:     c,
		collection: collection,
		dimension:  dimension,
	}, nil
}

// EnsureCollection 幂等性检查 Collection 状态，若缺失则自动按预设字段定义创建
func (m *MilvusClient) EnsureCollection(ctx context.Context) error {
	exists, err := m.client.HasCollection(ctx, m.collection)
	if err != nil {
		return fmt.Errorf("check collection exists failed: %w", err)
	}

	if exists {
		log.Printf("Collection '%s' 已存在，跳过创建步骤", m.collection)
		return nil
	}

	// 定义 Schema：主键(Int64) + 文本块(VarChar) + 来源(VarChar) + 向量(FloatVector)
	schema := entity.NewSchema().WithName(m.collection).WithDescription("智能客服知识库").
		WithField(entity.NewField().WithName("chunk_id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true).WithIsAutoID(true)).
		WithField(entity.NewField().WithName("text").WithDataType(entity.FieldTypeVarChar).WithMaxLength(65535)).
		WithField(entity.NewField().WithName("source").WithDataType(entity.FieldTypeVarChar).WithMaxLength(512)).
		WithField(entity.NewField().WithName("embedding").WithDataType(entity.FieldTypeFloatVector).WithDim(int64(m.dimension)))

	// 创建物理集合
	err = m.client.CreateCollection(ctx, schema, entity.DefaultShardNumber)
	if err != nil {
		return fmt.Errorf("create collection failed: %w", err)
	}

	log.Printf("✅ 集合 '%s' 创建成功 (维度: %d)", m.collection, m.dimension)

	// 关键性能调优：创建 IVF_FLAT 倒排索引
	// L2 欧式距离；nlist=128 (聚类中心数量)
	idx, err := entity.NewIndexIvfFlat(entity.L2, 128)
	if err != nil {
		return fmt.Errorf("create index params failed: %w", err)
	}

	err = m.client.CreateIndex(ctx, m.collection, "embedding", idx, false)
	if err != nil {
		return fmt.Errorf("create index failed: %w", err)
	}

	log.Printf("✅ 集合 '%s' 索引构建完成 (IVF_FLAT)", m.collection)
	return nil
}

// Insert 将清洗后的知识点及其向量入库，并强制 Flush 确保可见性
func (m *MilvusClient) Insert(ctx context.Context, texts, sources []string, embeddings [][]float32) error {
	if len(texts) != len(embeddings) || len(texts) != len(sources) {
		return fmt.Errorf("length mismatch: texts=%d, sources=%d, embeddings=%d", len(texts), len(sources), len(embeddings))
	}

	// 构造各字段列数据
	textCol := entity.NewColumnVarChar("text", texts)
	sourceCol := entity.NewColumnVarChar("source", sources)
	embCol := entity.NewColumnFloatVector("embedding", m.dimension, embeddings)

	// 执行批量插入
	_, err := m.client.Insert(ctx, m.collection, "", textCol, sourceCol, embCol)
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}

	// Flush 强制数据同步刷新至磁盘，确保后续 Search 可见
	err = m.client.Flush(ctx, m.collection, false)
	if err != nil {
		return fmt.Errorf("flush failed: %w", err)
	}

	log.Printf("✅ 成功插入 %d 条知识向量到集合 '%s'", len(texts), m.collection)
	return nil
}

// Search 执行核心的向量空间检索，寻找与 Query 语义最贴近的背景知识
func (m *MilvusClient) Search(ctx context.Context, queryVec []float32, topK int) ([]SearchResult, error) {
	// 在搜索前必须确保 Collection 加载到内存内存 (Load)
	err := m.client.LoadCollection(ctx, m.collection, false)
	if err != nil {
		log.Printf("💡 提示: 集合加载状态确认: %v (可能已在线)", err)
	}

	// 指定搜索参数：nprobe 代表扫描的聚类中心数量，值越大精度越高但耗时增加
	sp, err := entity.NewIndexIvfFlatSearchParam(16)
	if err != nil {
		return nil, fmt.Errorf("create search param failed: %w", err)
	}

	// 执行向量检索
	results, err := m.client.Search(
		ctx,
		m.collection,
		nil,                        // 不限定分片
		"",                         // 无标量过滤表达式
		[]string{"text", "source"}, // 指定返回的字段
		[]entity.Vector{entity.FloatVector(queryVec)}, // 目标向量
		"embedding", // 检索字段
		entity.L2,   // 度量指标 (欧氏距离)
		topK,        // 召回数量
		sp,          // 搜索参数
	)
	if err != nil {
		return nil, fmt.Errorf("milvus search execution failed: %w", err)
	}

	// 解析多路检索结果 (此处由于只传了一个 Query，遍历一次 result 即可)
	var out []SearchResult
	for _, result := range results {
		// 动态转换为列数据类型
		textCol, ok := result.Fields.GetColumn("text").(*entity.ColumnVarChar)
		if !ok {
			continue
		}
		sourceCol, ok := result.Fields.GetColumn("source").(*entity.ColumnVarChar)
		if !ok {
			continue
		}

		for i := 0; i < result.ResultCount; i++ {
			chunkID, _ := result.IDs.GetAsInt64(i)
			out = append(out, SearchResult{
				ChunkID: chunkID,
				Text:    textCol.Data()[i],
				Source:  sourceCol.Data()[i],
				Score:   result.Scores[i],
			})
		}
	}

	log.Printf("🔍 检索完成，从向量库召回 %d 条相关背景", len(out))
	return out, nil
}

// Close 安全关闭连接池
func (m *MilvusClient) Close() error {
	return m.client.Close()
}

// GetRowCount 实用工具函数：实时获取集合内存储的向量总数
func (m *MilvusClient) GetRowCount(ctx context.Context) (int64, error) {
	stats, err := m.client.GetCollectionStatistics(ctx, m.collection)
	if err != nil {
		return 0, err
	}
	if rowCountStr, ok := stats["row_count"]; ok {
		var count int64
		fmt.Sscanf(rowCountStr, "%d", &count)
		return count, nil
	}
	return 0, nil
}
