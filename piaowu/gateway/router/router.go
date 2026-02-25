package router

import (
	"encoding/json"
	"net/http"

	"example_shop/gateway/handler"
	"example_shop/gateway/middleware"
	"example_shop/gateway/ws"
)

// SetupRoutes 配置所有HTTP路由
// 创建并返回一个配置好的HTTP路由器
// 参数:
//   - customerHandler: 客服业务处理器
//   - hub: WebSocket Hub实例
//
// 返回: 配置完成的HTTP多路复用器
func SetupRoutes(customerHandler *handler.CustomerHandler, hub *ws.Hub) *http.ServeMux { // 方法定义：注册并返回网关的 HTTP 路由表（ServeMux）
	mux := http.NewServeMux() // 初始化路由器：使用标准库 ServeMux 作为多路复用器

	// ============ WebSocket 实时通信接口 ============
	// WebSocket连接，支持客服与用户的实时消息通信
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) { // 注册 WebSocket 入口：对 /ws 请求做鉴权并升级连接
		// 优先从 URL 获取 token // 约定：前端可通过 ws://.../ws?token=xxx 传入 token

		token := r.URL.Query().Get("token") // 从 query 参数读取 token：用于鉴权并识别用户/角色
		if token == "" {                    // 若 query 未提供 token：尝试兼容 WebSocket 子协议传参方式
			// 备选：从 WebSocket 子协议头获取 // 兼容：某些 WebSocket 客户端只能在子协议头里携带 token
			token = r.Header.Get("Sec-WebSocket-Protocol") // 从 Sec-WebSocket-Protocol 读取 token：作为鉴权凭证
		} // token 来源选择结束
		// 验证 Token // 步骤：解析 token 并获取 claims（用户ID/角色等）
		claims, err := middleware.ParseToken(token) // 解析 token：失败则拒绝升级，避免匿名建立连接
		if err != nil {                             // token 无效或过期：返回 401
			http.Error(w, "Unauthorized", http.StatusUnauthorized) // 返回未授权：WebSocket 握手阶段直接失败
			return                                                 // 中断处理：不再继续升级连接
		} // token 校验结束
		// 如果从 Sec-WebSocket-Protocol 获取的 token，需要设置对应的响应头 // 兼容：浏览器要求握手响应回显已协商的子协议
		if r.Header.Get("Sec-WebSocket-Protocol") != "" { // 当请求携带子协议头：握手响应需要回写相同值
			w.Header().Set("Sec-WebSocket-Protocol", token) // 回写子协议：避免浏览器认为协商失败而断开
		} // 子协议回写结束
		// 升级为 WebSocket 连接 // 步骤：完成握手并将连接交给 Hub 管理
		ws.ServeWs(hub, w, r, claims.UserID, claims.RoleCode) // 交由 ws 模块处理：建立连接并按用户/角色注册到 Hub
	}) // /ws 路由处理结束

	// 在线状态统计接口（获取当前在线客服和连接数）
	mux.HandleFunc("/api/stats/online", func(w http.ResponseWriter, r *http.Request) { // 注册在线统计接口：返回当前在线客服数/连接数等
		stats := hub.GetOnlineStats()                      // 查询 Hub 在线统计：从内存状态汇总在线信息
		w.Header().Set("Content-Type", "application/json") // 设置响应类型：明确返回 JSON
		json.NewEncoder(w).Encode(map[string]interface{}{  // 写出统一响应结构：code/data/msg
			"code": 0,         // 响应码：0 表示成功
			"data": stats,     // 响应数据：在线统计结构体/映射
			"msg":  "success", // 响应消息：成功提示
		}) // JSON 编码结束
	}) // /api/stats/online 路由处理结束

	// ============ 公共接口（无需登录） ============
	mux.HandleFunc("/api/v1/user/login", customerHandler.Login)       // 用户登录
	mux.HandleFunc("/api/v1/user/register", customerHandler.Register) // 用户注册（仅客服账号）
	mux.HandleFunc("/api/v1/user/logout", customerHandler.Logout)     // 用户退出登录

	// ============ 需要登录的接口 ============
	mux.HandleFunc("/api/v1/user/current", customerHandler.GetCurrentUser) // 获取当前用户信息

	// ============ 客服管理接口 ============
	mux.HandleFunc("/api/customer/get", customerHandler.GetCustomerService)   // 获取单个客服信息
	mux.HandleFunc("/api/customer/list", customerHandler.ListCustomerService) // 查询客服列表

	// ============ 班次配置接口 ============
	mux.HandleFunc("/api/shift/create", middleware.CheckAdminPermission(customerHandler.CreateShiftConfig)) // 创建班次配置
	mux.HandleFunc("/api/shift/list", customerHandler.ListShiftConfig)                                      // 查询班次列表
	mux.HandleFunc("/api/shift/update", middleware.CheckAdminPermission(customerHandler.UpdateShiftConfig)) // 更新班次配置
	mux.HandleFunc("/api/shift/delete", middleware.CheckAdminPermission(customerHandler.DeleteShiftConfig)) // 删除班次配置

	// ============ 排班管理接口 ============
	mux.HandleFunc("/api/schedule/assign", middleware.CheckAdminPermission(customerHandler.AssignSchedule))          // 手动分配排班
	mux.HandleFunc("/api/schedule/auto", middleware.CheckAdminPermission(customerHandler.AutoSchedule))              // 自动排班
	mux.HandleFunc("/api/schedule/grid", customerHandler.ListScheduleGrid)                                           // 查询排班表格数据
	mux.HandleFunc("/api/schedule/cell/upsert", middleware.CheckAdminPermission(customerHandler.UpsertScheduleCell)) // 更新排班单元格
	mux.HandleFunc("/api/schedule/export", middleware.CheckAdminPermission(customerHandler.ExportScheduleExcel))     // 导出排班Excel

	// ============ 请假/调班管理接口 ============
	mux.HandleFunc("/api/leave/apply", customerHandler.ApplyLeaveTransfer)            // 提交请假/调班申请
	mux.HandleFunc("/api/leave/approve", customerHandler.ApproveLeaveTransfer)        // 审批申请
	mux.HandleFunc("/api/leave/get", customerHandler.GetLeaveTransfer)                // 获取申请详情
	mux.HandleFunc("/api/leave/list", customerHandler.ListLeaveTransfer)              // 查询申请列表
	mux.HandleFunc("/api/leave/audit-log", customerHandler.GetLeaveAuditLog)          // 获取请假审计日志
	mux.HandleFunc("/api/leave/swap-candidates", customerHandler.GetSwapCandidates)   // 获取调班候选人
	mux.HandleFunc("/api/leave/check-conflict", customerHandler.CheckSwapConflict)    // 检测调班冲突
	mux.HandleFunc("/api/leave/chain-swap/apply", customerHandler.ApplyChainSwap)     // 提交链式调班申请
	mux.HandleFunc("/api/leave/chain-swap/approve", customerHandler.ApproveChainSwap) // 审批链式调班申请
	mux.HandleFunc("/api/leave/chain-swap/list", customerHandler.ListChainSwap)       // 查询链式调班列表
	mux.HandleFunc("/api/leave/chain-swap/get", customerHandler.GetChainSwap)         // 获取链式调班详情

	// ============ 心跳与在线状态接口 ============
	mux.HandleFunc("/api/customer/heartbeat", customerHandler.Heartbeat)             // 客服心跳上报
	mux.HandleFunc("/api/customer/online-list", customerHandler.ListOnlineCustomers) // 获取在线客服列表

	// ============ 会话管理接口 ============
	mux.HandleFunc("/api/conversation/create", customerHandler.CreateConversation)            // 创建会话
	mux.HandleFunc("/api/conversation/end", customerHandler.EndConversation)                  // 结束会话
	mux.HandleFunc("/api/conversation/transfer", customerHandler.TransferConversation)        // 转接会话
	mux.HandleFunc("/api/conversation/assign", customerHandler.AssignCustomer)                // 自动分配客服（用户发起咨询时调用）
	mux.HandleFunc("/api/conversation/list", customerHandler.ListConversation)                // 查询会话列表
	mux.HandleFunc("/api/conversation/history/list", customerHandler.ListConversationHistory) // 查询历史会话
	mux.HandleFunc("/api/conversation/message/list", customerHandler.ListConversationMessage) // 查询会话消息
	mux.HandleFunc("/api/conversation/message/send", customerHandler.SendConversationMessage) // 发送消息

	// ============ 快捷回复接口 ============
	mux.HandleFunc("/api/quick_reply/list", customerHandler.ListQuickReply)     // 查询快捷回复列表
	mux.HandleFunc("/api/quick_reply/create", customerHandler.CreateQuickReply) // 创建快捷回复
	mux.HandleFunc("/api/quick_reply/update", customerHandler.UpdateQuickReply) // 更新快捷回复
	mux.HandleFunc("/api/quick_reply/delete", customerHandler.DeleteQuickReply) // 删除快捷回复

	// ============ 会话分类管理接口 ============
	mux.HandleFunc("/api/conversation/category/create", customerHandler.CreateConvCategory)         // 创建会话分类
	mux.HandleFunc("/api/conversation/category/list", customerHandler.ListConvCategory)             // 查询分类列表
	mux.HandleFunc("/api/conversation/classify/update", customerHandler.UpdateConversationClassify) // 更新会话分类标签

	// ============ 会话标签管理接口 ============
	mux.HandleFunc("/api/conversation/tag/create", customerHandler.CreateConvTag) // 创建标签
	mux.HandleFunc("/api/conversation/tag/list", customerHandler.ListConvTag)     // 查询标签列表
	mux.HandleFunc("/api/conversation/tag/update", customerHandler.UpdateConvTag) // 更新标签
	mux.HandleFunc("/api/conversation/tag/delete", customerHandler.DeleteConvTag) // 删除标签

	// ============ 统计看板接口 ============
	mux.HandleFunc("/api/conversation/stats", customerHandler.GetConversationStats) // 获取会话统计数据

	// ============ 会话监控与导出接口 ============
	mux.HandleFunc("/api/conversation/monitor", customerHandler.GetConversationMonitor) // 获取会话监控数据（实时）
	mux.HandleFunc("/api/conversation/export", customerHandler.ExportConversations)     // 导出会话记录

	// ============ 消息分类管理接口 ============
	mux.HandleFunc("/api/msg/classify/auto", customerHandler.MsgAutoClassify)     // 消息自动分类
	mux.HandleFunc("/api/msg/classify/adjust", customerHandler.AdjustMsgClassify) // 人工调整分类
	mux.HandleFunc("/api/msg/classify/stats", customerHandler.GetClassifyStats)   // 分类统计数据

	// ============ 消息分类维度管理接口 ============
	mux.HandleFunc("/api/msg/category/create", customerHandler.CreateMsgCategory) // 创建消息分类维度
	mux.HandleFunc("/api/msg/category/list", customerHandler.ListMsgCategory)     // 查询消息分类维度列表
	mux.HandleFunc("/api/msg/category/update", customerHandler.UpdateMsgCategory) // 更新消息分类维度
	mux.HandleFunc("/api/msg/category/delete", customerHandler.DeleteMsgCategory) // 删除消息分类维度

	// ============ 消息加密与脱敏接口 ============
	mux.HandleFunc("/api/msg/encrypt", customerHandler.EncryptMessage)         // 加密消息内容
	mux.HandleFunc("/api/msg/decrypt", customerHandler.DecryptMessage)         // 解密消息内容
	mux.HandleFunc("/api/msg/desensitize", customerHandler.DesensitizeMessage) // 消息脱敏处理

	// ============ 数据归档管理接口 ============
	mux.HandleFunc("/api/archive/conversations", customerHandler.ArchiveConversations) // 归档历史会话
	mux.HandleFunc("/api/archive/task", customerHandler.GetArchiveTask)                // 获取归档任务状态
	mux.HandleFunc("/api/archive/query", customerHandler.QueryArchivedConversation)    // 查询归档会话

	// ============ AI 助理接口 ============
	mux.HandleFunc("/api/ai/process", customerHandler.ProxyAIProcess)      // AI 处理接口 (转发至 chatModel)
	mux.HandleFunc("/api/ai/job/status", customerHandler.ProxyAIJobStatus) // AI 任务状态查询

	// ============ 系统接口 ============
	// 健康检查：用于探活与部署环境健康监测
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { // 逻辑处理
		w.WriteHeader(http.StatusOK) // 写入HTTP状态码
		w.Write([]byte("OK"))        // 逻辑处理
	}) // 逻辑处理

	return mux // 返回结果/结束处理
} // 代码块结束
