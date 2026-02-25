package agent

import (
	"context"
	"encoding/json"
	"example_shop/chatModel/config"
	"example_shop/chatModel/embedding"
	chatModelModel "example_shop/chatModel/model"
	"example_shop/chatModel/policy"
	"example_shop/chatModel/repository"
	"example_shop/pkg/ai"
	"fmt"
	"log"
	"strings"
	"time"

	eino_model "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	eino_schema "github.com/cloudwego/eino/schema"
)

// GraphInput 定义了 Eino 决策图的初始输入参数
type GraphInput struct {
	ConversationID string // 当前会话 ID
	Content        string // 用户输入的消息原文
	ActorID        int64  // 操作人 ID
	TraceID        string // 追踪 ID (通常是 JobID)
	Context        string // RAG 检索到的背景知识 (由 RAG 节点填充)
	Citations      []int  // 引用文献的 ID 列表 (用于溯源)
}

// GraphOutput 定义了 Eino 决策图的最终输出产物
type GraphOutput struct {
	Summary    string // AI 对会话的简明摘要
	Suggestion string // AI 给出的专业处理建议
	RiskLevel  string // 风险等级
	RiskType   string // 风险类型
	Citations  []int  // 最终使用的引用 ID 列表
}

var (
	// runnable 是编译后的 Eino 图，可直接通过线程安全的方式调用
	runnable        compose.Runnable[GraphInput, GraphOutput]
	classifier      *ai.Classifier
	checker         *policy.Checker          // 数据域隔离检查器
	guard           *policy.Guard            // Prompt 注入检测守卫
	embeddingClient *embedding.Client        // Embedding 向量化客户端
	milvusClient    *repository.MilvusClient // Milvus 向量库客户端接口
	ragEnabled      bool                     // 当前 RAG (检索增强) 是否真正启用
)

// Init 初始化 Agent 引擎，编排 Eino 决策拓扑图
func Init(chatModel eino_model.ChatModel, kb []struct{ Question, Answer string }) error {
	// 1. 初始化安全与分类组件
	classifier = ai.NewClassifier()
	classifier.AddCategory(1, "投诉倾向", []string{"投诉", "举报", "工商局"})

	checker = policy.NewChecker()
	guard = policy.NewGuard()

	// 2. 尝试初始化 RAG 核心组件 (Milvus + Embedding)
	// 如果配置不全或初始化失败，系统将自动进入“回退模式 (Fallback)”
	ragEnabled = false
	if config.GlobalConfig.Embedding.APIKey != "" && config.GlobalConfig.Milvus.Host != "" {
		if err := initRAG(); err != nil {
			log.Printf("⚠️ RAG 组件初始化失败，将使用回退模式: %v", err)
		} else {
			ragEnabled = true
			log.Println("✅ RAG 模式已启用 (Milvus + Embedding)")
		}
	} else {
		log.Println("ℹ️ RAG 配置不完整，默认使用回退模式 (本地知识库)")
	}

	// 3. 构建 Eino Graph
	// 我们采用的是标准的顺序流：START -> Policy -> RAG -> LLM -> END
	g := compose.NewGraph[GraphInput, GraphOutput]()

	// 节点 A: Policy Node (安全与隔离检查)
	// 职责：拦截不合法的请求，防止敏感信息通过 AI 泄露
	g.AddLambdaNode("policy", compose.InvokableLambda(func(ctx context.Context, in GraphInput) (GraphInput, error) {
		// 检查操作人是否真的有权访问该会话 (Row-Level Scope)
		if err := checker.CheckSessionPermission(ctx, in.ActorID, in.ConversationID); err != nil {
			return in, err
		}
		// 检测消息是否包含恶意的 Prompt 注入指令
		if guard.DetectInjection(in.Content) {
			return in, fmt.Errorf("security alert: potential prompt injection detected")
		}
		return in, nil
	}))

	// 节点 B: RAG Node (知识检索增强)
	// 职责：为 LLM 提供外部参考知识，解决其内部知识过时或缺乏业务私有数据的问题
	g.AddLambdaNode("rag", compose.InvokableLambda(func(ctx context.Context, in GraphInput) (GraphInput, error) {
		start := time.Now()
		var found []string // 检索到的背景文本块
		var ids []int      // 文献引用 ID

		// 支路 1: 向量引擎检索 (精度最高)
		if ragEnabled {
			results, err := retrieveFromMilvus(ctx, in.Content)
			if err != nil {
				log.Printf("向量检索异常，启动备用方案: %v", err)
			} else {
				for i, r := range results {
					found = append(found, fmt.Sprintf("[%d] %s (来源: %s)", i+1, r.Text, r.Source))
					ids = append(ids, int(r.ChunkID))
				}
			}
		}

		// 支路 2: 回退搜索 (兜底逻辑)
		// 如果向量库无结果或未启用，则进行关键词粗排匹配
		if len(found) == 0 {
			found, ids = fallbackKeywordMatch(in.Content, kb)
		}

		in.Context = strings.Join(found, "\n")
		in.Citations = ids

		// 记录工具调用审计日志，便于后续分析哪些问题召回了哪些文档
		repository.DB.Create(&chatModelModel.AIToolAuditLog{
			ActorID:   in.ActorID,
			ToolName:  "RAG",
			Input:     in.Content,
			Output:    fmt.Sprintf("Retrieved %d results", len(found)),
			TraceID:   in.TraceID,
			LatencyMs: time.Since(start).Milliseconds(),
		})

		return in, nil
	}))

	// 节点 C: LLM Node (智能生成)
	// 职责：结合背景知识，生成结构化的最终建议和摘要
	g.AddLambdaNode("llm", compose.InvokableLambda(func(ctx context.Context, in GraphInput) (GraphOutput, error) {
		start := time.Now()

		// 组装最终发给大模型的 Prompt
		prompt := buildPrompt(in.Context, in.Content)

		// 调用集成的 LLM 能力 (ark/qianfan/openai 等)
		resp, err := chatModel.Generate(ctx, []*eino_schema.Message{{Role: eino_schema.User, Content: prompt}})

		// 记录大语言模型调用的输入、输出及耗时
		repository.DB.Create(&chatModelModel.AIToolAuditLog{
			ActorID:   in.ActorID,
			ToolName:  "LLM",
			Input:     prompt,
			Output:    fmt.Sprintf("%+v", resp),
			TraceID:   in.TraceID,
			LatencyMs: time.Since(start).Milliseconds(),
		})

		if err != nil {
			return GraphOutput{}, err
		}

		// 关键步骤：解析模型返回的 JSON (处理其可能夹杂的 Markdown 标记)
		res := parseLLMResponse(resp.Content)

		return GraphOutput{
			Summary:    res.Summary,
			Suggestion: res.Suggestion,
			Citations:  in.Citations,
		}, nil
	}))

	// 4. 串联节点，完成 Graph 定稿
	g.AddEdge(compose.START, "policy")
	g.AddEdge("policy", "rag")
	g.AddEdge("rag", "llm")
	g.AddEdge("llm", compose.END)

	// 5. 编译图，生成可执行实例
	compiled, err := g.Compile(context.Background())
	if err != nil {
		return err
	}
	runnable = compiled
	return nil
}

// initRAG 初始化向量检索所需的 Embedding 客户端和 Milvus 连接
func initRAG() error {
	ctx := context.Background()

	// 向量化 API 初始化 (用于将用户问题变为浮点数组)
	embeddingClient = embedding.NewClient(
		config.GlobalConfig.Embedding.APIKey,
		config.GlobalConfig.Embedding.Model,
	)

	// 发起一次预检，确保 Embedding API 状态正常
	testVec, err := embeddingClient.Embed(ctx, "测试连接")
	if err != nil {
		return fmt.Errorf("embedding test failed: %w", err)
	}
	dimension := len(testVec)
	log.Printf("Embedding 维度确认: %d", dimension)

	// 初始化 Milvus 向量库连接
	milvusClient, err = repository.NewMilvusClient(
		config.GlobalConfig.Milvus.Host,
		config.GlobalConfig.Milvus.Port,
		config.GlobalConfig.Milvus.Collection,
		dimension,
	)
	if err != nil {
		return fmt.Errorf("milvus connect failed: %w", err)
	}

	// 检查集合是否存在，不存在则按预设 Schema 自动创建
	if err := milvusClient.EnsureCollection(ctx); err != nil {
		return fmt.Errorf("ensure collection failed: %w", err)
	}

	// 打印当前库内挂载的数据量
	rowCount, _ := milvusClient.GetRowCount(ctx)
	log.Printf("Milvus Collection '%s' 数据行数: %d", config.GlobalConfig.Milvus.Collection, rowCount)

	return nil
}

// retrieveFromMilvus 执行端到端的向量搜索流程
func retrieveFromMilvus(ctx context.Context, query string) ([]repository.SearchResult, error) {
	if embeddingClient == nil || milvusClient == nil {
		return nil, fmt.Errorf("RAG components not initialized")
	}

	// 步骤 1: 将查询短语转为向量点
	queryVec, err := embeddingClient.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query vectorization failed: %w", err)
	}

	// 步骤 2: 在 Milvus 库中寻找距离最近的前 TopK 个点
	topK := config.GlobalConfig.Milvus.TopK
	if topK <= 0 {
		topK = 5
	}

	results, err := milvusClient.Search(ctx, queryVec, topK)
	if err != nil {
		return nil, fmt.Errorf("milvus search failed: %w", err)
	}

	return results, nil
}

// fallbackKeywordMatch 本地知识库兜底检索机制
// 采用基础的关键词包含检测，不依赖外部向量库，适合核心服务高可用降级
func fallbackKeywordMatch(content string, kb []struct{ Question, Answer string }) ([]string, []int) {
	var found []string
	var ids []int
	contentLower := strings.ToLower(content)
	// 高频服务关键词定义
	keywords := []string{"退款", "退货", "政策", "发票", "物流", "订单", "支付", "会员", "积分", "投诉"}

	match := false
	for _, kw := range keywords {
		if strings.Contains(contentLower, kw) {
			match = true
			break
		}
	}

	if match && len(kb) > 0 {
		// 如果命中了任一关键词，则返回预设的头 5 条 FAQ 作为背景提示
		for i, item := range kb {
			found = append(found, fmt.Sprintf("[%d] %s: %s", i+1, item.Question, item.Answer))
			ids = append(ids, i+1)
			if len(found) >= 5 {
				break
			}
		}
	}

	return found, ids
}

// buildPrompt 实现最终的提示词工程 (Prompt Engineering)
// 动态根据是否有 RAG 背景知识来切换 Prompt 模版
func buildPrompt(context, content string) string {
	if context == "" {
		// 无知识背景时的 Prompt
		return fmt.Sprintf(`你是一个专业的客服助手。请根据用户问题给出专业的回复建议。

用户问题: %s

要求:
1. 回复必须为 JSON 格式: {"summary": "问题概述", "suggestion": "处理建议"}
2. 建议要具体、专业、有帮助性。
`, content)
	}

	// 检索到知识后的增强 Prompt
	return fmt.Sprintf(`你是一个专业的客服助手。请根据以下背景知识回答用户问题。

背景知识:
%s

---
用户问题: %s

要求:
1. 回复必须为 JSON 格式: {"summary": "问题概述", "suggestion": "处理建议"}
2. 如果引用了背景知识，请使用 [1], [2] 等标记引用来源。
3. 建议要具体、专业、有帮助性。
`, context, content)
}

// LLMResult 用于结构化解析大模型返回的内容
type LLMResult struct {
	Summary    string `json:"summary"`
	Suggestion string `json:"suggestion"`
}

// parseLLMResponse 解析 LLM 响应，
// 能够自动剔除大模型生成的 Markdown 分界符 (```json)
func parseLLMResponse(content string) LLMResult {
	var res LLMResult

	cleanContent := content
	if strings.Contains(cleanContent, "```json") {
		parts := strings.Split(cleanContent, "```json")
		if len(parts) > 1 {
			cleanContent = strings.Split(parts[1], "```")[0]
		}
	} else if strings.Contains(cleanContent, "```") {
		parts := strings.Split(cleanContent, "```")
		if len(parts) > 1 {
			cleanContent = parts[1]
		}
	}
	cleanContent = strings.TrimSpace(cleanContent)

	if err := json.Unmarshal([]byte(cleanContent), &res); err != nil {
		// 容错逻辑：如果模型生成的 JSON 严重损毁，则直接将原文作为摘要，并提供兜底话术
		res.Summary = content
		res.Suggestion = "处理该请求时遇到逻辑异常，请联系系统管理员或人工客服。"
	}

	return res
}

// GetRunnable 提供给外部调用者的执行句柄
func GetRunnable() compose.Runnable[GraphInput, GraphOutput] {
	return runnable
}

// IsRAGEnabled 查看当前向量引擎状态
func IsRAGEnabled() bool {
	return ragEnabled
}
