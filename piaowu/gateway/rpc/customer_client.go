package rpc // 包声明

import ( // 导入依赖列表开始
	"context" // 上下文：超时/取消/链路信息传递
	"fmt"     // 格式化：拼接字符串/错误信息/缓存键
	"net"     // 网络：DNS 解析
	"strings" // 字符串：分割 host:port

	"example_shop/pkg/logger"                                          // 日志工具
	"example_shop/pkg/trace"                                           // 链路追踪
	"example_shop/service/customer/kitex_gen/customer"                 // RPC服务接口定义
	"example_shop/service/customer/kitex_gen/customer/customerservice" // Kitex生成的客户端

	"github.com/cloudwego/kitex/client" // Kitex客户端配置
	"go.uber.org/zap"                   // 依赖导入
) // 导入依赖列表结束/或参数列表结束

// CustomerClient 客服服务RPC客户端封装
// 封装Kitex生成的客户端，提供统一的调用接口
type CustomerClient struct { // 类型/结构体定义
	client      customerservice.Client // Kitex生成的客户端实例
	serviceName string                 // 服务名称（用于追踪）
} // 代码块结束

// resolveAddress 将主机名解析为 IP 地址
// Kitex 的 WithHostPorts 在某些情况下会把纯主机名误判为 Unix socket 路径
// 通过预先 DNS 解析将主机名转换为 IP 地址来解决此问题
func resolveAddress(address string) (string, error) {
	// 分割 host:port
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return address, nil // 无法解析则返回原始地址
	}

	// 如果已经是 IP 地址，直接返回
	if ip := net.ParseIP(host); ip != nil {
		return address, nil
	}

	// DNS 解析主机名
	ips, err := net.LookupIP(host)
	if err != nil {
		return "", fmt.Errorf("failed to resolve hostname %q: %w", host, err)
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("no IP addresses found for hostname %q", host)
	}

	// 优先使用 IPv4 地址
	var resolvedIP string
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			resolvedIP = ipv4.String()
			break
		}
	}
	if resolvedIP == "" {
		resolvedIP = ips[0].String()
	}

	return net.JoinHostPort(resolvedIP, port), nil
}

// NewCustomerClient 创建客服服务RPC客户端
// 参数:
//   - serviceName: 服务名称（用于服务发现）
//   - address: 服务地址（如 "127.0.0.1:9999" 或 "customer-service:8888"）
//
// 返回: 客户端实例和错误信息
func NewCustomerClient(serviceName, address string) (*CustomerClient, error) { // 函数定义/HTTP处理入口
	if address == "" { // 条件判断
		return nil, fmt.Errorf("address is required") // 返回结果/结束处理
	} // 代码块结束
	if serviceName == "" { // 条件判断
		serviceName = "CustomerService" // 默认服务名
	} // 代码块结束

	// 解析主机名为 IP 地址，避免 Kitex 误判为 Unix socket
	resolvedAddr, err := resolveAddress(address)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address %q: %w", address, err)
	}

	// 如果地址发生变化，记录日志
	if resolvedAddr != address && !strings.HasPrefix(address, "127.") {
		logger.Info("Resolved service address",
			zap.String("original", address),
			zap.String("resolved", resolvedAddr),
		)
	}

	// 创建Kitex客户端实例
	c, err := customerservice.NewClient( // 调用并接收错误
		serviceName,                        // 逻辑处理
		client.WithHostPorts(resolvedAddr), // 指定解析后的服务地址
	) // 导入依赖列表结束/或参数列表结束
	if err != nil { // 条件判断
		return nil, fmt.Errorf("create kitex client failed (service=%q address=%q resolved=%q): %w", serviceName, address, resolvedAddr, err) // 返回结果/结束处理
	} // 代码块结束
	return &CustomerClient{ // 返回结果/结束处理
		client:      c,           // 逻辑处理
		serviceName: serviceName, // 逻辑处理
	}, nil // 逻辑处理
} // 代码块结束

// wrapContext 包装context，创建 Client Span 并添加RPC调用日志
// 在每次RPC调用前创建 span，记录方法名和TraceID
// 返回：新的 context 和 span（调用方需要在 RPC 调用后调用 span.End()）
func (c *CustomerClient) wrapContext(ctx context.Context, method string) (context.Context, *trace.Span) { // 函数定义/HTTP处理入口
	// 创建 Client Span
	ctx, span := trace.StartClientSpan(ctx, c.serviceName, method) // 逻辑处理

	// 记录日志（兼容旧代码）
	traceID := trace.GetTraceID(ctx) // 逻辑处理
	if traceID != "" {               // 条件判断
		logger.InfoWithTrace(ctx, "RPC Call", // 逻辑处理
			zap.String("method", method),         // 逻辑处理
			zap.String("service", c.serviceName), // 逻辑处理
		) // 导入依赖列表结束/或参数列表结束
	} // 代码块结束

	return ctx, span // 返回结果/结束处理
} // 代码块结束

// ==================== 客服基础信息接口 ====================

// GetCustomerService 获取客服信息
func (c *CustomerClient) GetCustomerService(ctx context.Context, req *customer.GetCustomerServiceReq) (*customer.GetCustomerServiceResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "GetCustomerService") // 逻辑处理
	defer span.End()                                      // 延迟执行清理逻辑
	resp, err := c.client.GetCustomerService(ctx, req)    // 调用并接收错误
	if err != nil {                                       // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ListCustomerService 查询客服列表
func (c *CustomerClient) ListCustomerService(ctx context.Context, req *customer.ListCustomerServiceReq) (*customer.ListCustomerServiceResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ListCustomerService") // 逻辑处理
	defer span.End()                                       // 延迟执行清理逻辑
	resp, err := c.client.ListCustomerService(ctx, req)    // 调用并接收错误
	if err != nil {                                        // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 班次配置接口 ====================

// CreateShiftConfig 创建班次配置
func (c *CustomerClient) CreateShiftConfig(ctx context.Context, req *customer.CreateShiftConfigReq) (*customer.CreateShiftConfigResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "CreateShiftConfig") // 逻辑处理
	defer span.End()                                     // 延迟执行清理逻辑
	resp, err := c.client.CreateShiftConfig(ctx, req)    // 调用并接收错误
	if err != nil {                                      // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ListShiftConfig 查询班次配置列表
func (c *CustomerClient) ListShiftConfig(ctx context.Context, req *customer.ListShiftConfigReq) (*customer.ListShiftConfigResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ListShiftConfig") // 逻辑处理
	defer span.End()                                   // 延迟执行清理逻辑
	resp, err := c.client.ListShiftConfig(ctx, req)    // 调用并接收错误
	if err != nil {                                    // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// UpdateShiftConfig 更新班次配置
func (c *CustomerClient) UpdateShiftConfig(ctx context.Context, req *customer.UpdateShiftConfigReq) (*customer.UpdateShiftConfigResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "UpdateShiftConfig") // 逻辑处理
	defer span.End()                                     // 延迟执行清理逻辑
	resp, err := c.client.UpdateShiftConfig(ctx, req)    // 调用并接收错误
	if err != nil {                                      // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// DeleteShiftConfig 删除班次配置
func (c *CustomerClient) DeleteShiftConfig(ctx context.Context, req *customer.DeleteShiftConfigReq) (*customer.DeleteShiftConfigResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "DeleteShiftConfig") // 逻辑处理
	defer span.End()                                     // 延迟执行清理逻辑
	resp, err := c.client.DeleteShiftConfig(ctx, req)    // 调用并接收错误
	if err != nil {                                      // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 排班管理接口 ====================

// AssignSchedule 手动分配排班
func (c *CustomerClient) AssignSchedule(ctx context.Context, req *customer.AssignScheduleReq) (*customer.AssignScheduleResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "AssignSchedule") // 逻辑处理
	defer span.End()                                  // 延迟执行清理逻辑
	resp, err := c.client.AssignSchedule(ctx, req)    // 调用并接收错误
	if err != nil {                                   // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// AutoSchedule 自动排班
func (c *CustomerClient) AutoSchedule(ctx context.Context, req *customer.AutoScheduleReq) (*customer.AutoScheduleResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "AutoSchedule") // 逻辑处理
	defer span.End()                                // 延迟执行清理逻辑
	resp, err := c.client.AutoSchedule(ctx, req)    // 调用并接收错误
	if err != nil {                                 // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 请假/调班接口 ====================

// ApplyLeaveTransfer 提交请假/调班申请
func (c *CustomerClient) ApplyLeaveTransfer(ctx context.Context, req *customer.ApplyLeaveTransferReq) (*customer.ApplyLeaveTransferResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ApplyLeaveTransfer") // 逻辑处理
	defer span.End()                                      // 延迟执行清理逻辑
	resp, err := c.client.ApplyLeaveTransfer(ctx, req)    // 调用并接收错误
	if err != nil {                                       // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ApproveLeaveTransfer 审批请假/调班申请
func (c *CustomerClient) ApproveLeaveTransfer(ctx context.Context, req *customer.ApproveLeaveTransferReq) (*customer.ApproveLeaveTransferResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ApproveLeaveTransfer") // 逻辑处理
	defer span.End()                                        // 延迟执行清理逻辑
	resp, err := c.client.ApproveLeaveTransfer(ctx, req)    // 调用并接收错误
	if err != nil {                                         // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ApplyChainSwap 提交链式调班申请
func (c *CustomerClient) ApplyChainSwap(ctx context.Context, req *customer.ApplyChainSwapReq) (*customer.ApplyChainSwapResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ApplyChainSwap") // 逻辑处理
	defer span.End()                                  // 延迟执行清理逻辑
	resp, err := c.client.ApplyChainSwap(ctx, req)    // 调用并接收错误
	if err != nil {                                   // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ApproveChainSwap 审批链式调班申请
func (c *CustomerClient) ApproveChainSwap(ctx context.Context, req *customer.ApproveChainSwapReq) (*customer.ApproveChainSwapResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ApproveChainSwap") // 逻辑处理
	defer span.End()                                    // 延迟执行清理逻辑
	resp, err := c.client.ApproveChainSwap(ctx, req)    // 调用并接收错误
	if err != nil {                                     // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ListChainSwap 查询链式调班申请列表
func (c *CustomerClient) ListChainSwap(ctx context.Context, req *customer.ListChainSwapReq) (*customer.ListChainSwapResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ListChainSwap") // 逻辑处理
	defer span.End()                                 // 延迟执行清理逻辑
	resp, err := c.client.ListChainSwap(ctx, req)    // 调用并接收错误
	if err != nil {                                  // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// GetChainSwap 获取链式调班申请详情
func (c *CustomerClient) GetChainSwap(ctx context.Context, req *customer.GetChainSwapReq) (*customer.GetChainSwapResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "GetChainSwap") // 逻辑处理
	defer span.End()                                // 延迟执行清理逻辑
	resp, err := c.client.GetChainSwap(ctx, req)    // 调用并接收错误
	if err != nil {                                 // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// GetLeaveTransfer 获取请假/调班申请详情
func (c *CustomerClient) GetLeaveTransfer(ctx context.Context, req *customer.GetLeaveTransferReq) (*customer.GetLeaveTransferResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "GetLeaveTransfer") // 逻辑处理
	defer span.End()                                    // 延迟执行清理逻辑
	resp, err := c.client.GetLeaveTransfer(ctx, req)    // 调用并接收错误
	if err != nil {                                     // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ListLeaveTransfer 查询请假/调班申请列表
func (c *CustomerClient) ListLeaveTransfer(ctx context.Context, req *customer.ListLeaveTransferReq) (*customer.ListLeaveTransferResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ListLeaveTransfer") // 逻辑处理
	defer span.End()                                     // 延迟执行清理逻辑
	resp, err := c.client.ListLeaveTransfer(ctx, req)    // 调用并接收错误
	if err != nil {                                      // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// GetLeaveAuditLog 获取请假/调班申请的审计日志
func (c *CustomerClient) GetLeaveAuditLog(ctx context.Context, req *customer.GetLeaveAuditLogReq) (*customer.GetLeaveAuditLogResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "GetLeaveAuditLog") // 逻辑处理
	defer span.End()                                    // 延迟执行清理逻辑
	resp, err := c.client.GetLeaveAuditLog(ctx, req)    // 调用并接收错误
	if err != nil {                                     // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 排班表格接口 ====================

// ListScheduleGrid 查询排班表格数据
func (c *CustomerClient) ListScheduleGrid(ctx context.Context, req *customer.ListScheduleGridReq) (*customer.ListScheduleGridResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ListScheduleGrid") // 逻辑处理
	defer span.End()                                    // 延迟执行清理逻辑
	resp, err := c.client.ListScheduleGrid(ctx, req)    // 调用并接收错误
	if err != nil {                                     // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// UpsertScheduleCell 更新排班单元格
func (c *CustomerClient) UpsertScheduleCell(ctx context.Context, req *customer.UpsertScheduleCellReq) (*customer.UpsertScheduleCellResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "UpsertScheduleCell") // 逻辑处理
	defer span.End()                                      // 延迟执行清理逻辑
	resp, err := c.client.UpsertScheduleCell(ctx, req)    // 调用并接收错误
	if err != nil {                                       // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 会话管理接口 ====================

// AssignCustomer 自动分配客服
func (c *CustomerClient) AssignCustomer(ctx context.Context, req *customer.AssignCustomerReq) (*customer.AssignCustomerResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "AssignCustomer") // 逻辑处理
	defer span.End()                                  // 延迟执行清理逻辑
	resp, err := c.client.AssignCustomer(ctx, req)    // 调用并接收错误
	if err != nil {                                   // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// CreateConversation 创建会话
// 支持用户发起新会话，可指定客服或自动分配
func (c *CustomerClient) CreateConversation(ctx context.Context, req *customer.CreateConversationReq) (*customer.CreateConversationResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "CreateConversation") // 逻辑处理
	defer span.End()                                      // 延迟执行清理逻辑
	// 设置业务属性（自动脱敏）
	if req.UserId != "" { // 条件判断
		span.SetBusinessAttrs(req.UserId, "") // 逻辑处理
	} // 代码块结束
	resp, err := c.client.CreateConversation(ctx, req) // 调用并接收错误
	if err != nil {                                    // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// EndConversation 结束会话
// 由客服或系统主动结束会话，支持乐观锁并发控制
func (c *CustomerClient) EndConversation(ctx context.Context, req *customer.EndConversationReq) (*customer.EndConversationResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "EndConversation") // 逻辑处理
	defer span.End()                                   // 延迟执行清理逻辑
	resp, err := c.client.EndConversation(ctx, req)    // 调用并接收错误
	if err != nil {                                    // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// TransferConversation 转接会话
// 将会话从当前客服转接给另一位客服，支持传递上下文备注
func (c *CustomerClient) TransferConversation(ctx context.Context, req *customer.TransferConversationReq) (*customer.TransferConversationResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "TransferConversation") // 逻辑处理
	defer span.End()                                        // 延迟执行清理逻辑
	resp, err := c.client.TransferConversation(ctx, req)    // 调用并接收错误
	if err != nil {                                         // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ListConversation 查询会话列表
func (c *CustomerClient) ListConversation(ctx context.Context, req *customer.ListConversationReq) (*customer.ListConversationResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ListConversation") // 逻辑处理
	defer span.End()                                    // 延迟执行清理逻辑
	resp, err := c.client.ListConversation(ctx, req)    // 调用并接收错误
	if err != nil {                                     // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ListConversationHistory 查询历史会话记录
func (c *CustomerClient) ListConversationHistory(ctx context.Context, req *customer.ListConversationHistoryReq) (*customer.ListConversationResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ListConversationHistory") // 逻辑处理
	defer span.End()                                           // 延迟执行清理逻辑
	resp, err := c.client.ListConversationHistory(ctx, req)    // 调用并接收错误
	if err != nil {                                            // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ListConversationMessage 查询会话消息列表
func (c *CustomerClient) ListConversationMessage(ctx context.Context, req *customer.ListConversationMessageReq) (*customer.ListConversationMessageResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ListConversationMessage") // 逻辑处理
	defer span.End()                                           // 延迟执行清理逻辑
	resp, err := c.client.ListConversationMessage(ctx, req)    // 调用并接收错误
	if err != nil {                                            // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// SendConversationMessage 发送会话消息
func (c *CustomerClient) SendConversationMessage(ctx context.Context, req *customer.SendConversationMessageReq) (*customer.SendConversationMessageResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "SendConversationMessage") // 逻辑处理
	defer span.End()                                           // 延迟执行清理逻辑
	resp, err := c.client.SendConversationMessage(ctx, req)    // 调用并接收错误
	if err != nil {                                            // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ListQuickReply 查询快捷回复列表
func (c *CustomerClient) ListQuickReply(ctx context.Context, req *customer.ListQuickReplyReq) (*customer.ListQuickReplyResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ListQuickReply") // 逻辑处理
	defer span.End()                                  // 延迟执行清理逻辑
	resp, err := c.client.ListQuickReply(ctx, req)    // 调用并接收错误
	if err != nil {                                   // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// CreateQuickReply 创建快捷回复
func (c *CustomerClient) CreateQuickReply(ctx context.Context, req *customer.CreateQuickReplyReq) (*customer.CreateQuickReplyResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "CreateQuickReply") // 逻辑处理
	defer span.End()                                    // 延迟执行清理逻辑
	resp, err := c.client.CreateQuickReply(ctx, req)    // 调用并接收错误
	if err != nil {                                     // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// UpdateQuickReply 更新快捷回复
func (c *CustomerClient) UpdateQuickReply(ctx context.Context, req *customer.UpdateQuickReplyReq) (*customer.UpdateQuickReplyResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "UpdateQuickReply") // 逻辑处理
	defer span.End()                                    // 延迟执行清理逻辑
	resp, err := c.client.UpdateQuickReply(ctx, req)    // 调用并接收错误
	if err != nil {                                     // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// DeleteQuickReply 删除快捷回复
func (c *CustomerClient) DeleteQuickReply(ctx context.Context, req *customer.DeleteQuickReplyReq) (*customer.DeleteQuickReplyResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "DeleteQuickReply") // 逻辑处理
	defer span.End()                                    // 延迟执行清理逻辑
	resp, err := c.client.DeleteQuickReply(ctx, req)    // 调用并接收错误
	if err != nil {                                     // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 会话分类接口 ====================

// CreateConvCategory 创建会话分类
func (c *CustomerClient) CreateConvCategory(ctx context.Context, req *customer.CreateConvCategoryReq) (*customer.CreateConvCategoryResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "CreateConvCategory") // 逻辑处理
	defer span.End()                                      // 延迟执行清理逻辑
	resp, err := c.client.CreateConvCategory(ctx, req)    // 调用并接收错误
	if err != nil {                                       // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ListConvCategory 查询会话分类列表
func (c *CustomerClient) ListConvCategory(ctx context.Context, req *customer.ListConvCategoryReq) (*customer.ListConvCategoryResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ListConvCategory") // 逻辑处理
	defer span.End()                                    // 延迟执行清理逻辑
	resp, err := c.client.ListConvCategory(ctx, req)    // 调用并接收错误
	if err != nil {                                     // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// UpdateConversationClassify 更新会话分类/标签
func (c *CustomerClient) UpdateConversationClassify(ctx context.Context, req *customer.UpdateConversationClassifyReq) (*customer.UpdateConversationClassifyResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "UpdateConversationClassify") // 逻辑处理
	defer span.End()                                              // 延迟执行清理逻辑
	resp, err := c.client.UpdateConversationClassify(ctx, req)    // 调用并接收错误
	if err != nil {                                               // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 会话标签接口 ====================

// CreateConvTag 创建会话标签
func (c *CustomerClient) CreateConvTag(ctx context.Context, req *customer.CreateConvTagReq) (*customer.CreateConvTagResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "CreateConvTag") // 逻辑处理
	defer span.End()                                 // 延迟执行清理逻辑
	resp, err := c.client.CreateConvTag(ctx, req)    // 调用并接收错误
	if err != nil {                                  // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ListConvTag 查询会话标签列表
func (c *CustomerClient) ListConvTag(ctx context.Context, req *customer.ListConvTagReq) (*customer.ListConvTagResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ListConvTag") // 逻辑处理
	defer span.End()                               // 延迟执行清理逻辑
	resp, err := c.client.ListConvTag(ctx, req)    // 调用并接收错误
	if err != nil {                                // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// UpdateConvTag 更新会话标签
func (c *CustomerClient) UpdateConvTag(ctx context.Context, req *customer.UpdateConvTagReq) (*customer.UpdateConvTagResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "UpdateConvTag") // 逻辑处理
	defer span.End()                                 // 延迟执行清理逻辑
	resp, err := c.client.UpdateConvTag(ctx, req)    // 调用并接收错误
	if err != nil {                                  // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// DeleteConvTag 删除会话标签
func (c *CustomerClient) DeleteConvTag(ctx context.Context, req *customer.DeleteConvTagReq) (*customer.DeleteConvTagResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "DeleteConvTag") // 逻辑处理
	defer span.End()                                 // 延迟执行清理逻辑
	resp, err := c.client.DeleteConvTag(ctx, req)    // 调用并接收错误
	if err != nil {                                  // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 统计接口 ====================

// GetConversationStats 获取会话统计数据
func (c *CustomerClient) GetConversationStats(ctx context.Context, req *customer.GetConversationStatsReq) (*customer.GetConversationStatsResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "GetConversationStats") // 逻辑处理
	defer span.End()                                        // 延迟执行清理逻辑
	resp, err := c.client.GetConversationStats(ctx, req)    // 调用并接收错误
	if err != nil {                                         // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 会话监控与导出接口 ====================

// GetConversationMonitor 获取会话监控数据
// 实时查看会话状态、客服在线状态、等待中会话数等
func (c *CustomerClient) GetConversationMonitor(ctx context.Context, req *customer.GetConversationMonitorReq) (*customer.GetConversationMonitorResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "GetConversationMonitor") // 逻辑处理
	defer span.End()                                          // 延迟执行清理逻辑
	resp, err := c.client.GetConversationMonitor(ctx, req)    // 调用并接收错误
	if err != nil {                                           // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ExportConversations 导出会话记录
// 支持按条件筛选导出会话记录为Excel/CSV格式
func (c *CustomerClient) ExportConversations(ctx context.Context, req *customer.ExportConversationsReq) (*customer.ExportConversationsResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ExportConversations") // 逻辑处理
	defer span.End()                                       // 延迟执行清理逻辑
	resp, err := c.client.ExportConversations(ctx, req)    // 调用并接收错误
	if err != nil {                                        // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 消息分类管理接口 ====================

// MsgAutoClassify 消息自动分类
// 基于关键词匹配对会话消息进行自动分类
func (c *CustomerClient) MsgAutoClassify(ctx context.Context, req *customer.MsgAutoClassifyReq) (*customer.MsgAutoClassifyResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "MsgAutoClassify") // 逻辑处理
	defer span.End()                                   // 延迟执行清理逻辑
	resp, err := c.client.MsgAutoClassify(ctx, req)    // 调用并接收错误
	if err != nil {                                    // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// AdjustMsgClassify 人工调整消息分类
// 客服手动修正自动分类结果
func (c *CustomerClient) AdjustMsgClassify(ctx context.Context, req *customer.AdjustMsgClassifyReq) (*customer.AdjustMsgClassifyResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "AdjustMsgClassify") // 逻辑处理
	defer span.End()                                     // 延迟执行清理逻辑
	resp, err := c.client.AdjustMsgClassify(ctx, req)    // 调用并接收错误
	if err != nil {                                      // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// GetClassifyStats 获取分类统计数据
// 查询消息分类的统计信息，支持按日期范围和统计类型筛选
func (c *CustomerClient) GetClassifyStats(ctx context.Context, req *customer.GetClassifyStatsReq) (*customer.GetClassifyStatsResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "GetClassifyStats") // 逻辑处理
	defer span.End()                                    // 延迟执行清理逻辑
	resp, err := c.client.GetClassifyStats(ctx, req)    // 调用并接收错误
	if err != nil {                                     // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 消息分类维度CRUD ====================

// CreateMsgCategory 创建消息分类维度
func (c *CustomerClient) CreateMsgCategory(ctx context.Context, req *customer.CreateMsgCategoryReq) (*customer.CreateMsgCategoryResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "CreateMsgCategory") // 逻辑处理
	defer span.End()                                     // 延迟执行清理逻辑
	resp, err := c.client.CreateMsgCategory(ctx, req)    // 调用并接收错误
	if err != nil {                                      // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ListMsgCategory 查询消息分类维度列表
func (c *CustomerClient) ListMsgCategory(ctx context.Context, req *customer.ListMsgCategoryReq) (*customer.ListMsgCategoryResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ListMsgCategory") // 逻辑处理
	defer span.End()                                   // 延迟执行清理逻辑
	resp, err := c.client.ListMsgCategory(ctx, req)    // 调用并接收错误
	if err != nil {                                    // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// UpdateMsgCategory 更新消息分类维度
func (c *CustomerClient) UpdateMsgCategory(ctx context.Context, req *customer.UpdateMsgCategoryReq) (*customer.UpdateMsgCategoryResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "UpdateMsgCategory") // 逻辑处理
	defer span.End()                                     // 延迟执行清理逻辑
	resp, err := c.client.UpdateMsgCategory(ctx, req)    // 调用并接收错误
	if err != nil {                                      // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// DeleteMsgCategory 删除消息分类维度
func (c *CustomerClient) DeleteMsgCategory(ctx context.Context, req *customer.DeleteMsgCategoryReq) (*customer.DeleteMsgCategoryResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "DeleteMsgCategory") // 逻辑处理
	defer span.End()                                     // 延迟执行清理逻辑
	resp, err := c.client.DeleteMsgCategory(ctx, req)    // 调用并接收错误
	if err != nil {                                      // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 用户认证接口 ====================

// Login 用户登录
func (c *CustomerClient) Login(ctx context.Context, req *customer.LoginReq) (*customer.LoginResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "Login") // 逻辑处理
	defer span.End()                         // 延迟执行清理逻辑
	resp, err := c.client.Login(ctx, req)    // 调用并接收错误
	if err != nil {                          // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// GetCurrentUser 获取当前登录用户信息
func (c *CustomerClient) GetCurrentUser(ctx context.Context, req *customer.GetCurrentUserReq) (*customer.GetCurrentUserResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "GetCurrentUser") // 逻辑处理
	defer span.End()                                  // 延迟执行清理逻辑
	resp, err := c.client.GetCurrentUser(ctx, req)    // 调用并接收错误
	if err != nil {                                   // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// Register 用户注册
func (c *CustomerClient) Register(ctx context.Context, req *customer.RegisterReq) (*customer.RegisterResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "Register") // 逻辑处理
	defer span.End()                            // 延迟执行清理逻辑
	resp, err := c.client.Register(ctx, req)    // 调用并接收错误
	if err != nil {                             // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// Logout 用户退出登录
// 更新客服在线状态为离线
func (c *CustomerClient) Logout(ctx context.Context, req *customer.LogoutReq) (*customer.LogoutResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "Logout") // 逻辑处理
	defer span.End()                          // 延迟执行清理逻辑
	resp, err := c.client.Logout(ctx, req)    // 调用并接收错误
	if err != nil {                           // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 消息加密与脱敏接口 ====================

// EncryptMessage 加密消息内容
func (c *CustomerClient) EncryptMessage(ctx context.Context, req *customer.EncryptMessageReq) (*customer.EncryptMessageResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "EncryptMessage") // 逻辑处理
	defer span.End()                                  // 延迟执行清理逻辑
	resp, err := c.client.EncryptMessage(ctx, req)    // 调用并接收错误
	if err != nil {                                   // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// DecryptMessage 解密消息内容
func (c *CustomerClient) DecryptMessage(ctx context.Context, req *customer.DecryptMessageReq) (*customer.DecryptMessageResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "DecryptMessage") // 逻辑处理
	defer span.End()                                  // 延迟执行清理逻辑
	resp, err := c.client.DecryptMessage(ctx, req)    // 调用并接收错误
	if err != nil {                                   // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// DesensitizeMessage 消息脱敏处理
func (c *CustomerClient) DesensitizeMessage(ctx context.Context, req *customer.DesensitizeMessageReq) (*customer.DesensitizeMessageResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "DesensitizeMessage") // 逻辑处理
	defer span.End()                                      // 延迟执行清理逻辑
	resp, err := c.client.DesensitizeMessage(ctx, req)    // 调用并接收错误
	if err != nil {                                       // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 数据归档管理接口 ====================

// ArchiveConversations 归档历史会话数据
func (c *CustomerClient) ArchiveConversations(ctx context.Context, req *customer.ArchiveConversationsReq) (*customer.ArchiveConversationsResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ArchiveConversations") // 逻辑处理
	defer span.End()                                        // 延迟执行清理逻辑
	resp, err := c.client.ArchiveConversations(ctx, req)    // 调用并接收错误
	if err != nil {                                         // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// GetArchiveTask 查询归档任务状态
func (c *CustomerClient) GetArchiveTask(ctx context.Context, req *customer.GetArchiveTaskReq) (*customer.GetArchiveTaskResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "GetArchiveTask") // 逻辑处理
	defer span.End()                                  // 延迟执行清理逻辑
	resp, err := c.client.GetArchiveTask(ctx, req)    // 调用并接收错误
	if err != nil {                                   // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// QueryArchivedConversation 查询已归档会话
func (c *CustomerClient) QueryArchivedConversation(ctx context.Context, req *customer.QueryArchivedConversationReq) (*customer.QueryArchivedConversationResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "QueryArchivedConversation") // 逻辑处理
	defer span.End()                                             // 延迟执行清理逻辑
	resp, err := c.client.QueryArchivedConversation(ctx, req)    // 调用并接收错误
	if err != nil {                                              // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 心跳与在线状态接口 ====================

// Heartbeat 客服心跳上报
func (c *CustomerClient) Heartbeat(ctx context.Context, req *customer.HeartbeatReq) (*customer.HeartbeatResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "Heartbeat") // 逻辑处理
	defer span.End()                             // 延迟执行清理逻辑
	resp, err := c.client.Heartbeat(ctx, req)    // 调用并接收错误
	if err != nil {                              // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ListOnlineCustomers 获取在线客服列表
func (c *CustomerClient) ListOnlineCustomers(ctx context.Context, req *customer.ListOnlineCustomersReq) (*customer.ListOnlineCustomersResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "ListOnlineCustomers") // 逻辑处理
	defer span.End()                                       // 延迟执行清理逻辑
	resp, err := c.client.ListOnlineCustomers(ctx, req)    // 调用并接收错误
	if err != nil {                                        // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// ==================== 调班辅助接口 ====================

// GetSwapCandidates 获取调班候选人
func (c *CustomerClient) GetSwapCandidates(ctx context.Context, req *customer.GetSwapCandidatesReq) (*customer.GetSwapCandidatesResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "GetSwapCandidates") // 逻辑处理
	defer span.End()                                     // 延迟执行清理逻辑
	resp, err := c.client.GetSwapCandidates(ctx, req)    // 调用并接收错误
	if err != nil {                                      // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// CheckSwapConflict 检测调班冲突
func (c *CustomerClient) CheckSwapConflict(ctx context.Context, req *customer.CheckSwapConflictReq) (*customer.CheckSwapConflictResp, error) { // 函数定义/HTTP处理入口
	ctx, span := c.wrapContext(ctx, "CheckSwapConflict") // 逻辑处理
	defer span.End()                                     // 延迟执行清理逻辑
	resp, err := c.client.CheckSwapConflict(ctx, req)    // 调用并接收错误
	if err != nil {                                      // 条件判断
		span.SetError(err) // 逻辑处理
	} // 代码块结束
	return resp, err // 返回结果/结束处理
} // 代码块结束

// Close 关闭RPC客户端连接
// 当前实现为空操作，Kitex客户端无需显式关闭
func (c *CustomerClient) Close() error { // 函数定义/HTTP处理入口
	return nil // 返回结果/结束处理
} // 代码块结束
