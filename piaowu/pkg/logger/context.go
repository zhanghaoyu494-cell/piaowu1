package logger

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ============ 链路追踪 Context 键定义 ============

// contextKey 是 context 中存储值的键类型
// 使用自定义类型避免键冲突
type contextKey string

const (
	// TraceIDKey TraceID 在 context 中的键
	// 用于在请求链路中传递唯一标识
	TraceIDKey contextKey = "trace_id"
)

// ============ TraceID 生成与管理 ============

// NewTraceID 生成新的 TraceID
// 使用 UUID v4 生成唯一标识
// 返回:
//   - string: UUID 格式的 TraceID
func NewTraceID() string {
	return uuid.New().String()
}

// WithTraceID 将 TraceID 注入到 context 中
// 参数:
//   - ctx: 原始 context
//   - traceID: 要注入的 TraceID
//
// 返回:
//   - context.Context: 包含 TraceID 的新 context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID 从 context 中获取 TraceID
// 参数:
//   - ctx: 包含 TraceID 的 context
//
// 返回:
//   - string: TraceID，不存在返回空字符串
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	traceID, ok := ctx.Value(TraceIDKey).(string)
	if !ok {
		return ""
	}
	return traceID
}

// ============ 带 TraceID 的日志工具函数 ============

// TraceField 返回包含 TraceID 的 zap.Field
// 用于将 TraceID 添加到日志中
// 参数:
//   - ctx: 包含 TraceID 的 context
//
// 返回:
//   - zap.Field: TraceID 字段，TraceID 不存在时返回 Skip 字段
func TraceField(ctx context.Context) zap.Field {
	traceID := GetTraceID(ctx)
	if traceID == "" {
		return zap.Skip()
	}
	return zap.String("trace_id", traceID)
}

// InfoWithTrace 记录带 TraceID 的 Info 日志
// 自动从 context 提取 TraceID 并添加到日志中
// 参数:
//   - ctx: 包含 TraceID 的 context
//   - msg: 日志消息
//   - fields: 可选的结构化字段
func InfoWithTrace(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append([]zap.Field{TraceField(ctx)}, fields...)
	Info(msg, allFields...)
}

// WarnWithTrace 记录带 TraceID 的 Warn 日志
// 自动从 context 提取 TraceID 并添加到日志中
// 参数:
//   - ctx: 包含 TraceID 的 context
//   - msg: 日志消息
//   - fields: 可选的结构化字段
func WarnWithTrace(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append([]zap.Field{TraceField(ctx)}, fields...)
	Warn(msg, allFields...)
}

// ErrorWithTrace 记录带 TraceID 的 Error 日志
// 自动从 context 提取 TraceID 并添加到日志中
// 参数:
//   - ctx: 包含 TraceID 的 context
//   - msg: 日志消息
//   - fields: 可选的结构化字段
func ErrorWithTrace(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append([]zap.Field{TraceField(ctx)}, fields...)
	Error(msg, allFields...)
}

// DebugWithTrace 记录带 TraceID 的 Debug 日志
// 自动从 context 提取 TraceID 并添加到日志中
// 参数:
//   - ctx: 包含 TraceID 的 context
//   - msg: 日志消息
//   - fields: 可选的结构化字段
func DebugWithTrace(ctx context.Context, msg string, fields ...zap.Field) {
	allFields := append([]zap.Field{TraceField(ctx)}, fields...)
	Debug(msg, allFields...)
}
