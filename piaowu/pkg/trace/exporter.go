// Package trace 提供 Jaeger 导出器功能
// 支持将 Span 数据导出到 Jaeger 进行可视化分析
// 功能：
// 1. 配置 Jaeger 连接
// 2. 异步批量导出 Span
// 3. 采样策略配置
package trace

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"example_shop/pkg/logger"

	"go.uber.org/zap"
)

// ============ 全局 Exporter ============

var (
	globalExporter Exporter
	exporterMu     sync.RWMutex
)

// Exporter Span 导出器接口
type Exporter interface {
	Export(span *Span)
	Shutdown()
}

// SetExporter 设置全局导出器
func SetExporter(exp Exporter) {
	exporterMu.Lock()
	defer exporterMu.Unlock()
	globalExporter = exp
}

// GetExporter 获取全局导出器
func GetExporter() Exporter {
	exporterMu.RLock()
	defer exporterMu.RUnlock()
	return globalExporter
}

// ============ Jaeger Exporter 配置 ============

// JaegerConfig Jaeger 导出器配置
type JaegerConfig struct {
	// Endpoint Jaeger Collector 地址
	// 例: "http://localhost:14268/api/traces"
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`

	// ServiceName 服务名称（显示在 Jaeger UI 中）
	ServiceName string `json:"service_name" yaml:"service_name" mapstructure:"service_name"`

	// Enabled 是否启用
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// SampleRate 采样率 (0.0 ~ 1.0)
	// 1.0 表示全量采样，0.1 表示 10% 采样
	SampleRate float64 `json:"sample_rate" yaml:"sample_rate" mapstructure:"sample_rate"`

	// BatchSize 批量发送大小
	BatchSize int `json:"batch_size" yaml:"batch_size" mapstructure:"batch_size"`

	// FlushInterval 刷新间隔
	FlushInterval time.Duration `json:"flush_interval" yaml:"flush_interval" mapstructure:"flush_interval"`
}

// DefaultJaegerConfig 默认 Jaeger 配置
func DefaultJaegerConfig() *JaegerConfig {
	return &JaegerConfig{
		Endpoint:      "http://jaeger:4318/v1/traces", // OTLP HTTP 端点
		ServiceName:   "unknown",
		Enabled:       false,
		SampleRate:    0.1, // 默认 10% 采样
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
	}
}

// ============ Jaeger Exporter 实现 ============

// JaegerExporter Jaeger 导出器
type JaegerExporter struct {
	config     *JaegerConfig
	spans      []*JaegerSpan
	mu         sync.Mutex
	httpClient *http.Client
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// JaegerSpan Jaeger 格式的 Span（Thrift/JSON 兼容）
type JaegerSpan struct {
	TraceID       string            `json:"traceID"`
	SpanID        string            `json:"spanID"`
	ParentSpanID  string            `json:"parentSpanID,omitempty"`
	OperationName string            `json:"operationName"`
	StartTime     int64             `json:"startTime"` // 微秒
	Duration      int64             `json:"duration"`  // 微秒
	Tags          []JaegerTag       `json:"tags,omitempty"`
	Logs          []JaegerLog       `json:"logs,omitempty"`
	ProcessID     string            `json:"processID"`
	Warnings      []string          `json:"warnings,omitempty"`
	References    []JaegerReference `json:"references,omitempty"`
}

// JaegerTag Jaeger 标签
type JaegerTag struct {
	Key   string      `json:"key"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// JaegerLog Jaeger 日志
type JaegerLog struct {
	Timestamp int64       `json:"timestamp"`
	Fields    []JaegerTag `json:"fields"`
}

// JaegerReference Span 引用关系
type JaegerReference struct {
	RefType string `json:"refType"` // CHILD_OF 或 FOLLOWS_FROM
	TraceID string `json:"traceID"`
	SpanID  string `json:"spanID"`
}

// JaegerBatch Jaeger 批量数据
type JaegerBatch struct {
	Process *JaegerProcess `json:"process"`
	Spans   []*JaegerSpan  `json:"spans"`
}

// JaegerProcess 进程信息
type JaegerProcess struct {
	ServiceName string      `json:"serviceName"`
	Tags        []JaegerTag `json:"tags,omitempty"`
}

// NewJaegerExporter 创建 Jaeger 导出器
func NewJaegerExporter(config *JaegerConfig) *JaegerExporter {
	if config == nil {
		config = DefaultJaegerConfig()
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = 5 * time.Second
	}

	exp := &JaegerExporter{
		config: config,
		spans:  make([]*JaegerSpan, 0, config.BatchSize),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		stopCh: make(chan struct{}),
	}

	// 启动后台刷新协程
	exp.wg.Add(1)
	go exp.flushLoop()

	logger.Info("Jaeger exporter initialized",
		zap.String("endpoint", config.Endpoint),
		zap.String("service", config.ServiceName),
		zap.Float64("sample_rate", config.SampleRate),
	)

	return exp
}

// Export 导出 Span
func (e *JaegerExporter) Export(span *Span) {
	if !e.config.Enabled {
		return
	}

	// 采样判断（简单随机采样）
	if e.config.SampleRate < 1.0 {
		// 使用 TraceID 的 hash 值进行一致性采样
		// 确保同一条链路的所有 Span 要么全部采样，要么全部不采样
		hash := hashTraceID(span.TraceID)
		if float64(hash%100)/100.0 > e.config.SampleRate {
			return
		}
	}

	// 转换为 Jaeger 格式
	jaegerSpan := e.convertSpan(span)

	e.mu.Lock()
	e.spans = append(e.spans, jaegerSpan)
	shouldFlush := len(e.spans) >= e.config.BatchSize
	e.mu.Unlock()

	if shouldFlush {
		go e.flush()
	}
}

// convertSpan 将内部 Span 转换为 Jaeger 格式
func (e *JaegerExporter) convertSpan(span *Span) *JaegerSpan {
	tags := make([]JaegerTag, 0, len(span.Attributes)+3)

	// 添加标准标签
	tags = append(tags, JaegerTag{Key: "span.kind", Type: "string", Value: span.Kind.String()})
	if span.Status == SpanStatusError {
		tags = append(tags, JaegerTag{Key: "error", Type: "bool", Value: true})
	}

	// 添加自定义属性
	for k, v := range span.Attributes {
		tags = append(tags, JaegerTag{Key: k, Type: "string", Value: v})
	}

	// 转换事件为日志
	logs := make([]JaegerLog, 0, len(span.Events))
	for _, event := range span.Events {
		fields := make([]JaegerTag, 0, len(event.Attributes)+1)
		fields = append(fields, JaegerTag{Key: "event", Type: "string", Value: event.Name})
		for k, v := range event.Attributes {
			fields = append(fields, JaegerTag{Key: k, Type: "string", Value: v})
		}
		logs = append(logs, JaegerLog{
			Timestamp: event.Timestamp.UnixMicro(),
			Fields:    fields,
		})
	}

	// 构建引用关系
	var refs []JaegerReference
	if span.ParentSpanID != "" {
		refs = []JaegerReference{{
			RefType: "CHILD_OF",
			TraceID: span.TraceID,
			SpanID:  span.ParentSpanID,
		}}
	}

	return &JaegerSpan{
		TraceID:       span.TraceID,
		SpanID:        span.SpanID,
		ParentSpanID:  span.ParentSpanID,
		OperationName: span.Name,
		StartTime:     span.StartTime.UnixMicro(),
		Duration:      span.Duration.Microseconds(),
		Tags:          tags,
		Logs:          logs,
		ProcessID:     "p1",
		References:    refs,
	}
}

// flushLoop 后台定时刷新
func (e *JaegerExporter) flushLoop() {
	defer e.wg.Done()
	ticker := time.NewTicker(e.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.flush()
		case <-e.stopCh:
			e.flush() // 最后一次刷新
			return
		}
	}
}

// flush 刷新 Span 到 Jaeger
func (e *JaegerExporter) flush() {
	e.mu.Lock()
	if len(e.spans) == 0 {
		e.mu.Unlock()
		return
	}
	spans := e.spans
	e.spans = make([]*JaegerSpan, 0, e.config.BatchSize)
	e.mu.Unlock()

	// 构建批量数据
	batch := &JaegerBatch{
		Process: &JaegerProcess{
			ServiceName: e.config.ServiceName,
			Tags: []JaegerTag{
				{Key: "hostname", Type: "string", Value: getHostname()},
				{Key: "ip", Type: "string", Value: getLocalIP()},
			},
		},
		Spans: spans,
	}

	// 发送到 Jaeger
	data, err := json.Marshal(batch)
	if err != nil {
		logger.Error("Failed to marshal spans", zap.Error(err))
		return
	}

	resp, err := e.httpClient.Post(e.config.Endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		logger.Warn("Failed to send spans to Jaeger", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		logger.Warn("Jaeger returned non-OK status", zap.Int("status", resp.StatusCode))
	}
}

// Shutdown 关闭导出器
func (e *JaegerExporter) Shutdown() {
	close(e.stopCh)
	e.wg.Wait()
	logger.Info("Jaeger exporter shutdown")
}

// ============ 辅助函数 ============

// hashTraceID 计算 TraceID 的 hash 值（用于一致性采样）
func hashTraceID(traceID string) int {
	hash := 0
	for _, c := range traceID {
		hash = 31*hash + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// getHostname 获取主机名
func getHostname() string {
	// 简化实现，实际应使用 os.Hostname()
	return "localhost"
}

// getLocalIP 获取本地 IP
func getLocalIP() string {
	// 简化实现
	return "127.0.0.1"
}

// ============ 初始化函数 ============

// InitJaeger 初始化 Jaeger 导出器
// 如果配置中 Enabled 为 false，则不会创建导出器
func InitJaeger(config *JaegerConfig) {
	if config == nil || !config.Enabled {
		logger.Info("Jaeger exporter is disabled")
		return
	}
	exporter := NewJaegerExporter(config)
	SetExporter(exporter)
}

// ShutdownJaeger 关闭 Jaeger 导出器
func ShutdownJaeger() {
	if exp := GetExporter(); exp != nil {
		exp.Shutdown()
	}
}
