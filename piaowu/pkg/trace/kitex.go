// Package trace 提供 Kitex RPC 中间件
// 功能：
// 1. Server 端: 从请求中提取 TraceID/SpanID，创建 Server Span
// 2. Client 端: 注入 TraceID/SpanID 到请求中，创建 Client Span
package trace

import (
	"context"
	"time"

	"example_shop/pkg/logger"

	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"go.uber.org/zap"
)

// ============ Kitex MetaInfo Key 定义 ============

const (
	// MetaKeyTraceID Kitex metainfo 中 TraceID 的键
	MetaKeyTraceID = "trace-id"
	// MetaKeySpanID Kitex metainfo 中 SpanID 的键
	MetaKeySpanID = "span-id"
	// MetaKeyParentSpanID Kitex metainfo 中 ParentSpanID 的键
	MetaKeyParentSpanID = "parent-span-id"
	// MetaKeyUserID Kitex metainfo 中 UserID 的键（已脱敏）
	MetaKeyUserID = "user-id"
)

// ============ Server 端中间件 ============

// ServerTraceMiddleware Kitex Server 端链路追踪中间件
// 功能：
// 1. 从上游请求中提取 TraceID/SpanID
// 2. 创建 Server Span
// 3. 记录请求信息和耗时
// 4. 在请求结束时结束 Span
func ServerTraceMiddleware(serviceName string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req, resp interface{}) (err error) {
			startTime := time.Now()

			// 获取 RPC 信息
			ri := rpcinfo.GetRPCInfo(ctx)
			var methodName string
			if ri != nil && ri.To() != nil {
				methodName = ri.To().Method()
			}

			// 1. 从上游提取 TraceID（如果有）
			traceID := extractMetaValue(ctx, MetaKeyTraceID)
			if traceID == "" {
				traceID = NewTraceID() // 生成新的 TraceID
			}

			// 2. 从上游提取 ParentSpanID
			parentSpanID := extractMetaValue(ctx, MetaKeySpanID)

			// 3. 生成新的 SpanID
			spanID := NewSpanID()

			// 4. 创建 Server Span
			ctx, span := StartSpan(ctx, methodName,
				WithSpanKind(SpanKindServer),
				WithService(serviceName),
			)

			// 覆盖自动生成的 ID（使用提取的上游 ID）
			span.TraceID = traceID
			span.SpanID = spanID
			span.ParentSpanID = parentSpanID

			// 5. 设置 RPC 属性
			span.SetRPCAttrs(serviceName, methodName)

			// 6. 提取并设置 UserID（如果有）
			if userID := extractMetaValue(ctx, MetaKeyUserID); userID != "" {
				span.SetAttribute("user_id", userID) // 已在上游脱敏
			}

			// 7. 将追踪信息注入 context
			ctx = WithTraceID(ctx, traceID)
			ctx = WithSpanID(ctx, spanID)
			ctx = WithParentSpanID(ctx, parentSpanID)

			// 记录请求开始
			logger.InfoWithTrace(ctx, "RPC Server Request Start",
				zap.String("service", serviceName),
				zap.String("method", methodName),
			)

			// 8. 执行业务逻辑
			defer func() {
				duration := time.Since(startTime)

				// 记录结果
				if err != nil {
					span.SetError(err)
					logger.ErrorWithTrace(ctx, "RPC Server Request Error",
						zap.String("service", serviceName),
						zap.String("method", methodName),
						zap.Duration("duration", duration),
						zap.Error(err),
					)
				} else {
					span.SetStatus(SpanStatusOK)
					logger.InfoWithTrace(ctx, "RPC Server Request End",
						zap.String("service", serviceName),
						zap.String("method", methodName),
						zap.Duration("duration", duration),
					)
				}

				// 结束 Span
				span.End()
			}()

			return next(ctx, req, resp)
		}
	}
}

// extractMetaValue 从 context 中提取 metainfo 值
// Kitex 使用 metainfo 包传递元数据
func extractMetaValue(ctx context.Context, key string) string {
	// 尝试从 context 获取
	if v := GetTraceID(ctx); key == MetaKeyTraceID && v != "" {
		return v
	}
	if v := GetSpanID(ctx); key == MetaKeySpanID && v != "" {
		return v
	}
	if v := GetParentSpanID(ctx); key == MetaKeyParentSpanID && v != "" {
		return v
	}

	// 从 Kitex rpcinfo 获取（上游传递的）
	ri := rpcinfo.GetRPCInfo(ctx)
	if ri == nil {
		return ""
	}

	// 尝试从 TransInfo 获取
	if ri.Invocation() != nil {
		// Kitex 的 transient info 机制
		// 实际项目中应使用 github.com/cloudwego/kitex/pkg/transmeta
		// 这里简化处理
	}

	return ""
}

// ============ Client 端中间件 ============

// ClientTraceMiddleware Kitex Client 端链路追踪中间件
// 功能：
// 1. 从当前 context 获取 TraceID/SpanID
// 2. 创建 Client Span
// 3. 将追踪信息注入到下游请求
// 4. 记录调用信息和耗时
func ClientTraceMiddleware(serviceName string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req, resp interface{}) (err error) {
			startTime := time.Now()

			// 获取 RPC 信息
			ri := rpcinfo.GetRPCInfo(ctx)
			var methodName string
			var targetService string
			if ri != nil {
				if ri.To() != nil {
					methodName = ri.To().Method()
					targetService = ri.To().ServiceName()
				}
			}
			if targetService == "" {
				targetService = serviceName
			}

			// 1. 创建 Client Span（继承当前 context 的 TraceID）
			ctx, span := StartSpan(ctx, methodName,
				WithSpanKind(SpanKindClient),
				WithService(targetService),
			)

			// 2. 设置 RPC 属性
			span.SetRPCAttrs(targetService, methodName)

			// 3. 注入追踪信息到 context（用于下游传播）
			// 在实际项目中，应使用 github.com/cloudwego/kitex/pkg/transmeta
			// 将 TraceID 等信息注入到 RPC 请求的 TransInfo 中

			// 记录调用开始
			logger.InfoWithTrace(ctx, "RPC Client Call Start",
				zap.String("service", targetService),
				zap.String("method", methodName),
			)

			// 4. 执行 RPC 调用
			defer func() {
				duration := time.Since(startTime)

				// 记录结果
				if err != nil {
					span.SetError(err)
					logger.ErrorWithTrace(ctx, "RPC Client Call Error",
						zap.String("service", targetService),
						zap.String("method", methodName),
						zap.Duration("duration", duration),
						zap.Error(err),
					)
				} else {
					span.SetStatus(SpanStatusOK)
					logger.InfoWithTrace(ctx, "RPC Client Call End",
						zap.String("service", targetService),
						zap.String("method", methodName),
						zap.Duration("duration", duration),
					)
				}

				// 结束 Span
				span.End()
			}()

			return next(ctx, req, resp)
		}
	}
}

// ============ Context 传播辅助函数 ============

// InjectTraceContext 将追踪信息注入到 context（用于跨服务传播）
// 返回包含追踪信息的新 context
func InjectTraceContext(ctx context.Context) context.Context {
	// 确保有 TraceID
	if GetTraceID(ctx) == "" {
		ctx = WithTraceID(ctx, NewTraceID())
	}
	// 确保有 SpanID
	if GetSpanID(ctx) == "" {
		ctx = WithSpanID(ctx, NewSpanID())
	}
	return ctx
}

// ExtractTraceContext 从请求中提取追踪信息
// 返回 TraceID, SpanID, ParentSpanID
func ExtractTraceContext(ctx context.Context) (traceID, spanID, parentSpanID string) {
	traceID = GetTraceID(ctx)
	spanID = GetSpanID(ctx)
	parentSpanID = GetParentSpanID(ctx)
	return
}

// PropagateTraceHeaders 获取需要传播的 HTTP Header
// 用于 HTTP Client 调用时注入 Header
func PropagateTraceHeaders(ctx context.Context) map[string]string {
	headers := make(map[string]string)
	if traceID := GetTraceID(ctx); traceID != "" {
		headers[HTTPHeaderTraceID] = traceID
	}
	if spanID := GetSpanID(ctx); spanID != "" {
		headers[HTTPHeaderSpanID] = spanID
	}
	if parentSpanID := GetParentSpanID(ctx); parentSpanID != "" {
		headers[HTTPHeaderParentSpanID] = parentSpanID
	}
	return headers
}
