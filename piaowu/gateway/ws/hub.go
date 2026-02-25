package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"example_shop/gateway/rpc"
	"example_shop/pkg/logger"
	"example_shop/service/customer/kitex_gen/customer"

	"go.uber.org/zap"
)

// Hub 维护活跃的 WebSocket 连接并提供消息路由能力。
//
// 设计要点：
// - 同账号互踢：同一 user_id 的新连接会替换旧连接，避免多端同时在线导致消息乱序/重复
// - 非阻塞投递：向 Client.send 投递采用 select+default，缓冲满则认为连接不可用并注销
// - 连接状态与统计：staffs 仅用于在线客服统计，不参与权限判断
type Hub struct {
	// clients 保存当前在线连接：key 为用户 ID。
	clients map[int64]*Client
	// staffs 记录在线客服/管理员的用户 ID，用于统计在线客服人数。

	staffs map[int64]bool
	// broadcast 用于广播消息：Run 主循环会将消息投递给所有在线连接。
	broadcast chan []byte
	// register 用于注册连接：由建立连接的逻辑投递到该通道。
	register chan *Client
	// unregister 用于注销连接：由连接关闭、写缓冲溢出等场景触发。
	unregister chan *Client
	// rpcClient 用于调用后端 RPC 做消息持久化、状态同步等。
	rpcClient *rpc.CustomerClient
	// aiServiceURL AI 服务地址，用于调用 chatModel
	aiServiceURL string
	// mu 保护 clients/staffs 的并发访问。
	mu sync.RWMutex
}

// NewHub 创建新的Hub实例
func NewHub(rpcClient *rpc.CustomerClient, aiServiceURL string) *Hub {
	if aiServiceURL == "" {
		aiServiceURL = "http://chat-model:8082" // 默认本地开发地址
	}
	logger.Info("Hub initialized with AI service", zap.String("aiServiceURL", aiServiceURL))
	return &Hub{
		broadcast:    make(chan []byte),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		clients:      make(map[int64]*Client),
		staffs:       make(map[int64]bool),
		rpcClient:    rpcClient,
		aiServiceURL: aiServiceURL,
	}
}

// Run 启动 Hub 的主循环。
//
// 该方法应在单独 goroutine 中运行（例如：go hub.Run()），通过三类事件驱动：
// - register：新连接注册 + 同账号互踢
// - unregister：连接注销 + 资源释放
// - broadcast：向所有在线连接广播消息（非阻塞投递）
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()

			// 同账号互踢：先关闭旧连接的发送通道，再替换映射。
			// 这里仅关闭 send 通道，不直接操作底层网络连接；Client 的写协程应在 send 关闭后自行退出并关闭 conn。
			if old, ok := h.clients[client.UserID]; ok {
				old.closeOnce.Do(func() {
					close(old.send)
				})
				delete(h.clients, client.UserID)
			}

			h.clients[client.UserID] = client
			if client.Role == "customer_service" || client.Role == "admin" {
				h.staffs[client.UserID] = true
			}

			h.mu.Unlock()
			logger.Info("Client registered", zap.Int64("user_id", client.UserID), zap.String("role", client.Role))

		case client := <-h.unregister:
			h.mu.Lock()

			// 仅当该 client 仍在映射中时才执行清理，避免重复注销造成误删。
			if _, ok := h.clients[client.UserID]; ok {
				delete(h.clients, client.UserID)
				client.closeOnce.Do(func() {
					close(client.send)
				})
				delete(h.staffs, client.UserID)
			}

			h.mu.Unlock()
			logger.Info("Client unregistered", zap.Int64("user_id", client.UserID))

		case message := <-h.broadcast:
			// 广播采用“尽力而为”策略：某个连接写缓冲满，不阻塞整体广播，直接认为该连接不健康并清理。
			//
			// 注意这里使用写锁，因为清理不健康连接时需要 delete clients。
			h.mu.Lock()
			for _, client := range h.clients {
				select {
				case client.send <- message:
				default:
					client.closeOnce.Do(func() {
						close(client.send)
					})
					delete(h.clients, client.UserID)
				}
			}
			h.mu.Unlock()
		}
	}
}

// UnicastRaw 发送原始字节消息给指定用户
// 参数:
//   - targetID: 目标用户ID
//   - message: 要发送的字节消息
func (h *Hub) UnicastRaw(targetID int64, message []byte) {
	h.mu.RLock()
	client, ok := h.clients[targetID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	// 单播同样采用非阻塞投递：缓冲满说明对端消费太慢或已失联，避免阻塞调用方。
	select {
	case client.send <- message:
	default:
		logger.Warn("Failed to send message to client (buffer full)", zap.Int64("user_id", targetID))
		h.unregister <- client
	}
}

// UnicastJSON 发送JSON格式消息给指定用户
// 自动将对象序列化为JSON后发送
// 参数:
//   - targetID: 目标用户ID
//   - v: 要发送的对象
func (h *Hub) UnicastJSON(targetID int64, v interface{}) {
	msg, err := json.Marshal(v)
	if err != nil {
		logger.Error("JSON marshal error", zap.Error(err))
		return
	}
	h.UnicastRaw(targetID, msg)
}

// HandleMessage 处理从客户端接收到的业务消息
// 当前支持的消息类型：
// - "chat": 聊天消息，会持久化到数据库并转发给目标用户
// 参数:
//   - client: 发送消息的客户端
//   - msg: 原始消息字节
func (h *Hub) HandleMessage(client *Client, msg []byte) {
	// 统一的消息信封格式：
	// {
	//   "type": "chat",
	//   "payload": { ... }
	// }
	var input struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(msg, &input); err != nil {
		logger.Warn("Invalid message format", zap.Error(err))
		return
	}

	if input.Type == "chat" {
		// chat payload 示例：
		// {
		//   "conversation_id": "xxx",
		//   "content": "hello",
		//   "msg_type": 1,
		//   "to_user_id": 1001
		// }
		var chatMsg struct {
			ConversationID string `json:"conversation_id"`
			Content        string `json:"content"`
			MsgType        int32  `json:"msg_type"`
			ToUserID       int64  `json:"to_user_id"`
		}
		if err := json.Unmarshal(input.Payload, &chatMsg); err != nil {
			logger.Warn("Invalid chat payload", zap.Error(err))
			return
		}

		// 为本次消息处理链路创建 TraceID，便于在网关日志与 RPC 日志之间串联排查。
		ctx := context.Background()
		traceID := logger.NewTraceID()
		ctx = logger.WithTraceID(ctx, traceID)

		logger.InfoWithTrace(ctx, "Received chat message",
			zap.Int64("from", client.UserID),
			zap.Int64("to", chatMsg.ToUserID),
			zap.String("content", chatMsg.Content))

		// 后端按 sender_type 区分消息来源：
		// - 0: 普通用户
		// - 1: 客服/管理员（统一视作客服侧）
		// - 2: 系统消息
		senderType := int8(0)
		if client.Role == "customer_service" || client.Role == "admin" {
			senderType = 1
		}

		// 先持久化再转发：即使对端不在线，消息也能落库用于后续拉取/审计。
		req := &customer.SendConversationMessageReq{
			ConvId:     chatMsg.ConversationID,
			SenderType: senderType,
			SenderId:   fmt.Sprintf("%d", client.UserID),
			MsgContent: chatMsg.Content,
		}

		_, err := h.rpcClient.SendConversationMessage(ctx, req)
		if err != nil {
			logger.ErrorWithTrace(ctx, "Failed to save message", zap.Error(err))
			return
		}

		// 构造转发给双方的消息体。
		// create_time 由网关生成用于前端展示；与后端落库时间可能存在细微差异。
		response := map[string]interface{}{
			"type": "chat",
			"payload": map[string]interface{}{
				"conversation_id": chatMsg.ConversationID,
				"from_user_id":    client.UserID,
				"content":         chatMsg.Content,
				"msg_type":        chatMsg.MsgType,
				"create_time":     time.Now().Format(time.RFC3339),
			},
		}

		// ACK 给发送方：用于前端确认已发送/回显（并不代表对端已收到）。
		h.UnicastJSON(client.UserID, response)

		// 转发给目标用户：对端不在线或缓冲满时会丢弃并触发注销，不影响发送方 ACK。
		if chatMsg.ToUserID != 0 {
			h.UnicastJSON(chatMsg.ToUserID, response)
		}

		// [AI-First] 如果接收方是 ROBOT (9999)，则触发 AI 处理逻辑
		if chatMsg.ToUserID == 9999 {
			go h.handleAIMessage(ctx, client.UserID, chatMsg.ConversationID, chatMsg.Content)
		}
	}
}

// handleAIMessage 处理发送给 AI 的消息
func (h *Hub) handleAIMessage(ctx context.Context, userID int64, convID string, content string) {
	// 1. 调用 AI 服务异步接口
	aiURL := h.aiServiceURL + "/api/ai/process"
	reqBody, _ := json.Marshal(map[string]interface{}{
		"conversation_id": convID,
		"content":         content,
		"actor_id":        userID,
	})

	resp, err := http.Post(aiURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		logger.ErrorWithTrace(ctx, "Failed to call AI service", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var aiInitResp struct {
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(body, &aiInitResp); err != nil {
		logger.ErrorWithTrace(ctx, "Failed to parse AI response", zap.Error(err))
		return
	}

	// 2. 轮询 AI 任务状态
	jobID := aiInitResp.JobID
	if jobID == "" {
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)
	for {
		select {
		case <-ticker.C:
			statusURL := fmt.Sprintf("%s/api/ai/job/status?job_id=%s", h.aiServiceURL, jobID)
			sResp, err := http.Get(statusURL)
			if err != nil {
				continue
			}
			sBody, _ := ioutil.ReadAll(sResp.Body)
			sResp.Body.Close()

			var jobInfo struct {
				Status string `json:"status"`
				Result string `json:"result"`
			}
			json.Unmarshal(sBody, &jobInfo)

			if jobInfo.Status == "completed" {
				var result struct {
					Suggestion string `json:"suggestion"`
				}
				json.Unmarshal([]byte(jobInfo.Result), &result)

				// 推送回复给用户
				response := map[string]interface{}{
					"type": "chat",
					"payload": map[string]interface{}{
						"conversation_id": convID,
						"from_user_id":    9999,
						"content":         result.Suggestion,
						"msg_type":        1,
						"create_time":     time.Now().Format(time.RFC3339),
					},
				}
				h.UnicastJSON(userID, response)
				return
			} else if jobInfo.Status == "failed" {
				return
			}
		case <-timeout:
			logger.WarnWithTrace(ctx, "AI job timeout", zap.String("job_id", jobID))
			return
		case <-ctx.Done():
			return
		}
	}
}

// GetOnlineStats 获取在线统计信息
// 返回当前在线客服数量、客服ID列表和总连接数
// 返回: 包含统计信息的map
func (h *Hub) GetOnlineStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// staffs 的 value 不重要，只需要 key 列表即可。
	staffList := make([]int64, 0, len(h.staffs))
	for uid := range h.staffs {
		staffList = append(staffList, uid)
	}

	return map[string]interface{}{
		"online_staff_count": len(h.staffs),
		"online_staff_ids":   staffList,
		"total_connections":  len(h.clients),
	}
}
