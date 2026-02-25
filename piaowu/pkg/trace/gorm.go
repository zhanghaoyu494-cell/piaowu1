// Package trace 提供数据库追踪插件
// 集成 GORM，自动记录 SQL 操作的 child span
package trace

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ============ GORM Trace Plugin ============

// GormTracePlugin GORM 链路追踪插件
// 在每个数据库操作前后创建 span，记录 SQL 执行信息
type GormTracePlugin struct {
	serviceName string // 服务名称
}

// NewGormTracePlugin 创建 GORM 追踪插件
func NewGormTracePlugin(serviceName string) *GormTracePlugin {
	return &GormTracePlugin{
		serviceName: serviceName,
	}
}

// Name 插件名称
func (p *GormTracePlugin) Name() string {
	return "gorm_trace_plugin"
}

// Initialize 初始化插件，注册回调
func (p *GormTracePlugin) Initialize(db *gorm.DB) error {
	// Create 操作
	_ = db.Callback().Create().Before("gorm:create").Register("trace:before_create", p.beforeCreate)
	_ = db.Callback().Create().After("gorm:create").Register("trace:after_create", p.afterCreate)

	// Query 操作
	_ = db.Callback().Query().Before("gorm:query").Register("trace:before_query", p.beforeQuery)
	_ = db.Callback().Query().After("gorm:query").Register("trace:after_query", p.afterQuery)

	// Update 操作
	_ = db.Callback().Update().Before("gorm:update").Register("trace:before_update", p.beforeUpdate)
	_ = db.Callback().Update().After("gorm:update").Register("trace:after_update", p.afterUpdate)

	// Delete 操作
	_ = db.Callback().Delete().Before("gorm:delete").Register("trace:before_delete", p.beforeDelete)
	_ = db.Callback().Delete().After("gorm:delete").Register("trace:after_delete", p.afterDelete)

	// Raw 操作
	_ = db.Callback().Raw().Before("gorm:raw").Register("trace:before_raw", p.beforeRaw)
	_ = db.Callback().Raw().After("gorm:raw").Register("trace:after_raw", p.afterRaw)

	// Row 操作
	_ = db.Callback().Row().Before("gorm:row").Register("trace:before_row", p.beforeRow)
	_ = db.Callback().Row().After("gorm:row").Register("trace:after_row", p.afterRow)

	return nil
}

// ============ Context Key for Span Storage ============

type gormSpanKey struct{}

func setSpan(db *gorm.DB, span *Span) {
	db.Statement.Context = context.WithValue(db.Statement.Context, gormSpanKey{}, span)
}

func getSpan(db *gorm.DB) *Span {
	if v, ok := db.Statement.Context.Value(gormSpanKey{}).(*Span); ok {
		return v
	}
	return nil
}

// ============ Create Callbacks ============

func (p *GormTracePlugin) beforeCreate(db *gorm.DB) {
	p.startSpan(db, "INSERT", "create")
}

func (p *GormTracePlugin) afterCreate(db *gorm.DB) {
	p.endSpan(db)
}

// ============ Query Callbacks ============

func (p *GormTracePlugin) beforeQuery(db *gorm.DB) {
	p.startSpan(db, "SELECT", "query")
}

func (p *GormTracePlugin) afterQuery(db *gorm.DB) {
	p.endSpan(db)
}

// ============ Update Callbacks ============

func (p *GormTracePlugin) beforeUpdate(db *gorm.DB) {
	p.startSpan(db, "UPDATE", "update")
}

func (p *GormTracePlugin) afterUpdate(db *gorm.DB) {
	p.endSpan(db)
}

// ============ Delete Callbacks ============

func (p *GormTracePlugin) beforeDelete(db *gorm.DB) {
	p.startSpan(db, "DELETE", "delete")
}

func (p *GormTracePlugin) afterDelete(db *gorm.DB) {
	p.endSpan(db)
}

// ============ Raw Callbacks ============

func (p *GormTracePlugin) beforeRaw(db *gorm.DB) {
	p.startSpan(db, "RAW", "raw")
}

func (p *GormTracePlugin) afterRaw(db *gorm.DB) {
	p.endSpan(db)
}

// ============ Row Callbacks ============

func (p *GormTracePlugin) beforeRow(db *gorm.DB) {
	p.startSpan(db, "ROW", "row")
}

func (p *GormTracePlugin) afterRow(db *gorm.DB) {
	p.endSpan(db)
}

// ============ Span 管理 ============

func (p *GormTracePlugin) startSpan(db *gorm.DB, operation, callbackType string) {
	if db.Statement.Context == nil {
		db.Statement.Context = context.Background()
	}

	// 获取表名
	table := db.Statement.Table
	if table == "" && db.Statement.Schema != nil {
		table = db.Statement.Schema.Table
	}
	if table == "" {
		table = "unknown"
	}

	// 创建 span
	spanName := fmt.Sprintf("DB %s %s", operation, table)
	_, span := StartSpan(db.Statement.Context, spanName,
		WithSpanKind(SpanKindClient),
		WithService("mysql"),
	)

	// 设置属性
	span.SetAttribute("db.system", "mysql")
	span.SetAttribute("db.operation", operation)
	span.SetAttribute("db.table", table)
	span.SetAttribute("db.callback_type", callbackType)

	// 存储 span 到 context
	setSpan(db, span)
}

func (p *GormTracePlugin) endSpan(db *gorm.DB) {
	span := getSpan(db)
	if span == nil {
		return
	}

	// 获取 SQL（注意：某些情况下 SQL 可能为空）
	sql := db.Statement.SQL.String()
	if sql != "" {
		// 截断过长的 SQL
		if len(sql) > 500 {
			sql = sql[:500] + "..."
		}
		span.SetAttribute("db.statement", sql)
	}

	// 记录影响行数
	span.SetAttribute("db.rows_affected", fmt.Sprintf("%d", db.Statement.RowsAffected))

	// 检查错误
	if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
		span.SetError(db.Error)
	}

	span.End()
}

// ============ 带追踪的数据库操作辅助函数 ============

// DBOperation 表示一个数据库操作
type DBOperation func(db *gorm.DB) *gorm.DB

// WithDBTrace 包装数据库操作，自动添加追踪
// 用于手动控制 span 的场景
func WithDBTrace(ctx context.Context, db *gorm.DB, operation, table string, fn DBOperation) *gorm.DB {
	// 确保 context 传递到 GORM
	db = db.WithContext(ctx)

	// 创建 span
	spanName := fmt.Sprintf("DB %s %s", operation, table)
	ctx, span := StartSpan(ctx, spanName,
		WithSpanKind(SpanKindClient),
		WithService("mysql"),
	)
	span.SetAttribute("db.operation", operation)
	span.SetAttribute("db.table", table)

	// 更新 context
	db = db.WithContext(ctx)

	// 执行操作
	startTime := time.Now()
	result := fn(db)
	duration := time.Since(startTime)

	// 记录结果
	span.SetAttribute("db.duration_ms", fmt.Sprintf("%d", duration.Milliseconds()))
	span.SetAttribute("db.rows_affected", fmt.Sprintf("%d", result.RowsAffected))

	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		span.SetError(result.Error)
	}

	span.End()
	return result
}

// ============ 事务追踪 ============

// TxOperation 事务操作函数
type TxOperation func(tx *gorm.DB) error

// WithTxTrace 包装事务操作，自动添加追踪
func WithTxTrace(ctx context.Context, db *gorm.DB, fn TxOperation) error {
	// 创建事务 span
	ctx, span := StartSpan(ctx, "DB Transaction",
		WithSpanKind(SpanKindClient),
		WithService("mysql"),
	)
	span.SetAttribute("db.operation", "transaction")

	startTime := time.Now()

	// 执行事务
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 传递追踪 context
		return fn(tx.WithContext(ctx))
	})

	duration := time.Since(startTime)
	span.SetAttribute("db.duration_ms", fmt.Sprintf("%d", duration.Milliseconds()))

	if err != nil {
		span.SetError(err)
		span.AddEvent("transaction_rollback", nil)
	} else {
		span.AddEvent("transaction_commit", nil)
	}

	span.End()
	return err
}
