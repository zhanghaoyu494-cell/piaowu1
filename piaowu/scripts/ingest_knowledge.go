package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"example_shop/chatModel/config"
	"example_shop/chatModel/embedding"
	"example_shop/chatModel/repository"
)

func main() {
	// 命令行参数
	kbDir := flag.String("dir", "chatModel/knowledgeBase", "知识库目录路径")
	batchSize := flag.Int("batch", 10, "批量处理大小")
	flag.Parse()

	fmt.Println("========================================")
	fmt.Println("知识库导入工具 v1.0")
	fmt.Println("========================================")

	// 加载配置
	if err := config.InitConfig(); err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 验证配置
	if config.GlobalConfig.Embedding.APIKey == "" {
		fmt.Println("❌ Embedding API Key 未配置")
		os.Exit(1)
	}
	if config.GlobalConfig.Milvus.Host == "" {
		fmt.Println("❌ Milvus Host 未配置")
		os.Exit(1)
	}

	fmt.Printf("📂 知识库目录: %s\n", *kbDir)
	fmt.Printf("🔢 批量大小: %d\n", *batchSize)
	fmt.Printf("🌐 Milvus: %s:%d\n", config.GlobalConfig.Milvus.Host, config.GlobalConfig.Milvus.Port)
	fmt.Printf("🤖 Embedding模型: %s\n", config.GlobalConfig.Embedding.Model)
	fmt.Println("----------------------------------------")

	ctx := context.Background()

	// 初始化 Embedding 客户端
	embClient := embedding.NewClient(
		config.GlobalConfig.Embedding.APIKey,
		config.GlobalConfig.Embedding.Model,
	)

	// 测试 Embedding API
	fmt.Print("🔄 测试 Embedding API... ")
	testVec, err := embClient.Embed(ctx, "测试文本")
	if err != nil {
		fmt.Printf("❌ 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 成功 (维度: %d)\n", len(testVec))

	// 使用实际维度
	dimension := len(testVec)

	// 初始化 Milvus 客户端
	fmt.Print("🔄 连接 Milvus... ")
	milvusClient, err := repository.NewMilvusClient(
		config.GlobalConfig.Milvus.Host,
		config.GlobalConfig.Milvus.Port,
		config.GlobalConfig.Milvus.Collection,
		dimension,
	)
	if err != nil {
		fmt.Printf("❌ 失败: %v\n", err)
		os.Exit(1)
	}
	defer milvusClient.Close()
	fmt.Println("✅ 成功")

	// 确保 Collection 存在
	fmt.Print("🔄 初始化 Collection... ")
	if err := milvusClient.EnsureCollection(ctx); err != nil {
		fmt.Printf("❌ 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ 成功")

	// 扫描并加载知识库文件
	fmt.Println("----------------------------------------")
	fmt.Println("📖 开始加载知识库文件...")

	var allChunks []chunkInfo
	err = filepath.Walk(*kbDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".md" || ext == ".txt" {
			chunks := loadAndChunk(path)
			fmt.Printf("  📄 %s: %d 个文本块\n", filepath.Base(path), len(chunks))
			allChunks = append(allChunks, chunks...)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("❌ 扫描目录失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📊 总计: %d 个文本块\n", len(allChunks))
	fmt.Println("----------------------------------------")

	if len(allChunks) == 0 {
		fmt.Println("⚠️ 没有找到可导入的内容")
		os.Exit(0)
	}

	// 批量向量化和入库
	fmt.Println("🚀 开始向量化和入库...")
	startTime := time.Now()
	successCount := 0

	for i := 0; i < len(allChunks); i += *batchSize {
		end := i + *batchSize
		if end > len(allChunks) {
			end = len(allChunks)
		}
		batch := allChunks[i:end]

		texts := make([]string, len(batch))
		sources := make([]string, len(batch))
		for j, c := range batch {
			texts[j] = c.text
			sources[j] = c.source
		}

		// 向量化
		embeddings, err := embClient.EmbedBatch(ctx, texts)
		if err != nil {
			fmt.Printf("  ⚠️ 批次 %d-%d 向量化失败: %v\n", i, end, err)
			continue
		}

		// 入库
		if err := milvusClient.Insert(ctx, texts, sources, embeddings); err != nil {
			fmt.Printf("  ⚠️ 批次 %d-%d 入库失败: %v\n", i, end, err)
			continue
		}

		successCount += len(batch)
		fmt.Printf("  ✅ 进度: %d/%d (%.1f%%)\n", successCount, len(allChunks), float64(successCount)/float64(len(allChunks))*100)
	}

	elapsed := time.Since(startTime)
	fmt.Println("----------------------------------------")
	fmt.Printf("🎉 导入完成!\n")
	fmt.Printf("   成功: %d/%d 条\n", successCount, len(allChunks))
	fmt.Printf("   耗时: %v\n", elapsed)

	// 验证
	rowCount, _ := milvusClient.GetRowCount(ctx)
	fmt.Printf("   Collection 总行数: %d\n", rowCount)
}

type chunkInfo struct {
	text   string
	source string
}

// loadAndChunk 加载文件并按段落切分
func loadAndChunk(path string) []chunkInfo {
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("警告: 无法打开文件 %s: %v\n", path, err)
		return nil
	}
	defer file.Close()

	source := filepath.Base(path)
	var chunks []chunkInfo
	var current strings.Builder
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// 空行表示段落结束
		if trimmed == "" {
			content := strings.TrimSpace(current.String())
			if len(content) >= 30 { // 至少30字符
				chunks = append(chunks, chunkInfo{
					text:   content,
					source: source,
				})
			}
			current.Reset()
		} else {
			current.WriteString(line + " ")
		}
	}

	// 处理最后一段
	content := strings.TrimSpace(current.String())
	if len(content) >= 30 {
		chunks = append(chunks, chunkInfo{
			text:   content,
			source: source,
		})
	}

	return chunks
}
