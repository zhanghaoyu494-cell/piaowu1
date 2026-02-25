// Package plugin 提供 GORM 插件实现
// 包含：
// - TraceLogger: 支持链路追踪的 GORM 日志记录器
// - TracePlugin: GORM 插件，确保 context 中的 TraceID 正确传递
//
// 功能特性：
// 1. 所有 SQL 日志自动带上 TraceID
// 2. 区分 SQL 错误和正常的"记录不存在"
// 3. 自动标记慢查询（>200ms）
package plugin

import (
	"context"
	"fmt"
	"time"

	"example_shop/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// ============ TraceLogger 实现 ============

// TraceLogger GORM 日志记录器，支持 TraceID
// 所有 SQL 日志都会带上当前请求的 TraceID，便于日志关联和问题排查
type TraceLogger struct {
	LogLevel gormlogger.LogLevel // 日志级别
}

// NewTraceLogger 创建新的 TraceLogger
func NewTraceLogger() *TraceLogger {
	return &TraceLogger{
		LogLevel: gormlogger.Info,
	}
}

// LogMode 设置日志级别
func (l *TraceLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info 记录 Info 级别日志
func (l *TraceLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		logger.InfoWithTrace(ctx, fmt.Sprintf(msg, data...))
	}
}

// Warn 记录 Warn 级别日志
func (l *TraceLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		logger.WarnWithTrace(ctx, fmt.Sprintf(msg, data...))
	}
}

// Error 记录 Error 级别日志
func (l *TraceLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		logger.ErrorWithTrace(ctx, fmt.Sprintf(msg, data...))
	}
}

// Trace 记录 SQL 执行日志（包含 TraceID）
func (l *TraceLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// 根据不同情况记录日志
	switch {
	case err != nil && l.LogLevel >= gormlogger.Error:
		// 区分真正的错误和正常的"record not found"
		if err == gorm.ErrRecordNotFound {
			// record not found 是正常的查询结果，不应记录为错误
			logger.DebugWithTrace(ctx, "No record found",
				zap.Duration("elapsed", elapsed),
				zap.String("sql", sql),
				zap.Int64("rows", rows),
			)
		} else {
			// 真正的SQL错误（如连接失败、语法错误等）
			logger.ErrorWithTrace(ctx, "SQL Error",
				zap.Error(err),
				zap.Duration("elapsed", elapsed),
				zap.String("sql", sql),
				zap.Int64("rows", rows),
			)
		}
	case elapsed > 200*time.Millisecond && l.LogLevel >= gormlogger.Warn:
		// 慢查询（超过 200ms）
		logger.WarnWithTrace(ctx, "Slow SQL",
			zap.Duration("elapsed", elapsed),
			zap.String("sql", sql),
			zap.Int64("rows", rows),
		)
	case l.LogLevel >= gormlogger.Info:
		// 正常查询
		logger.DebugWithTrace(ctx, "SQL Query",
			zap.Duration("elapsed", elapsed),
			zap.String("sql", sql),
			zap.Int64("rows", rows),
		)
	}
}

// ============ TracePlugin 实现 ============

// TracePlugin GORM 插件，用于从 context 提取 TraceID
// 在所有数据库操作前注册回调，确保 context 正确传递
type TracePlugin struct{}

// Name 插件名称
func (p *TracePlugin) Name() string {
	return "trace_plugin"
}

// Initialize 初始化插件
func (p *TracePlugin) Initialize(db *gorm.DB) error {
	// 注册回调，在执行前从 context 提取 TraceID
	callback := db.Callback()

	// 在所有操作之前设置 TraceID
	_ = callback.Create().Before("gorm:before_create").Register("trace:before_create", p.before)
	_ = callback.Query().Before("gorm:query").Register("trace:before_query", p.before)
	_ = callback.Update().Before("gorm:before_update").Register("trace:before_update", p.before)
	_ = callback.Delete().Before("gorm:before_delete").Register("trace:before_delete", p.before)
	_ = callback.Row().Before("gorm:row").Register("trace:before_row", p.before)
	_ = callback.Raw().Before("gorm:raw").Register("trace:before_raw", p.before)

	return nil
}

// before 在执行前的回调
func (p *TracePlugin) before(db *gorm.DB) {
	// TraceID 已经在 context 中，GORM 会自动传递
	// 这里只是确保 context 正确传递
	if db.Statement.Context == nil {
		db.Statement.Context = context.Background()
	}
}
