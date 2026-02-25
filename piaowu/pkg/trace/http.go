// Package trace 提供 HTTP 中间件追踪
// 用于 Gateway 层的链路追踪集成
package trace

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// ============ HTTP Server 中间件 ============

// HTTPServerMiddleware HTTP 服务端追踪中间件
// 功能：
// 1. 从请求 Header 中提取 TraceID/SpanID
// 2. 创建 Server Span
// 3. 在响应 Header 中返回 TraceID
// 4. 记录请求信息和耗时
func HTTPServerMiddleware(serviceName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// 1. 从 Header 提取追踪信息
		traceID := r.Header.Get(HTTPHeaderTraceID)
		if traceID == "" {
			traceID = NewTraceID()
		}
		parentSpanID := r.Header.Get(HTTPHeaderSpanID)

		// 2. 创建 Server Span
		ctx := r.Context()
		ctx = WithTraceID(ctx, traceID)
		if parentSpanID != "" {
			ctx = WithParentSpanID(ctx, parentSpanID)
		}

		spanName := r.Method + " " + r.URL.Path
		ctx, span := StartSpan(ctx, spanName,
			WithSpanKind(SpanKindServer),
			WithService(serviceName),
		)

		// 设置 HTTP 属性
		span.SetAttribute("http.method", r.Method)
		span.SetAttribute("http.path", r.URL.Path)
		span.SetAttribute("http.host", r.Host)
		span.SetAttribute("http.user_agent", r.UserAgent())
		span.SetAttribute("http.remote_addr", r.RemoteAddr)

		// 如果有查询参数，记录（脱敏）
		if r.URL.RawQuery != "" {
			span.SetAttribute("http.query", maskQueryString(r.URL.RawQuery))
		}

		// 3. 在响应 Header 中返回 TraceID
		w.Header().Set(HTTPHeaderTraceID, traceID)
		w.Header().Set(HTTPHeaderSpanID, span.SpanID)

		// 4. 创建响应记录器
		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// 更新请求的 context
		r = r.WithContext(ctx)

		// 5. 执行下一个处理器
		next.ServeHTTP(rec, r)

		// 6. 记录响应信息
		duration := time.Since(startTime)
		span.SetAttribute("http.status_code", strconv.Itoa(rec.statusCode))
		span.SetAttribute("http.duration_ms", strconv.FormatInt(duration.Milliseconds(), 10))
		span.SetAttribute("http.response_size", strconv.Itoa(rec.size))

		// 根据状态码设置 span 状态
		if rec.statusCode >= 400 {
			span.SetStatus(SpanStatusError)
			span.SetAttribute("error", "true")
		}

		// 结束 Span
		span.End()
	})
}

// responseRecorder 响应记录器
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	size       int
}

// WriteHeader 记录状态码
func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Write 记录响应大小
func (r *responseRecorder) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.size += size
	return size, err
}

// maskQueryString 脱敏查询字符串
func maskQueryString(query string) string {
	if len(query) > 50 {
		return query[:50] + "..."
	}
	return query
}

// ============ HTTP Client 追踪 ============

// TracedHTTPClient 带追踪的 HTTP 客户端
type TracedHTTPClient struct {
	client      *http.Client
	serviceName string
}

// NewTracedHTTPClient 创建带追踪的 HTTP 客户端
func NewTracedHTTPClient(client *http.Client, serviceName string) *TracedHTTPClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &TracedHTTPClient{
		client:      client,
		serviceName: serviceName,
	}
}

// Do 执行 HTTP 请求（带追踪）
func (c *TracedHTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// 创建 Client Span
	spanName := req.Method + " " + req.URL.Path
	ctx, span := StartSpan(ctx, spanName,
		WithSpanKind(SpanKindClient),
		WithService(c.serviceName),
	)

	// 设置属性
	span.SetAttribute("http.method", req.Method)
	span.SetAttribute("http.url", maskURL(req.URL.String()))
	span.SetAttribute("http.host", req.URL.Host)

	// 注入追踪信息到请求 Header
	req.Header.Set(HTTPHeaderTraceID, GetTraceID(ctx))
	req.Header.Set(HTTPHeaderSpanID, span.SpanID)
	if parentSpanID := GetParentSpanID(ctx); parentSpanID != "" {
		req.Header.Set(HTTPHeaderParentSpanID, parentSpanID)
	}

	// 执行请求
	startTime := time.Now()
	resp, err := c.client.Do(req.WithContext(ctx))
	duration := time.Since(startTime)

	span.SetAttribute("http.duration_ms", strconv.FormatInt(duration.Milliseconds(), 10))

	if err != nil {
		span.SetError(err)
		span.End()
		return nil, err
	}

	// 记录响应
	span.SetAttribute("http.status_code", strconv.Itoa(resp.StatusCode))
	if resp.StatusCode >= 400 {
		span.SetStatus(SpanStatusError)
	}

	span.End()
	return resp, nil
}

// ============ Context 辅助函数 ============

// ContextWithHTTPRequest 从 HTTP 请求创建带追踪信息的 context
func ContextWithHTTPRequest(r *http.Request) context.Context {
	ctx := r.Context()

	// 从 Header 提取追踪信息
	if traceID := r.Header.Get(HTTPHeaderTraceID); traceID != "" {
		ctx = WithTraceID(ctx, traceID)
	} else {
		ctx = WithTraceID(ctx, NewTraceID())
	}

	if spanID := r.Header.Get(HTTPHeaderSpanID); spanID != "" {
		ctx = WithParentSpanID(ctx, spanID) // 上游的 SpanID 成为当前的 ParentSpanID
	}

	return ctx
}

// SetTraceResponseHeaders 设置追踪相关的响应 Header
func SetTraceResponseHeaders(w http.ResponseWriter, ctx context.Context) {
	if traceID := GetTraceID(ctx); traceID != "" {
		w.Header().Set(HTTPHeaderTraceID, traceID)
	}
	if spanID := GetSpanID(ctx); spanID != "" {
		w.Header().Set(HTTPHeaderSpanID, spanID)
	}
}
