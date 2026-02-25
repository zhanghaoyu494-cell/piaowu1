// Package trace 提供 Redis 追踪功能
// 包装 Redis 操作，自动记录 child span
package trace

import (
	"context"
	"fmt"
	"time"
)

// ============ Redis 追踪包装器 ============

// RedisTracer Redis 追踪器
// 包装 Redis 操作，自动创建 span
type RedisTracer struct {
	serviceName string
}

// NewRedisTracer 创建 Redis 追踪器
func NewRedisTracer(serviceName string) *RedisTracer {
	return &RedisTracer{
		serviceName: serviceName,
	}
}

// TraceCommand 追踪 Redis 命令
// 参数:
//   - ctx: 上下文
//   - cmd: 命令名（如 GET, SET, HGET 等）
//   - key: Redis Key
//   - fn: 实际执行函数
//
// 返回: 执行结果和错误
func (t *RedisTracer) TraceCommand(ctx context.Context, cmd, key string, fn func() (interface{}, error)) (interface{}, error) {
	// 创建 span
	spanName := fmt.Sprintf("Redis %s", cmd)
	ctx, span := StartSpan(ctx, spanName,
		WithSpanKind(SpanKindClient),
		WithService("redis"),
	)

	// 设置属性
	span.SetAttribute("redis.command", cmd)
	span.SetAttribute("redis.key", maskRedisKey(key))
	span.SetAttribute("db.system", "redis")

	startTime := time.Now()

	// 执行命令
	result, err := fn()

	duration := time.Since(startTime)
	span.SetAttribute("redis.duration_ms", fmt.Sprintf("%d", duration.Milliseconds()))

	if err != nil {
		span.SetError(err)
	}

	span.End()
	return result, err
}

// TracePipeline 追踪 Redis Pipeline
func (t *RedisTracer) TracePipeline(ctx context.Context, cmdCount int, fn func() error) error {
	spanName := "Redis Pipeline"
	ctx, span := StartSpan(ctx, spanName,
		WithSpanKind(SpanKindClient),
		WithService("redis"),
	)

	span.SetAttribute("redis.command", "PIPELINE")
	span.SetAttribute("redis.cmd_count", fmt.Sprintf("%d", cmdCount))
	span.SetAttribute("db.system", "redis")

	startTime := time.Now()
	err := fn()
	duration := time.Since(startTime)

	span.SetAttribute("redis.duration_ms", fmt.Sprintf("%d", duration.Milliseconds()))

	if err != nil {
		span.SetError(err)
	}

	span.End()
	return err
}

// maskRedisKey 脱敏 Redis Key
func maskRedisKey(key string) string {
	// 对包含敏感信息的 key 进行脱敏
	if len(key) <= 10 {
		return key
	}
	// 保留前缀和后缀
	return key[:6] + "****" + key[len(key)-4:]
}

// ============ 便捷追踪函数 ============

// TraceRedisGet 追踪 Redis GET 操作
func TraceRedisGet(ctx context.Context, key string, fn func() (string, error)) (string, error) {
	ctx, span := StartRedisSpan(ctx, "GET", key)
	defer span.End()

	result, err := fn()
	if err != nil {
		span.SetError(err)
	}
	return result, err
}

// TraceRedisSet 追踪 Redis SET 操作
func TraceRedisSet(ctx context.Context, key string, fn func() error) error {
	ctx, span := StartRedisSpan(ctx, "SET", key)
	defer span.End()

	err := fn()
	if err != nil {
		span.SetError(err)
	}
	return err
}

// TraceRedisDel 追踪 Redis DEL 操作
func TraceRedisDel(ctx context.Context, keys []string, fn func() error) error {
	keyStr := "multiple"
	if len(keys) == 1 {
		keyStr = keys[0]
	}
	ctx, span := StartRedisSpan(ctx, "DEL", keyStr)
	span.SetAttribute("redis.key_count", fmt.Sprintf("%d", len(keys)))
	defer span.End()

	err := fn()
	if err != nil {
		span.SetError(err)
	}
	return err
}

// TraceRedisHGet 追踪 Redis HGET 操作
func TraceRedisHGet(ctx context.Context, key, field string, fn func() (string, error)) (string, error) {
	ctx, span := StartRedisSpan(ctx, "HGET", key)
	span.SetAttribute("redis.field", field)
	defer span.End()

	result, err := fn()
	if err != nil {
		span.SetError(err)
	}
	return result, err
}

// TraceRedisHSet 追踪 Redis HSET 操作
func TraceRedisHSet(ctx context.Context, key string, fn func() error) error {
	ctx, span := StartRedisSpan(ctx, "HSET", key)
	defer span.End()

	err := fn()
	if err != nil {
		span.SetError(err)
	}
	return err
}

// TraceRedisExpire 追踪 Redis EXPIRE 操作
func TraceRedisExpire(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	ctx, span := StartRedisSpan(ctx, "EXPIRE", key)
	span.SetAttribute("redis.ttl_seconds", fmt.Sprintf("%d", int(ttl.Seconds())))
	defer span.End()

	err := fn()
	if err != nil {
		span.SetError(err)
	}
	return err
}

// ============ 外部 HTTP 调用追踪 ============

// HTTPClientTracer HTTP 客户端追踪器
type HTTPClientTracer struct {
	serviceName string
}

// NewHTTPClientTracer 创建 HTTP 客户端追踪器
func NewHTTPClientTracer(serviceName string) *HTTPClientTracer {
	return &HTTPClientTracer{
		serviceName: serviceName,
	}
}

// TraceRequest 追踪 HTTP 请求
func (t *HTTPClientTracer) TraceRequest(ctx context.Context, method, url string, fn func(ctx context.Context) (statusCode int, err error)) (int, error) {
	ctx, span := StartHTTPSpan(ctx, method, url)
	span.SetAttribute("http.url", maskURL(url))
	defer span.End()

	statusCode, err := fn(ctx)

	span.SetAttribute("http.status_code", fmt.Sprintf("%d", statusCode))
	if err != nil {
		span.SetError(err)
	} else if statusCode >= 400 {
		span.SetStatus(SpanStatusError)
		span.SetAttribute("error", "true")
	}

	return statusCode, err
}

// maskURL 脱敏 URL（去除敏感参数）
func maskURL(url string) string {
	// 简单实现：截断过长的 URL
	if len(url) > 100 {
		return url[:100] + "..."
	}
	return url
}

// ============ MQ 消息追踪 ============

// MQTracer 消息队列追踪器
type MQTracer struct {
	serviceName string
}

// NewMQTracer 创建 MQ 追踪器
func NewMQTracer(serviceName string) *MQTracer {
	return &MQTracer{
		serviceName: serviceName,
	}
}

// TracePublish 追踪消息发布
func (t *MQTracer) TracePublish(ctx context.Context, topic string, fn func() error) error {
	ctx, span := StartSpan(ctx, "MQ Publish",
		WithSpanKind(SpanKindProducer),
		WithService("mq"),
	)
	span.SetAttribute("mq.topic", topic)
	span.SetAttribute("mq.operation", "publish")
	defer span.End()

	err := fn()
	if err != nil {
		span.SetError(err)
	}
	return err
}

// TraceConsume 追踪消息消费
func (t *MQTracer) TraceConsume(ctx context.Context, topic string, fn func(ctx context.Context) error) error {
	ctx, span := StartSpan(ctx, "MQ Consume",
		WithSpanKind(SpanKindConsumer),
		WithService("mq"),
	)
	span.SetAttribute("mq.topic", topic)
	span.SetAttribute("mq.operation", "consume")
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.SetError(err)
	}
	return err
}
