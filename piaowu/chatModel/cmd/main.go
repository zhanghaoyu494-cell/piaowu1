package main

import (
	"example_shop/chatModel/config"
	"example_shop/chatModel/handler"
	"example_shop/chatModel/repository"
	"fmt"
	"log"
	"net/http"
)

// main 是 chatModel 服务的启动入口
func main() {
	fmt.Println(">>> 正在启动 ChatModel 智能客服助理服务...")

	// 1. 初始化系统配置 (从 config.yaml 加载)
	// 包含 LLM 密钥、数据库连接串、Milvus 地址等关键参数
	if err := config.InitConfig(); err != nil {
		log.Fatalf("配置加载失败 [FATAL]: %v", err)
	}

	// 2. 初始化持久化层 (MySQL)
	// 建立本地数据库连接池，用于 Job 状态控制和审计日志记录
	if err := repository.InitDB(); err != nil {
		log.Fatalf("数据库连接失败 [FATAL]: %v", err)
	}

	// 3. 初始化 AI 控制核心与 Agent 决策引擎
	// 此处会同步：初始化 LLM 客户端、构建 Eino 决策图、启动异步消费 Worker
	if err := handler.Init(); err != nil {
		log.Fatalf("AI 引擎初始化失败 [FATAL]: %v", err)
	}

	// 4. 注册 HTTP API 路由
	// /api/ai/process: 接受对话请求，执行风险分类并派发异步 AI 任务
	http.HandleFunc("/api/ai/process", handler.ProcessHandler)
	// /api/ai/job/status: 供客户端查询 Job 的处理完成情况（轮询接口）
	http.HandleFunc("/api/ai/job/status", handler.JobStatusHandler)

	// 5. 启动 HTTP 服务监听
	addr := fmt.Sprintf(":%d", config.GlobalConfig.App.Port)
	fmt.Printf("✅ ChatModel 服务 (V1.2 增强版) 已准备就绪，正在监听: %s\n", addr)

	// 开始阻塞执行，持续提供 API 服务
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("接口监听启动失败 [FATAL]: %v", err)
	}
}
