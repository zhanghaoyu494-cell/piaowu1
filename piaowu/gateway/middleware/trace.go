// Package middleware 提供 Gateway 层 HTTP 中间件能力。
//
// 本文件实现链路追踪中间件：为每个 HTTP 请求创建/传播 TraceID 与 Server Span，并把关键指标写入 span 与日志。
//
// Trace 传播约定：
// - 入站：从请求 Header 读取 X-Trace-ID / X-Span-ID（若缺失则生成新的 TraceID）
// - 出站：把当前 TraceID 与本次 Server SpanID 写回响应 Header，便于调用方关联日志与追踪
//
// 说明：
// - 中间件会把 TraceID 注入到 pkg/trace 的 context 中，同时也注入到 pkg/logger 的 context 中（兼容历史日志链路）
// - responseRecorder 代理 ResponseWriter 以捕获 status_code 与响应大小，并实现 Hijack 以保证 WebSocket 升级不被破坏
package middleware

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"example_shop/pkg/logger"
	"example_shop/pkg/trace"

	"go.uber.org/zap"
)

const (
	// TraceIDHeader 用于在 HTTP Header 中传递 TraceID。
	TraceIDHeader = "X-Trace-ID"
	// SpanIDHeader 用于在 HTTP Header 中传递父 SpanID（上游 span）。
	SpanIDHeader = "X-Span-ID"
)

// TraceMiddleware 创建链路追踪中间件，默认 serviceName 使用 "gateway"。

// 功能概览：
// 1) 提取/生成 TraceID，并可选读取父 SpanID 建立父子关系
// 2) 创建 Server Span，记录 http.method/http.path/http.status_code/http.duration_ms 等关键属性
// 3) 把 TraceID 注入 context，确保下游 handler、RPC 调用与日志能关联同一条链路
// 4) 在响应 Header 回传 TraceID 与本次 Server SpanID，便于排查与定位
func TraceMiddleware(next http.Handler) http.Handler {
	return TraceMiddlewareWithService("gateway", next)
}

// TraceMiddlewareWithService 创建带 serviceName 的链路追踪中间件。
//
// serviceName 会写入 span 的 service 维度属性，便于在多服务场景下区分来源。
func TraceMiddlewareWithService(serviceName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		traceID := r.Header.Get(TraceIDHeader)
		if traceID == "" {
			traceID = trace.NewTraceID()
		}
		parentSpanID := r.Header.Get(SpanIDHeader)

		ctx := r.Context()
		ctx = trace.WithTraceID(ctx, traceID)
		if parentSpanID != "" {
			ctx = trace.WithParentSpanID(ctx, parentSpanID)
		}

		spanName := r.Method + " " + r.URL.Path
		ctx, span := trace.StartSpan(
			ctx,
			spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithService(serviceName),
		)

		span.SetAttribute("http.method", r.Method)
		span.SetAttribute("http.path", r.URL.Path)
		span.SetAttribute("http.host", r.Host)
		span.SetAttribute("http.remote_addr", r.RemoteAddr)
		span.SetAttribute("http.user_agent", r.UserAgent())

		ctx = logger.WithTraceID(ctx, traceID)
		r = r.WithContext(ctx)

		w.Header().Set(TraceIDHeader, traceID)
		w.Header().Set(SpanIDHeader, span.SpanID)

		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		next.ServeHTTP(rec, r)

		duration := time.Since(startTime)
		span.SetAttribute("http.status_code", strconv.Itoa(rec.statusCode))
		span.SetAttribute("http.duration_ms", strconv.FormatInt(duration.Milliseconds(), 10))
		span.SetAttribute("http.response_size", strconv.Itoa(rec.size))

		if rec.statusCode >= 400 {
			span.SetStatus(trace.SpanStatusError)
		}
		span.End()

		logger.InfoWithTrace(
			ctx,
			"HTTP Request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.Int("status", rec.statusCode),
			zap.Duration("duration", duration),
		)
	})
}

// responseRecorder 代理 http.ResponseWriter，用于捕获响应状态码与响应体大小。
//
// 注意：
// - WriteHeader 用于记录最终 status_code
// - Write 用于累计响应体字节数
// - Hijack 透传到底层 ResponseWriter，确保 WebSocket 等需要接管底层连接的场景不受影响
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	size, err := rec.ResponseWriter.Write(b)
	rec.size += size
	return size, err
}

func (rec *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rec.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// GetTraceIDFromRequest 从 HTTP 请求上下文读取 TraceID（若不存在返回空字符串）。
func GetTraceIDFromRequest(r *http.Request) string {
	return trace.GetTraceID(r.Context())
}

// GetTraceIDFromContext 从 context 读取 TraceID（用于非 HTTP 场景复用）。
func GetTraceIDFromContext(ctx context.Context) string {
	return trace.GetTraceID(ctx)
}

// GetSpanIDFromContext 从 context 读取 SpanID（用于定位当前 span）。
func GetSpanIDFromContext(ctx context.Context) string {
	return trace.GetSpanID(ctx)
}
