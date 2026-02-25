package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"example_shop/chatModel/config"
	"example_shop/chatModel/embedding"
	"example_shop/chatModel/repository"
)

// 知识库根目录：存放待向量化的原始文档 (.md 或 .txt)
const knowledgeBaseDir = "chatModel/knowledgeBase"

// 语义分块大小 (Chunk Size)：单个向量化单位包含的最大字符数
// 500 字符通常能包含完整的上下文语义，同时也符合 Embedding 模型的输入限制
const chunkSize = 500

// main 是知识库同步工具的入口
// 流程：读取文件 -> 清洗分块 -> 向量化 -> 批量入库
func main() {
	fmt.Println("=== 🚀 开源知识库离线导入与向量化工具 (V1.0) ===")
	fmt.Println()

	// 1. 环境自检与配置加载
	fmt.Println("[1/5] 正在加载全局配置与连接参数...")
	if err := config.InitConfig(); err != nil {
		log.Fatalf("配置初始化失败: %v", err)
	}
	fmt.Printf("  ✓ 目标向量库集合: %s\n", config.GlobalConfig.Milvus.Collection)

	// 2. 初始化嵌入客户端并获取模型维度
	fmt.Println("\n[2/5] 正在建立 Embedding 模型长连接...")
	embClient := embedding.NewClient(
		config.GlobalConfig.Embedding.APIKey,
		config.GlobalConfig.Embedding.Model,
	)
	ctx := context.Background()
	dim, err := embClient.GetDimension(ctx)
	if err != nil {
		log.Fatalf("Embedding 预检失败: %v", err)
	}
	fmt.Printf("  ✓ 模型连接成功，探测向量维度: %d\n", dim)

	// 3. 准备 Milvus 存储集合
	fmt.Println("\n[3/5] 正在校验向量数据库集合状态...")
	milvusClient, err := repository.NewMilvusClient(
		config.GlobalConfig.Milvus.Host,
		config.GlobalConfig.Milvus.Port,
		config.GlobalConfig.Milvus.Collection,
		dim,
	)
	if err != nil {
		log.Fatalf("Milvus 连接失败: %v", err)
	}
	defer milvusClient.Close()

	if err := milvusClient.EnsureCollection(ctx); err != nil {
		log.Fatalf("创建/校验 Collection 失败: %v", err)
	}

	// 4. 读取原始磁盘文件并执行语义切分
	fmt.Println("\n[4/5] 正在扫描磁盘文档并执行语义切分 (Recursive Walk)...")
	chunks, sources, err := loadKnowledgeBase(knowledgeBaseDir)
	if err != nil {
		log.Fatalf("读取知识库失败: %v", err)
	}
	fmt.Printf("  ✓ 扫描完毕，生成共 %d 个语义文本块\n", len(chunks))

	// 5. 向量化处理并流式导入
	fmt.Println("\n[5/5] 正在启动批处理引擎进行向量化入库...")
	batchSize := 10 // 每批处理 10 条，平衡网络开销与同步速度
	total := len(chunks)
	imported := 0

	for i := 0; i < total; i += batchSize {
		end := i + batchSize
		if end > total {
			end = total
		}

		batchTexts := chunks[i:end]
		batchSources := sources[i:end]

		// 执行远程 Embedding 向量化
		embeddings, err := embClient.EmbedBatch(ctx, batchTexts)
		if err != nil {
			log.Printf("  ⚠ 批次 %d-%d 向量化失败，跳过: %v", i, end, err)
			continue
		}

		// 存入向量数据库
		if err := milvusClient.Insert(ctx, batchTexts, batchSources, embeddings); err != nil {
			log.Printf("  ⚠ 批次 %d-%d 写入向量库失败: %v", i, end, err)
			continue
		}

		imported += len(batchTexts)
		fmt.Printf("  ✓ 处理进度: %d/%d (%.1f%%)\n", imported, total, float64(imported)/float64(total)*100)

		// 指数级或固定间隔停顿，避免触发 API 限流 (Rate Limit)
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\n🎉 === 知识库同步任务全部完成 ===")
	count, _ := milvusClient.GetRowCount(ctx)
	fmt.Printf("✓ 本次成功导入: %d 条\n", imported)
	fmt.Printf("✓ 向量集当前总存量: %d 条\n", count)
}

// loadKnowledgeBase 递归扫描目录下的 Markdown 和文本文件
func loadKnowledgeBase(dir string) ([]string, []string, error) {
	var allChunks []string
	var allSources []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 文件后缀过滤逻辑
		ext := strings.ToLower(filepath.Ext(path))
		if info.IsDir() || (ext != ".md" && ext != ".txt") {
			return nil
		}

		fmt.Printf("  - 正在读取: %s\n", path)

		content, err := readFile(path)
		if err != nil {
			log.Printf("    ⚠ 无法读取该文件: %v", err)
			return nil
		}

		// 核心清洗环节：执行分块算法
		chunks := splitIntoChunks(content, chunkSize)
		source := filepath.Base(path) // 记录来源文件名，用于 RAG 溯源

		for _, chunk := range chunks {
			allChunks = append(allChunks, chunk)
			allSources = append(allSources, source)
		}

		return nil
	})

	return allChunks, allSources, err
}

// readFile 从磁盘读取原始文本内容
func readFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return strings.Join(lines, "\n"), scanner.Err()
}

// splitIntoChunks 语义分块核心算法
// 策略：优先按“双换行段落”切分，若段落过长则退而求其次按“句号”强行切分，确保不超过模型 Token 限制
func splitIntoChunks(text string, maxSize int) []string {
	var chunks []string
	paragraphs := strings.Split(text, "\n\n")

	var current strings.Builder
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// 若当前缓冲区 + 新段落未溢出，则合并，减少向量化碎片
		if current.Len()+len(para) > maxSize && current.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}

		// 处理极长段落：执行细粒度（按句号）分割逻辑
		if len(para) > maxSize {
			if current.Len() > 0 {
				chunks = append(chunks, strings.TrimSpace(current.String()))
				current.Reset()
			}
			sentences := strings.Split(para, "。")
			for _, s := range sentences {
				s = strings.TrimSpace(s)
				if s == "" {
					continue
				}
				if current.Len()+len(s) > maxSize && current.Len() > 0 {
					chunks = append(chunks, strings.TrimSpace(current.String()))
					current.Reset()
				}
				current.WriteString(s)
				current.WriteString("。")
			}
		} else {
			if current.Len() > 0 {
				current.WriteString("\n\n")
			}
			current.WriteString(para)
		}
	}

	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}

	return chunks
}
