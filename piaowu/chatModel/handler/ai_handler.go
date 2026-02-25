package handler

import (
	"context"
	"encoding/json"
	"example_shop/chatModel/agent"
	"example_shop/chatModel/config"
	"example_shop/chatModel/job"
	"example_shop/chatModel/model"
	"example_shop/chatModel/repository"
	"example_shop/pkg/ai"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qianfan"
	eino_model "github.com/cloudwego/eino/components/model"
)

// AIProcessRequest 定义了发起 AI 处理请求的结构
type AIProcessRequest struct {
	ConversationID string `json:"conversation_id"` // 会话 ID
	Content        string `json:"content"`         // 消息内容
	ActorID        int64  `json:"actor_id"`        // 操作人 ID (客服或系统)
}

// AIProcessResponse 定义了同步返回的风险评估结果
type AIProcessResponse struct {
	JobID string `json:"job_id"` // 异步任务 ID
	Risk  struct {
		Level string `json:"level"` // 风险等级: Low, Medium, High
		Type  string `json:"type"`  // 风险类型: 如投诉倾向、敏感词等
	} `json:"risk"`
}

var (
	classifier *ai.Classifier        // 本地风险分类器，用于同步秒回
	jobChan    chan agent.GraphInput // 异步任务通道，连接 Handler 和 Worker
	worker     *job.Worker           // 异步任务执行器
)

// Init 初始化 AI 处理器的核心组件
func Init() error {
	// 1. 初始化本地分类器 (同步预警的核心)
	classifier = ai.NewClassifier()

	// 2. 根据配置初始化不同的大模型供应商 (LLM)
	var chatModel eino_model.ChatModel
	var err error
	if config.GlobalConfig.AI.Provider == "qianfan" {
		// 百度千帆模型初始化
		qfCfg := qianfan.GetQianfanSingletonConfig()
		qfCfg.BearerToken = config.GlobalConfig.AI.APIKey
		chatModel, err = qianfan.NewChatModel(context.Background(), &qianfan.ChatModelConfig{Model: config.GlobalConfig.AI.Model})
	} else if config.GlobalConfig.AI.Provider == "ark" {
		// 火山引擎 (豆包) 模型初始化
		chatModel, err = ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
			BaseURL: config.GlobalConfig.AI.BaseURL,
			APIKey:  config.GlobalConfig.AI.APIKey,
			Model:   config.GlobalConfig.AI.Model,
		})
	} else {
		// 备用或通用的 OpenAI 兼容模型
		os.Setenv("OPENAI_API_BASE", config.GlobalConfig.AI.BaseURL)
		chatModel, err = openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
			BaseURL: config.GlobalConfig.AI.BaseURL,
			APIKey:  config.GlobalConfig.AI.APIKey,
			Model:   config.GlobalConfig.AI.Model,
		})
	}
	if err != nil {
		return fmt.Errorf("llm init failed: %w", err)
	}

	// 3. 加载本地知识库数据 (用于 RAG 失效时的兜底匹配)
	kbData, _ := ioutil.ReadFile("chatModel/resources/kb.json")
	var kb []struct{ Question, Answer string }
	json.Unmarshal(kbData, &kb)

	// 4. 初始化 Eino Agent (将 LLM 和 知识库注入到决策图中)
	if err := agent.Init(chatModel, kb); err != nil {
		return fmt.Errorf("agent init failed: %w", err)
	}

	// 5. 初始化并启动异步 Worker 协程
	// 缓冲区设置为 100，防止短时间高并发请求阻塞主流程
	jobChan = make(chan agent.GraphInput, 100)
	worker = job.NewWorker(jobChan)
	go worker.Start(context.Background())

	return nil
}

// ProcessHandler 处理同步接入请求
// 流程：风险分类 -> 创建 Job -> 派发异步任务 -> 立即返回
func ProcessHandler(w http.ResponseWriter, r *http.Request) {
	var req AIProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	// 核心设计：同步执行风险分类 (秒回逻辑)
	// 在不调用大模型的前提下，通过本地分类器快速判断是否有投诉或违规风险
	res := classifier.Classify(req.Content)
	riskLevel, riskType := "Low", "None"
	if res.Confidence > 0.3 {
		riskLevel, riskType = "Medium", res.CategoryName
	}

	// 生成唯一的任务 ID，由于是在同步链路生成，可立即通过 API 告知客户端
	jobID := fmt.Sprintf("job-%d-%s", time.Now().UnixNano(), req.ConversationID)

	// 将初始状态为 pending 的任务录入数据库，确保可追踪性
	repository.DB.Create(&model.AIJob{
		JobID:          jobID,
		ConversationID: req.ConversationID,
		Status:         "pending",
	})

	// 关键解耦点：将繁重的 RAG 和 LLM 计算逻辑通过 channel 派发给异步协程
	jobChan <- agent.GraphInput{
		ConversationID: req.ConversationID,
		Content:        req.Content,
		ActorID:        req.ActorID,
		TraceID:        jobID,
	}

	// 立即返回同步预警结果，客服系统据此决定是否优先人工介入
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AIProcessResponse{
		JobID: jobID,
		Risk: struct {
			Level string `json:"level"`
			Type  string `json:"type"`
		}{Level: riskLevel, Type: riskType},
	})
}

// JobStatusHandler 提供查询接口，供前端轮询异步任务的处理进展
func JobStatusHandler(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	var jobItem model.AIJob
	if err := repository.DB.Where("job_id = ?", jobID).First(&jobItem).Error; err != nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobItem)
}
