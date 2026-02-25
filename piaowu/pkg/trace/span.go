// Package trace 提供统一的分布式链路追踪功能
// 支持：
// 1. TraceID/SpanID 生成与传播
// 2. Span 生命周期管理（创建、结束、记录属性）
// 3. 多层级调用追踪（parent-child 关系）
// 4. 统一字段记录（service/method/user_id/order_id + 脱敏）
// 5. Jaeger 导出集成（可选）
package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"example_shop/pkg/logger"

	"go.uber.org/zap"
)

// ============ Context Key 定义 ============

// contextKey 是 context 中存储值的键类型
type contextKey string

const (
	// TraceIDKey TraceID 在 context 中的键
	TraceIDKey contextKey = "trace_id"
	// SpanIDKey SpanID 在 context 中的键
	SpanIDKey contextKey = "span_id"
	// ParentSpanIDKey 父 SpanID 在 context 中的键
	ParentSpanIDKey contextKey = "parent_span_id"
	// BaggageKey 链路携带数据的键（用于传递业务上下文）
	BaggageKey contextKey = "trace_baggage"
)

// HTTP Header 常量（用于跨服务传播）
const (
	HTTPHeaderTraceID      = "X-Trace-ID"
	HTTPHeaderSpanID       = "X-Span-ID"
	HTTPHeaderParentSpanID = "X-Parent-Span-ID"
)

// ============ Span 状态定义 ============

// SpanKind 表示 Span 的类型
type SpanKind int

const (
	SpanKindInternal SpanKind = iota // 内部操作
	SpanKindServer                   // 服务端接收请求
	SpanKindClient                   // 客户端发起请求
	SpanKindProducer                 // 消息生产者
	SpanKindConsumer                 // 消息消费者
)

// String 返回 SpanKind 的字符串表示
func (k SpanKind) String() string {
	switch k {
	case SpanKindServer:
		return "server"
	case SpanKindClient:
		return "client"
	case SpanKindProducer:
		return "producer"
	case SpanKindConsumer:
		return "consumer"
	default:
		return "internal"
	}
}

// SpanStatus 表示 Span 的状态
type SpanStatus int

const (
	SpanStatusUnset SpanStatus = iota
	SpanStatusOK
	SpanStatusError
)

// ============ Span 结构定义 ============

// Span 表示一个追踪单元
// 记录一次操作的开始时间、结束时间、属性和状态
type Span struct {
	TraceID      string            // 链路唯一标识
	SpanID       string            // 当前 Span 唯一标识
	ParentSpanID string            // 父 Span ID（可为空）
	Name         string            // Span 名称（如 "HTTP GET /api/user"）
	Kind         SpanKind          // Span 类型
	Service      string            // 服务名称
	StartTime    time.Time         // 开始时间
	EndTime      time.Time         // 结束时间
	Duration     time.Duration     // 持续时间
	Status       SpanStatus        // 状态
	Attributes   map[string]string // 属性（已脱敏）
	Events       []SpanEvent       // 事件列表
	Error        error             // 错误信息
	mu           sync.Mutex        // 保护并发写入
}

// SpanEvent 表示 Span 中的事件
type SpanEvent struct {
	Name       string            // 事件名称
	Timestamp  time.Time         // 事件时间
	Attributes map[string]string // 事件属性
}

// ============ Baggage 定义（链路携带数据）============

// Baggage 链路携带的业务数据（自动传播到下游）
type Baggage struct {
	UserID   string // 用户ID（脱敏后）
	OrderID  string // 订单ID（脱敏后）
	TenantID string // 租户ID
	Extra    map[string]string
}

// ============ ID 生成函数 ============

// NewTraceID 生成新的 TraceID（32字符十六进制）
func NewTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// NewSpanID 生成新的 SpanID（16字符十六进制）
func NewSpanID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ============ Context 操作函数 ============

// WithTraceID 将 TraceID 注入到 context 中
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID 从 context 中获取 TraceID
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(TraceIDKey).(string); ok {
		return v
	}
	return ""
}

// WithSpanID 将 SpanID 注入到 context 中
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// GetSpanID 从 context 中获取 SpanID
func GetSpanID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(SpanIDKey).(string); ok {
		return v
	}
	return ""
}

// WithParentSpanID 将父 SpanID 注入到 context 中
func WithParentSpanID(ctx context.Context, parentSpanID string) context.Context {
	return context.WithValue(ctx, ParentSpanIDKey, parentSpanID)
}

// GetParentSpanID 从 context 中获取父 SpanID
func GetParentSpanID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(ParentSpanIDKey).(string); ok {
		return v
	}
	return ""
}

// WithBaggage 将 Baggage 注入到 context 中
func WithBaggage(ctx context.Context, baggage *Baggage) context.Context {
	return context.WithValue(ctx, BaggageKey, baggage)
}

// GetBaggage 从 context 中获取 Baggage
func GetBaggage(ctx context.Context) *Baggage {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(BaggageKey).(*Baggage); ok {
		return v
	}
	return nil
}

// ============ Span 创建函数 ============

// StartSpan 创建并开始一个新的 Span
// 参数:
//   - ctx: 上下文（用于继承 TraceID 和 ParentSpanID）
//   - name: Span 名称
//   - opts: 可选配置
//
// 返回:
//   - context.Context: 包含新 Span 信息的 context
//   - *Span: 新创建的 Span
func StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
	// 获取或生成 TraceID
	traceID := GetTraceID(ctx)
	if traceID == "" {
		traceID = NewTraceID()
	}

	// 当前 SpanID 成为新 Span 的 ParentSpanID
	parentSpanID := GetSpanID(ctx)

	// 生成新的 SpanID
	spanID := NewSpanID()

	// 创建 Span
	span := &Span{
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		Name:         name,
		Kind:         SpanKindInternal,
		StartTime:    time.Now(),
		Attributes:   make(map[string]string),
		Events:       make([]SpanEvent, 0),
	}

	// 应用选项
	for _, opt := range opts {
		opt(span)
	}

	// 更新 context
	ctx = WithTraceID(ctx, traceID)
	ctx = WithSpanID(ctx, spanID)
	ctx = WithParentSpanID(ctx, parentSpanID)

	return ctx, span
}

// SpanOption Span 配置选项函数类型
type SpanOption func(*Span)

// WithSpanKind 设置 Span 类型
func WithSpanKind(kind SpanKind) SpanOption {
	return func(s *Span) {
		s.Kind = kind
	}
}

// WithService 设置服务名称
func WithService(service string) SpanOption {
	return func(s *Span) {
		s.Service = service
	}
}

// WithAttributes 设置初始属性
func WithAttributes(attrs map[string]string) SpanOption {
	return func(s *Span) {
		for k, v := range attrs {
			s.Attributes[k] = v
		}
	}
}

// ============ Span 操作方法 ============

// SetAttribute 设置 Span 属性
func (s *Span) SetAttribute(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Attributes[key] = value
}

// SetAttributes 批量设置 Span 属性
func (s *Span) SetAttributes(attrs map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range attrs {
		s.Attributes[k] = v
	}
}

// AddEvent 添加事件
func (s *Span) AddEvent(name string, attrs map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, SpanEvent{
		Name:       name,
		Timestamp:  time.Now(),
		Attributes: attrs,
	})
}

// SetStatus 设置 Span 状态
func (s *Span) SetStatus(status SpanStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
}

// SetError 设置错误
func (s *Span) SetError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Error = err
	s.Status = SpanStatusError
}

// End 结束 Span 并记录日志
func (s *Span) End() {
	s.mu.Lock()
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)
	if s.Status == SpanStatusUnset {
		s.Status = SpanStatusOK
	}
	s.mu.Unlock()

	// 记录日志
	s.log()

	// 导出到 Jaeger（如果已配置）
	if globalExporter != nil {
		globalExporter.Export(s)
	}
}

// log 记录 Span 日志
func (s *Span) log() {
	fields := []zap.Field{
		zap.String("trace_id", s.TraceID),
		zap.String("span_id", s.SpanID),
		zap.String("parent_span_id", s.ParentSpanID),
		zap.String("span_name", s.Name),
		zap.String("span_kind", s.Kind.String()),
		zap.String("service", s.Service),
		zap.Duration("duration", s.Duration),
	}

	// 添加属性
	for k, v := range s.Attributes {
		fields = append(fields, zap.String("attr."+k, v))
	}

	// 根据状态记录不同级别日志
	if s.Status == SpanStatusError {
		if s.Error != nil {
			fields = append(fields, zap.Error(s.Error))
		}
		logger.Error("Span End", fields...)
	} else if s.Duration > 200*time.Millisecond {
		// 慢操作警告
		logger.Warn("Span End (slow)", fields...)
	} else {
		logger.Debug("Span End", fields...)
	}
}

// ============ 便捷函数 ============

// StartServerSpan 创建服务端 Span（用于接收请求）
func StartServerSpan(ctx context.Context, service, method string) (context.Context, *Span) {
	return StartSpan(ctx, method,
		WithSpanKind(SpanKindServer),
		WithService(service),
		WithAttributes(map[string]string{
			"method": method,
		}),
	)
}

// StartClientSpan 创建客户端 Span（用于发起请求）
func StartClientSpan(ctx context.Context, service, method string) (context.Context, *Span) {
	return StartSpan(ctx, method,
		WithSpanKind(SpanKindClient),
		WithService(service),
		WithAttributes(map[string]string{
			"method": method,
		}),
	)
}

// StartDBSpan 创建数据库操作 Span
func StartDBSpan(ctx context.Context, operation, table string) (context.Context, *Span) {
	return StartSpan(ctx, "DB "+operation,
		WithSpanKind(SpanKindClient),
		WithService("mysql"),
		WithAttributes(map[string]string{
			"db.operation": operation,
			"db.table":     table,
		}),
	)
}

// StartRedisSpan 创建 Redis 操作 Span
func StartRedisSpan(ctx context.Context, operation, key string) (context.Context, *Span) {
	return StartSpan(ctx, "Redis "+operation,
		WithSpanKind(SpanKindClient),
		WithService("redis"),
		WithAttributes(map[string]string{
			"redis.operation": operation,
			"redis.key":       maskKey(key),
		}),
	)
}

// StartHTTPSpan 创建 HTTP 请求 Span
func StartHTTPSpan(ctx context.Context, method, path string) (context.Context, *Span) {
	return StartSpan(ctx, method+" "+path,
		WithSpanKind(SpanKindClient),
		WithService("http"),
		WithAttributes(map[string]string{
			"http.method": method,
			"http.path":   path,
		}),
	)
}

// ============ 脱敏工具函数 ============

// MaskUserID 脱敏用户ID
// 例: "12345678" -> "1234****"
func MaskUserID(userID string) string {
	if len(userID) <= 4 {
		return userID
	}
	return userID[:4] + "****"
}

// MaskOrderID 脱敏订单ID
// 例: "ORD20240101123456" -> "ORD2024****3456"
func MaskOrderID(orderID string) string {
	if len(orderID) <= 8 {
		return orderID
	}
	return orderID[:7] + "****" + orderID[len(orderID)-4:]
}

// maskKey 脱敏 Redis Key
func maskKey(key string) string {
	// 对包含敏感信息的 key 进行脱敏
	if strings.Contains(key, "user:") || strings.Contains(key, "token:") {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 && len(parts[1]) > 4 {
			return parts[0] + ":" + parts[1][:4] + "****"
		}
	}
	return key
}

// ============ 统一字段设置 ============

// SetBusinessAttrs 设置业务属性（自动脱敏）
// 用于统一记录 user_id、order_id 等业务字段
func (s *Span) SetBusinessAttrs(userID, orderID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if userID != "" {
		s.Attributes["user_id"] = MaskUserID(userID)
	}
	if orderID != "" {
		s.Attributes["order_id"] = MaskOrderID(orderID)
	}
}

// SetHTTPAttrs 设置 HTTP 相关属性
func (s *Span) SetHTTPAttrs(method, path string, statusCode int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Attributes["http.method"] = method
	s.Attributes["http.path"] = path
	s.Attributes["http.status_code"] = string(rune(statusCode))
}

// SetRPCAttrs 设置 RPC 相关属性
func (s *Span) SetRPCAttrs(service, method string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Attributes["rpc.service"] = service
	s.Attributes["rpc.method"] = method
}

// SetDBAttrs 设置数据库相关属性
func (s *Span) SetDBAttrs(operation, table string, rowsAffected int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Attributes["db.operation"] = operation
	s.Attributes["db.table"] = table
	s.Attributes["db.rows_affected"] = string(rune(rowsAffected))
}
