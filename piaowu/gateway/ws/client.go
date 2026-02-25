package ws

import (
	"net/http"
	"sync"
	"time"

	"example_shop/pkg/logger"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// writeWait 写操作允许的最长时间
	writeWait = 10 * time.Second

	// pongWait 等待客户端回复 Pong 的最长时间
	pongWait = 60 * time.Second

	// pingPeriod 发送 Ping 的间隔，必须小于 pongWait
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize 单条消息最大允许字节数
	maxMessageSize = 512 * 1024 // 512KB
)

// upgrader 用于将 HTTP 连接升级为 WebSocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许跨域（生产环境应配置白名单）
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client 代表单个 WebSocket 连接。
//
// 字段说明：
// - Hub: 所属的 Hub 实例，用于注册/注销和消息路由
// - conn: 底层 WebSocket 连接
// - send: 待发送消息的缓冲通道（容量 256）
// - UserID/Role: 连接关联的用户信息
// - closeOnce: 确保 send 通道只关闭一次
type Client struct {
	Hub *Hub

	// WebSocket 底层连接
	conn *websocket.Conn

	// 待发送消息的缓冲通道
	send chan []byte

	// 用户信息
	UserID int64
	Role   string // "user", "admin", "customer_service"

	// 确保 send channel 只关闭一次，防止 panic
	closeOnce sync.Once
}

// readPump 持续从 WebSocket 读取消息并交给 Hub 处理。
//
// 该方法应在独立 goroutine 中运行。当连接关闭或读取失败时，
// 会自动向 Hub 发送注销请求并关闭底层连接。
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Warn("WebSocket unexpected close", zap.Error(err))
			}
			break
		}
		c.Hub.HandleMessage(c, message)
	}
}

// writePump 持续从 send 通道读取消息并写入 WebSocket。
//
// 该方法应在独立 goroutine 中运行。主要职责：
// 1. 消费 send 通道中的消息并写入连接
// 2. 定期发送 Ping 维持心跳
// 3. 当 send 通道关闭时发送 CloseMessage 并退出
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub 关闭了通道，发送关闭帧
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量写入：将缓冲区中排队的消息一次性发送，减少系统调用
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// 定时发送 Ping 维持心跳
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWs 处理 WebSocket 升级请求，创建 Client 并启动读写协程。
//
// 参数:
//   - hub: Hub 实例
//   - w: HTTP 响应写入器
//   - r: HTTP 请求
//   - userID: 已认证的用户 ID
//   - role: 用户角色 ("user", "admin", "customer_service")
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, userID int64, role string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade websocket", zap.Error(err))
		return
	}

	client := &Client{
		Hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		UserID: userID,
		Role:   role,
	}

	client.Hub.register <- client

	// 启动读写协程
	go client.writePump()
	go client.readPump()
}
