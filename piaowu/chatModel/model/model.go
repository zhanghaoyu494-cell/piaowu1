package model

import (
	"time"
)

// ConversationSummary 会话摘要表：记录 AI 自动分析出的会话精简内容
type ConversationSummary struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID string    `gorm:"index;type:varchar(64);comment:关联会话ID" json:"conversation_id"`
	Summary        string    `gorm:"type:text;comment:会话内容摘要" json:"summary"`
	MainTopic      string    `gorm:"type:varchar(100);comment:会话主旨话题" json:"main_topic"`
	Sentiment      string    `gorm:"type:varchar(50);comment:情感倾向" json:"sentiment"`
	TraceID        string    `gorm:"type:varchar(64);comment:全链路追踪ID" json:"trace_id"` // 用于关联审计日志
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ConversationSummary) TableName() string {
	return "t_conversation_summary"
}

// AISuggestionLog AI 建议日志表：详细记录生成的每一步操作建议及其置信度
type AISuggestionLog struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID string    `gorm:"index;type:varchar(64);comment:关联会话ID" json:"conversation_id"`
	ActionType     string    `gorm:"type:varchar(50);comment:建议类型(退款/补发/解释)" json:"action_type"`
	Suggestion     string    `gorm:"type:text;comment:建议具体内容" json:"suggestion"`
	Confidence     float64   `gorm:"comment:置信度" json:"confidence"`
	IsAdopted      bool      `gorm:"default:false;comment:是否被客服采纳" json:"is_adopted"` // 为未来优化模型提供数据
	TraceID        string    `gorm:"type:varchar(64)" json:"trace_id"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (AISuggestionLog) TableName() string {
	return "t_ai_suggestion_log"
}

// RiskAlertLog 风险告警日志表：记录实时分类器触发的风险预警
type RiskAlertLog struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID string    `gorm:"index;type:varchar(64);comment:关联会话ID" json:"conversation_id"`
	Level          string    `gorm:"type:varchar(20);comment:风险等级(Low/Medium/High)" json:"level"`
	RiskType       string    `gorm:"type:varchar(50);comment:风险类型(投诉倾向/辱骂/欺诈)" json:"risk_type"`
	Content        string    `gorm:"type:text;comment:触发告警的关键文本内容" json:"content"`
	Status         int       `gorm:"default:0;comment:处理状态(0-未处理, 1-已处理)" json:"status"`
	TraceID        string    `gorm:"type:varchar(64)" json:"trace_id"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (RiskAlertLog) TableName() string {
	return "t_risk_alert_log"
}

// AIToolAuditLog AI 工具审计日志：记录 RAG、LLM 等每一个原子工具调用的详细入参、出参和耗时
type AIToolAuditLog struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ActorID   int64     `gorm:"index;comment:操作者ID(客服或系统)" json:"actor_id"`
	ToolName  string    `gorm:"type:varchar(100);comment:使用的工具或组件名" json:"tool_name"`
	Input     string    `gorm:"type:text;comment:输入参数" json:"input"`
	Output    string    `gorm:"type:text;comment:组件输出" json:"output"`
	TraceID   string    `gorm:"type:varchar(64)" json:"trace_id"`
	ModelName string    `gorm:"type:varchar(100);comment:底层模型名称" json:"model_name"`
	LatencyMs int64     `gorm:"comment:耗时(ms)" json:"latency_ms"` // 性能监控核心字段
	PromptVer string    `gorm:"type:varchar(50);comment:提示词版本" json:"prompt_ver"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (AIToolAuditLog) TableName() string {
	return "t_ai_tool_audit_log"
}

// AIJob AI 异步任务表：维护长耗时 AI 处理的状态机
type AIJob struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	JobID          string    `gorm:"uniqueIndex;type:varchar(64);comment:任务唯一标识" json:"job_id"`
	ConversationID string    `gorm:"index;type:varchar(64);comment:关联会话ID" json:"conversation_id"`
	Status         string    `gorm:"type:varchar(20);default:'pending';comment:任务状态(pending/processing/completed/failed)" json:"status"`
	Result         string    `gorm:"type:text;comment:摘要/建议结果JSON" json:"result"`
	Error          string    `gorm:"type:text;comment:错误信息" json:"error"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (AIJob) TableName() string {
	return "t_ai_job"
}

// ConvMessage 这里的定义是为了让 Worker 能够直接向业务消息表回填“建议回复”
type ConvMessage struct {
	MsgID        int64     `gorm:"primaryKey;autoIncrement;column:msg_id" json:"msg_id"`
	ConvID       string    `gorm:"column:conv_id;type:varchar(64)" json:"conv_id"`
	SenderType   int8      `gorm:"column:sender_type" json:"sender_type"` // 0:用户, 1:客服, 2:系统
	SenderID     string    `gorm:"column:sender_id;type:varchar(32)" json:"sender_id"`
	MsgContent   string    `gorm:"column:msg_content;type:text" json:"msg_content"`
	IsQuickReply int8      `gorm:"column:is_quick_reply" json:"is_quick_reply"`
	QuickReplyID int64     `gorm:"column:quick_reply_id" json:"quick_reply_id"`
	SendTime     time.Time `gorm:"column:send_time" json:"send_time"`
	CreateTime   time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime   time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
}

func (ConvMessage) TableName() string {
	return "t_conv_message"
}
