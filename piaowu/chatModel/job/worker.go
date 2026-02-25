package job

import (
	"context"
	"encoding/json"
	"example_shop/chatModel/agent"
	"example_shop/chatModel/model"
	"example_shop/chatModel/repository"
	"fmt"
	"time"
)

// Worker 负责后台异步任务的消费与生命周期管理
type Worker struct {
	jobChan chan agent.GraphInput // 任务接收通道
}

// NewWorker 创建一个新的 Worker 实例
func NewWorker(jobChan chan agent.GraphInput) *Worker {
	return &Worker{jobChan: jobChan}
}

// Start 启动无限循环监听任务通道，直到 Context 取消
func (w *Worker) Start(ctx context.Context) {
	fmt.Println("[Worker] 异步消费协程已启动，准备处理 AI 任务...")
	for {
		select {
		case <-ctx.Done():
			fmt.Println("[Worker] 接收到退出信号，正在关闭...")
			return
		case in := <-w.jobChan:
			// 每次从通道获取一个任务执行
			w.processJob(in)
		}
	}
}

// processJob 执行单个 Job 的核心业务逻辑
// 包含：状态更新、Eino 图运行、重试机制、结果落库
func (w *Worker) processJob(in agent.GraphInput) {
	// 流程 1: 将任务状态从 pending 切换为 processing
	// 这一步对于实现“任务可见性”和“故障恢复”至关重要
	repository.DB.Model(&model.AIJob{}).Where("job_id = ?", in.TraceID).Update("status", "processing")

	var out agent.GraphOutput
	var err error

	// 流程 2: 弹性执行机制 (重试逻辑)
	// 由于 LLM API 存在限流或偶发性网络超时，我们内置了 3 次重试机会
	for i := 0; i < 3; i++ {
		// 调用 Eino 编译后的图实例进行同步阻塞调用
		out, err = agent.GetRunnable().Invoke(context.Background(), in)
		if err == nil {
			break // 执行成功，退出重试
		}

		// 若执行失败，且还有重试次数，则进行指数级退避等待 (2s, 4s...)
		if i < 2 {
			fmt.Printf("[Worker] 任务 %s 执行异常，正在进行第 %d 次自动重试... 错误: %v\n", in.TraceID, i+1, err)
			time.Sleep(time.Duration(2*(i+1)) * time.Second)
		}
	}

	// 流程 3: 结果汇总处理
	updates := map[string]interface{}{}
	if err != nil {
		// 最终重试失败，记录错误原因
		updates["status"] = "failed"
		updates["error"] = err.Error()
	} else {
		// 成功完成，序列化 AI 生成的结构化产物
		resJSON, _ := json.Marshal(out)
		updates["status"] = "completed"
		updates["result"] = string(resJSON)

		// 支路 A: 沉淀会话摘要，用于管理端展示
		repository.DB.Create(&model.ConversationSummary{
			ConversationID: in.ConversationID,
			Summary:        out.Summary,
			TraceID:        in.TraceID,
		})

		// 支路 B: 将建议回复作为一条特殊的“系统消息”插入到消息流中
		// 发送者标记为 CS9999 (虚拟 AI 助理)
		if out.Suggestion != "" {
			repository.DB.Create(&model.ConvMessage{
				ConvID:     in.ConversationID,
				SenderType: 1, // 标识为客服侧发送
				SenderID:   "CS9999",
				MsgContent: out.Suggestion,
				SendTime:   time.Now(),
			})
		}
	}

	// 流程 4: 终态更新，将处理结果（包含成功数据或错误日志）持久化回 Job 表
	repository.DB.Model(&model.AIJob{}).Where("job_id = ?", in.TraceID).Updates(updates)
	fmt.Printf("[Worker] 任务 %s 处理完毕，最终状态: %s\n", in.TraceID, updates["status"])
}
