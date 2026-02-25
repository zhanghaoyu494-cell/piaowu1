// Package model 定义客服系统的数据库模型
// 包含所有业务实体的结构定义，与数据库表一一对应
// 主要包含：
// - 班次配置（ShiftConfig）
// - 客服信息（CustomerService）
// - 排班记录（Schedule）
// - 会话管理（Conversation, ConvMessage）
// - 请假调班（LeaveTransfer）
package model

import (
	"time"
)

// 客服系统角色枚举 (对应 CustomerService.Role)
const (
	CsRoleCustomerService = 0 // 客服
	CsRoleManager         = 1 // 部门经理
	CsRoleAdmin           = 2 // 管理员
)

// ShiftConfig 班次配置表
// 用于定义客服排班的班次模板，如早班、晚班、节假日班等
// 对应数据库表: t_shift_config
type ShiftConfig struct {
	ShiftID    int64     `gorm:"column:shift_id;primaryKey;autoIncrement" json:"shift_id"`      // 班次ID，主键
	ShiftName  string    `gorm:"column:shift_name;type:varchar(32);not null" json:"shift_name"` // 班次名称，如"早班"、"晚班"
	StartTime  string    `gorm:"column:start_time;type:time;not null" json:"start_time"`        // 开始时间，格式HH:MM:SS
	EndTime    string    `gorm:"column:end_time;type:time;not null" json:"end_time"`            // 结束时间，格式HH:MM:SS
	MinStaff   int       `gorm:"column:min_staff;not null" json:"min_staff"`                    // 最少在岗人数
	IsHoliday  int8      `gorm:"column:is_holiday;type:tinyint(1);not null" json:"is_holiday"`  // 是否节假日班: 0=否, 1=是
	CreateTime time.Time `gorm:"column:create_time;not null" json:"create_time"`                // 创建时间
	UpdateTime time.Time `gorm:"column:update_time;not null" json:"update_time"`                // 更新时间
	CreateBy   string    `gorm:"column:create_by;type:varchar(64);not null" json:"create_by"`   // 创建人
}

// TableName 指定数据库表名
func (ShiftConfig) TableName() string {
	return "t_shift_config"
}

// CustomerService 客服信息表
// 存储客服人员的基本信息，用于排班和会话分配
// 对应数据库表: t_customer_service
type CustomerService struct {
	CsID          string     `gorm:"column:cs_id;type:varchar(32);primaryKey" json:"cs_id"`                // 客服ID，如CS001
	CsName        string     `gorm:"column:cs_name;type:varchar(64);not null" json:"cs_name"`              // 客服姓名
	DeptID        string     `gorm:"column:dept_id;type:varchar(32);not null" json:"dept_id"`              // 部门ID
	TeamID        string     `gorm:"column:team_id;type:varchar(32)" json:"team_id"`                       // 班组ID
	SkillTags     string     `gorm:"column:skill_tags;type:varchar(128)" json:"skill_tags"`                // 技能标签，逗号分隔
	Status        int8       `gorm:"column:status;type:tinyint(1);not null" json:"status"`                 // 在职状态: 0=离职, 1=在职
	CurrentStatus int8       `gorm:"column:current_status;type:tinyint(1);not null" json:"current_status"` // 当前状态: 0=空闲, 1=工作中, 2=请假
	IsOnline      int8       `gorm:"column:is_online;type:tinyint(1);not null;default:0" json:"is_online"` // 在线状态: 0=离线, 1=在线
	LastHeartbeat *time.Time `gorm:"column:last_heartbeat" json:"last_heartbeat"`                          // 最后心跳时间
	Role          int8       `gorm:"column:role;type:tinyint(1);not null;default:0" json:"role"`           // 角色: 0=客服, 1=部门经理, 2=管理员
	PasswordHash  string     `gorm:"column:password_hash;type:varchar(128)" json:"-"`                      // 密码哈希（登录用，不返回给前端）
	CreateTime    time.Time  `gorm:"column:create_time;not null" json:"create_time"`                       // 创建时间
	UpdateTime    time.Time  `gorm:"column:update_time;not null" json:"update_time"`                       // 更新时间
}

// TableName 指定数据库表名
func (CustomerService) TableName() string {
	return "t_customer_service"
}

// Schedule 排班记录表
// 记录客服每天的排班信息
// 对应数据库表: t_schedule
type Schedule struct {
	ScheduleID   int64     `gorm:"column:schedule_id;primaryKey;autoIncrement" json:"schedule_id"`                                      // 排班记录ID
	CsID         string    `gorm:"column:cs_id;type:varchar(32);not null;index:idx_cs_date" json:"cs_id"`                               // 客服ID
	ShiftID      int64     `gorm:"column:shift_id;not null;index:idx_date_shift" json:"shift_id"`                                       // 班次ID
	ScheduleDate string    `gorm:"column:schedule_date;type:date;not null;index:idx_cs_date;index:idx_date_shift" json:"schedule_date"` // 排班日期，格式YYYY-MM-DD
	Status       int8      `gorm:"column:status;type:tinyint(1);not null" json:"status"`                                                // 状态: 0=正常, 1=请假, 2=换班
	ReplaceCsID  string    `gorm:"column:replace_cs_id;type:varchar(32)" json:"replace_cs_id"`                                          // 替班客服ID（换班时使用）
	CreateTime   time.Time `gorm:"column:create_time;not null" json:"create_time"`                                                      // 创建时间
	UpdateTime   time.Time `gorm:"column:update_time;not null" json:"update_time"`                                                      // 更新时间
}

// TableName 指定数据库表名
func (Schedule) TableName() string {
	return "t_schedule"
}

// ======================================
// 会话状态常量定义（状态机设计）
// ======================================
const (
	// ConvStatusWaiting 等待分配客服
	ConvStatusWaiting int8 = -1
	// ConvStatusOngoing 进行中
	ConvStatusOngoing int8 = 0
	// ConvStatusEnded 已结束
	ConvStatusEnded int8 = 1
	// ConvStatusTransferred 已转接
	ConvStatusTransferred int8 = 2
	// ConvStatusAbandoned 用户中途放弃
	ConvStatusAbandoned int8 = 3
)

// ConvStatusName 返回状态的中文名称
func ConvStatusName(status int8) string {
	switch status {
	case ConvStatusWaiting:
		return "等待中"
	case ConvStatusOngoing:
		return "进行中"
	case ConvStatusEnded:
		return "已结束"
	case ConvStatusTransferred:
		return "已转接"
	case ConvStatusAbandoned:
		return "已放弃"
	default:
		return "未知状态"
	}
}

// validConvTransitions 定义合法的状态转移规则
// key: 当前状态, value: 允许转移到的目标状态列表
var validConvTransitions = map[int8][]int8{
	ConvStatusWaiting:     {ConvStatusOngoing, ConvStatusAbandoned},                      // 等待 → 进行中/放弃
	ConvStatusOngoing:     {ConvStatusTransferred, ConvStatusEnded, ConvStatusAbandoned}, // 进行中 → 转接/结束/放弃
	ConvStatusTransferred: {ConvStatusOngoing, ConvStatusEnded, ConvStatusAbandoned},     // 已转接 → 进行中/结束/放弃
	ConvStatusEnded:       {},                                                            // 已结束 → 终态，不可转移
	ConvStatusAbandoned:   {},                                                            // 已放弃 → 终态，不可转移
}

// CanTransitionTo 检查是否可以从当前状态转移到目标状态
// 用于状态机验证，防止非法状态转移
func CanTransitionTo(currentStatus, nextStatus int8) bool {
	validNext, exists := validConvTransitions[currentStatus]
	if !exists {
		return false
	}
	for _, status := range validNext {
		if status == nextStatus {
			return true
		}
	}
	return false
}

// Conversation 会话表
// 记录用户与客服的会话信息
// 对应数据库表: t_conversation
// 状态机: waiting(-1) → ongoing(0) → transferred(2)/ended(1)/abandoned(3)
type Conversation struct {
	ConvID         string     `gorm:"column:conv_id;type:varchar(64);primaryKey" json:"conv_id"`                          // 会话ID，唯一标识
	UserID         string     `gorm:"column:user_id;type:varchar(64);not null;index:idx_user_time" json:"user_id"`        // 用户ID
	UserNickname   string     `gorm:"column:user_nickname;type:varchar(64)" json:"user_nickname"`                         // 用户昵称
	CsID           string     `gorm:"column:cs_id;type:varchar(32);not null;index:idx_cs_time" json:"cs_id"`              // 客服ID
	Source         string     `gorm:"column:source;type:varchar(32);not null" json:"source"`                              // 来源渠道，如Web、App
	StartTime      time.Time  `gorm:"column:start_time;not null;index:idx_user_time;index:idx_cs_time" json:"start_time"` // 开始时间
	EndTime        *time.Time `gorm:"column:end_time" json:"end_time"`                                                    // 结束时间：进行中会话为NULL，避免MySQL严格模式下写入0000-00-00导致报错
	LastMsgTime    time.Time  `gorm:"column:last_msg_time" json:"last_msg_time"`                                          // 最后消息时间（用于超时检测）
	MsgType        int8       `gorm:"column:msg_type;type:tinyint(1)" json:"msg_type"`                                    // 消息类型
	IsManualAdjust int8       `gorm:"column:is_manual_adjust;type:tinyint(1);not null" json:"is_manual_adjust"`           // 是否手动调整: 0=否, 1=是
	CategoryID     int64      `gorm:"column:category_id;not null;default:0;index:idx_category" json:"category_id"`        // 分类ID
	Tags           string     `gorm:"column:tags;type:varchar(256)" json:"tags"`                                          // 标签，逗号分隔
	IsCore         int8       `gorm:"column:is_core;type:tinyint(1);not null;default:0" json:"is_core"`                   // 是否核心会话: 0=否, 1=是
	Status         int8       `gorm:"column:status;type:tinyint(1);not null" json:"status"`                               // 状态: -1=等待, 0=进行中, 1=已结束, 2=已转接, 3=已放弃
	Version        int        `gorm:"column:version;type:int;not null;default:0" json:"version"`                          // 乐观锁版本号（并发控制）
	CreateTime     time.Time  `gorm:"column:create_time;not null" json:"create_time"`                                     // 创建时间
	UpdateTime     time.Time  `gorm:"column:update_time;not null" json:"update_time"`                                     // 更新时间
}

// TableName 指定数据库表名
func (Conversation) TableName() string {
	return "t_conversation"
}

// ConvTransfer 会话转接记录表
// 记录会话转接的历史和上下文信息
// 对应数据库表: t_conv_transfer
type ConvTransfer struct {
	TransferID     int64      `gorm:"column:transfer_id;primaryKey;autoIncrement" json:"transfer_id"`         // 转接记录ID
	ConvID         string     `gorm:"column:conv_id;type:varchar(64);not null;index:idx_conv" json:"conv_id"` // 会话ID
	FromCsID       string     `gorm:"column:from_cs_id;type:varchar(32);not null" json:"from_cs_id"`          // 转出客服ID
	FromCsName     string     `gorm:"column:from_cs_name;type:varchar(64)" json:"from_cs_name"`               // 转出客服名称
	ToCsID         string     `gorm:"column:to_cs_id;type:varchar(32);not null" json:"to_cs_id"`              // 转入客服ID
	ToCsName       string     `gorm:"column:to_cs_name;type:varchar(64)" json:"to_cs_name"`                   // 转入客服名称
	TransferReason string     `gorm:"column:transfer_reason;type:varchar(256)" json:"transfer_reason"`        // 转接原因
	ContextRemark  string     `gorm:"column:context_remark;type:text" json:"context_remark"`                  // 转接上下文(JSON结构化)
	TransferTime   time.Time  `gorm:"column:transfer_time;not null;index:idx_time" json:"transfer_time"`      // 转接时间
	AcceptTime     *time.Time `gorm:"column:accept_time" json:"accept_time"`                                  // 接受时间（改为指针以支持NULL）
	Status         int8       `gorm:"column:status;type:tinyint(1);not null;default:0" json:"status"`         // 状态: 0=待接受, 1=已接受, 2=已拒绝
}

// TableName 指定数据库表名
func (ConvTransfer) TableName() string {
	return "t_conv_transfer"
}

// TransferContext 转接上下文结构（存储在ContextRemark字段中）
// 用于记录转接时的详细信息，便于接收客服了解用户问题
type TransferContext struct {
	Category      string   `json:"category"`       // 问题分类：咨询|投诉|售后
	SubCategory   string   `json:"sub_category"`   // 子分类：发货延迟、退款问题等
	UserStatus    string   `json:"user_status"`    // 用户状态：VIP|普通|黑名单
	IssueDesc     string   `json:"issue_desc"`     // 问题描述（概要）
	UserSentiment string   `json:"user_sentiment"` // 用户情绪：满意|中立|不满意|愤怒
	ActionsTaken  []string `json:"actions_taken"`  // 已采取的行动
	Suggestions   string   `json:"suggestions"`    // 给接收客服的建议
	Priority      string   `json:"priority"`       // 优先级：high|medium|low
	OrderID       string   `json:"order_id"`       // 关联订单号（如有）
}

// ConvCategory 会话分类表
// 用于对会话进行分类管理，如"售前咨询"、"售后服务"、"投诉建议"等
// 对应数据库表: t_conv_category
type ConvCategory struct {
	CategoryID   int64     `gorm:"column:category_id;primaryKey;autoIncrement" json:"category_id"`                  // 分类ID
	CategoryName string    `gorm:"column:category_name;type:varchar(64);not null;uniqueIndex" json:"category_name"` // 分类名称（唯一）
	SortNo       int       `gorm:"column:sort_no;not null;default:0" json:"sort_no"`                                // 排序号
	CreateBy     string    `gorm:"column:create_by;type:varchar(32);not null" json:"create_by"`                     // 创建人
	CreateTime   time.Time `gorm:"column:create_time;not null" json:"create_time"`                                  // 创建时间
	UpdateTime   time.Time `gorm:"column:update_time;not null" json:"update_time"`                                  // 更新时间
}

// TableName 指定数据库表名
func (ConvCategory) TableName() string {
	return "t_conv_category"
}

// ConvMessage 会话消息表
// 存储会话中的具体消息记录
// 对应数据库表: t_conv_message
type ConvMessage struct {
	MsgID        int64     `gorm:"column:msg_id;primaryKey;autoIncrement" json:"msg_id"`                        // 消息ID
	ConvID       string    `gorm:"column:conv_id;type:varchar(64);not null;index:idx_conv_time" json:"conv_id"` // 会话ID
	SenderType   int8      `gorm:"column:sender_type;type:tinyint(1);not null" json:"sender_type"`              // 发送者类型: 0=用户, 1=客服, 2=系统
	SenderID     string    `gorm:"column:sender_id;type:varchar(64);not null" json:"sender_id"`                 // 发送者ID
	MsgContent   string    `gorm:"column:msg_content;type:text;not null" json:"msg_content"`                    // 消息内容
	FileURL      string    `gorm:"column:file_url;type:varchar(256)" json:"file_url"`                           // 文件URL（附件）
	FileType     string    `gorm:"column:file_type;type:varchar(32)" json:"file_type"`                          // 文件类型
	VoiceURL     string    `gorm:"column:voice_url;type:varchar(256)" json:"voice_url"`                         // 语音URL
	IsQuickReply int8      `gorm:"column:is_quick_reply;type:tinyint(1);not null" json:"is_quick_reply"`        // 是否快捷回复: 0=否, 1=是
	QuickReplyID int64     `gorm:"column:quick_reply_id" json:"quick_reply_id"`                                 // 快捷回复ID
	SendTime     time.Time `gorm:"column:send_time;not null;index:idx_conv_time" json:"send_time"`              // 发送时间
}

// TableName 指定数据库表名
func (ConvMessage) TableName() string {
	return "t_conv_message"
}

// QuickReply 快捷回复表
// 预设的常用回复语，用于客服快速响应用户
// 对应数据库表: t_quick_reply
type QuickReply struct {
	ReplyID      int64     `gorm:"column:reply_id;primaryKey;autoIncrement" json:"reply_id"`                           // 回复ID
	ReplyType    int8      `gorm:"column:reply_type;type:tinyint(1);not null;index:idx_type_public" json:"reply_type"` // 回复类型
	ReplyContent string    `gorm:"column:reply_content;type:text;not null" json:"reply_content"`                       // 回复内容
	CreateBy     string    `gorm:"column:create_by;type:varchar(32);not null" json:"create_by"`                        // 创建人
	IsPublic     int8      `gorm:"column:is_public;type:tinyint(1);not null;index:idx_type_public" json:"is_public"`   // 是否公开: 0=私有, 1=公开
	CreateTime   time.Time `gorm:"column:create_time;not null" json:"create_time"`                                     // 创建时间
	UpdateTime   time.Time `gorm:"column:update_time;not null" json:"update_time"`                                     // 更新时间
}

// TableName 指定数据库表名
func (QuickReply) TableName() string {
	return "t_quick_reply"
}

// LeaveTransfer 请假调班申请表
// 记录客服的请假和调班申请，支持审批流程
// 对应数据库表: t_leave_transfer
type LeaveTransfer struct {
	ApplyID        int64      `gorm:"column:apply_id;primaryKey;autoIncrement" json:"apply_id"`                                   // 申请ID
	CsID           string     `gorm:"column:cs_id;type:varchar(32);not null;index:idx_cs_status" json:"cs_id"`                    // 申请人客服ID
	ApplyType      int8       `gorm:"column:apply_type;type:tinyint(1);not null" json:"apply_type"`                               // 申请类型: 0=请假, 1=换班
	LeaveType      int8       `gorm:"column:leave_type;type:tinyint(1);default:0" json:"leave_type"`                              // 请假类型: 0=事假, 1=病假, 2=年假, 3=调休, 4=其他
	TargetDate     string     `gorm:"column:target_date;type:date;not null" json:"target_date"`                                   // 目标日期（兼容历史数据）
	StartDate      string     `gorm:"column:start_date;type:date" json:"start_date"`                                              // 开始日期
	EndDate        string     `gorm:"column:end_date;type:date" json:"end_date"`                                                  // 结束日期
	StartPeriod    int8       `gorm:"column:start_period;type:tinyint(1);default:0" json:"start_period"`                          // 开始时段: 0=全天, 1=上午, 2=下午
	EndPeriod      int8       `gorm:"column:end_period;type:tinyint(1);default:0" json:"end_period"`                              // 结束时段: 0=全天, 1=上午, 2=下午
	ShiftID        int64      `gorm:"column:shift_id;not null" json:"shift_id"`                                                   // 班次ID
	TargetCsID     string     `gorm:"column:target_cs_id;type:varchar(32)" json:"target_cs_id"`                                   // 目标客服ID（换班时使用）
	ApprovalStatus int8       `gorm:"column:approval_status;type:tinyint(1);not null;index:idx_cs_status" json:"approval_status"` // 审批状态: 0=待审批, 1=已通过, 2=已拒绝
	ApproverID     string     `gorm:"column:approver_id;type:varchar(32)" json:"approver_id"`                                     // 审批人ID
	ApproverName   string     `gorm:"column:approver_name;type:varchar(64)" json:"approver_name"`                                 // 审批人姓名
	ApprovalTime   *time.Time `gorm:"column:approval_time" json:"approval_time"`                                                  // 审批时间
	ApprovalRemark string     `gorm:"column:approval_remark;type:varchar(256)" json:"approval_remark"`                            // 审批备注
	Reason         string     `gorm:"column:reason;type:varchar(256)" json:"reason"`                                              // 申请理由
	Attachments    string     `gorm:"column:attachments;type:varchar(512)" json:"attachments"`                                    // 附件URL列表(JSON数组)
	ApproverRole   string     `gorm:"column:approver_role;type:varchar(32)" json:"approver_role"`                                 // 审批人角色
	CreateTime     time.Time  `gorm:"column:create_time;not null" json:"create_time"`                                             // 创建时间
	UpdateTime     time.Time  `gorm:"column:update_time;not null" json:"update_time"`                                             // 更新时间
}

// TableName 指定数据库表名
func (LeaveTransfer) TableName() string {
	return "t_leave_transfer"
}

// LeaveAuditLog 审批链路记录表
// 记录请假/调班申请的每一次状态变更和批注
type LeaveAuditLog struct {
	LogID        int64     `gorm:"column:log_id;primaryKey;autoIncrement" json:"log_id"`
	ApplyID      int64     `gorm:"column:apply_id;not null;index" json:"apply_id"`
	Action       string    `gorm:"column:action;type:varchar(32);not null" json:"action"` // SUBMIT, APPROVE, REJECT, CANCEL
	OperatorID   string    `gorm:"column:operator_id;type:varchar(32);not null" json:"operator_id"`
	OperatorName string    `gorm:"column:operator_name;type:varchar(64)" json:"operator_name"`
	OperatorRole string    `gorm:"column:operator_role;type:varchar(32)" json:"operator_role"`
	Remark       string    `gorm:"column:remark;type:varchar(256)" json:"remark"`
	CreateTime   time.Time `gorm:"column:create_time;not null" json:"create_time"`
}

func (LeaveAuditLog) TableName() string {
	return "t_leave_audit_log"
}

// CsPresence 在线状态缓存结构（存储在 Redis 中）
type CsPresence struct {
	CsID       string    `json:"cs_id"`
	IsOnline   bool      `json:"is_online"`
	LastSeenAt time.Time `json:"last_seen_at"`
	LoginAt    time.Time `json:"login_at"`
	Device     string    `json:"device"`
	IP         string    `json:"ip"`
}

// ConvTag 会话标签表
// 用于给会话打标签，方便分类和统计
// 对应数据库表: t_conv_tag
type ConvTag struct {
	TagID      int64     `gorm:"column:tag_id;primaryKey;autoIncrement" json:"tag_id"`                  // 标签ID
	TagName    string    `gorm:"column:tag_name;type:varchar(32);not null;uniqueIndex" json:"tag_name"` // 标签名称（唯一）
	TagColor   string    `gorm:"column:tag_color;type:varchar(16);default:'#1890ff'" json:"tag_color"`  // 标签颜色（十六进制）
	SortNo     int       `gorm:"column:sort_no;default:0;index" json:"sort_no"`                         // 排序号
	CreateBy   string    `gorm:"column:create_by;type:varchar(32);not null" json:"create_by"`           // 创建人
	CreateTime time.Time `gorm:"column:create_time;not null" json:"create_time"`                        // 创建时间
	UpdateTime time.Time `gorm:"column:update_time;not null" json:"update_time"`                        // 更新时间
}

// TableName 指定数据库表名
func (ConvTag) TableName() string {
	return "t_conv_tag"
}

// MsgCategory 消息分类维度表
// 定义消息的分类维度，如咨询类、投诉类、建议类、其他类
// 对应数据库表: t_msg_category
type MsgCategory struct {
	CategoryID   int64     `gorm:"column:category_id;primaryKey;autoIncrement" json:"category_id"`                  // 分类ID
	CategoryName string    `gorm:"column:category_name;type:varchar(64);not null;uniqueIndex" json:"category_name"` // 分类名称
	Keywords     string    `gorm:"column:keywords;type:text" json:"keywords"`                                       // 关键词列表(JSON)
	SortNo       int       `gorm:"column:sort_no;not null;default:0" json:"sort_no"`                                // 排序号
	CreateBy     string    `gorm:"column:create_by;type:varchar(32);not null" json:"create_by"`                     // 创建人
	CreateTime   time.Time `gorm:"column:create_time;not null" json:"create_time"`                                  // 创建时间
	UpdateTime   time.Time `gorm:"column:update_time;not null" json:"update_time"`                                  // 更新时间
}

// TableName 指定数据库表名
func (MsgCategory) TableName() string {
	return "t_msg_category"
}

// ClassifyAdjustLog 分类调整日志表
// 记录人工调整分类的历史，用于统计和模型优化
// 对应数据库表: t_classify_adjust_log
type ClassifyAdjustLog struct {
	LogID              int64     `gorm:"column:log_id;primaryKey;autoIncrement" json:"log_id"`                   // 日志ID
	ConvID             string    `gorm:"column:conv_id;type:varchar(64);not null;index:idx_conv" json:"conv_id"` // 会话ID
	OriginalCategoryID int64     `gorm:"column:original_category_id;not null" json:"original_category_id"`       // 原分类ID
	NewCategoryID      int64     `gorm:"column:new_category_id;not null" json:"new_category_id"`                 // 新分类ID
	OperatorID         string    `gorm:"column:operator_id;type:varchar(32);not null" json:"operator_id"`        // 操作人ID
	AdjustReason       string    `gorm:"column:adjust_reason;type:varchar(256)" json:"adjust_reason"`            // 调整原因
	CreateTime         time.Time `gorm:"column:create_time;not null;index:idx_time" json:"create_time"`          // 创建时间
}

// TableName 指定数据库表名
func (ClassifyAdjustLog) TableName() string {
	return "t_classify_adjust_log"
}

// ============ 数据归档模型 ============

// ArchivedConversation 已归档会话表
// 存储超过保留期的历史会话（冷数据）
// 对应数据库表: t_archived_conversation
type ArchivedConversation struct {
	ArchiveID     int64     `gorm:"column:archive_id;primaryKey;autoIncrement" json:"archive_id"`
	ConvID        string    `gorm:"column:conv_id;type:varchar(64);not null;uniqueIndex" json:"conv_id"`
	UserID        string    `gorm:"column:user_id;type:varchar(64);not null;index:idx_user" json:"user_id"`
	CsID          string    `gorm:"column:cs_id;type:varchar(32);not null;index:idx_cs" json:"cs_id"`
	ConvData      string    `gorm:"column:conv_data;type:mediumtext" json:"conv_data"` // 完整会话数据JSON
	MsgCount      int       `gorm:"column:msg_count;not null;default:0" json:"msg_count"`
	OriginalDate  time.Time `gorm:"column:original_date;not null;index:idx_date" json:"original_date"`
	ArchiveTime   time.Time `gorm:"column:archive_time;not null" json:"archive_time"`
	RetentionDays int       `gorm:"column:retention_days;not null;default:365" json:"retention_days"` // 归档保留天数
}

// TableName 指定数据库表名
func (ArchivedConversation) TableName() string {
	return "t_archived_conversation"
}

// ArchivedMessage 已归档消息表
// 存储超过保留期的历史消息
// 对应数据库表: t_archived_message
type ArchivedMessage struct {
	ArchiveID    int64     `gorm:"column:archive_id;primaryKey;autoIncrement" json:"archive_id"`
	MsgID        int64     `gorm:"column:msg_id;not null;uniqueIndex" json:"msg_id"`
	ConvID       string    `gorm:"column:conv_id;type:varchar(64);not null;index:idx_conv" json:"conv_id"`
	MsgData      string    `gorm:"column:msg_data;type:text" json:"msg_data"` // 完整消息数据JSON
	OriginalDate time.Time `gorm:"column:original_date;not null;index:idx_date" json:"original_date"`
	ArchiveTime  time.Time `gorm:"column:archive_time;not null" json:"archive_time"`
}

// TableName 指定数据库表名
func (ArchivedMessage) TableName() string {
	return "t_archived_message"
}

// ArchiveTask 归档任务记录表
// 记录每次归档操作的执行情况
// 对应数据库表: t_archive_task
type ArchiveTask struct {
	TaskID        int64     `gorm:"column:task_id;primaryKey;autoIncrement" json:"task_id"`
	TaskType      string    `gorm:"column:task_type;type:varchar(32);not null" json:"task_type"` // conv/message
	StartDate     string    `gorm:"column:start_date;type:date;not null" json:"start_date"`
	EndDate       string    `gorm:"column:end_date;type:date;not null" json:"end_date"`
	ArchivedCount int64     `gorm:"column:archived_count;not null;default:0" json:"archived_count"`
	DeletedCount  int64     `gorm:"column:deleted_count;not null;default:0" json:"deleted_count"`
	Status        int8      `gorm:"column:status;type:tinyint(1);not null" json:"status"` // 0-进行中 1-完成 2-失败
	ErrorMsg      string    `gorm:"column:error_msg;type:text" json:"error_msg"`
	StartTime     time.Time `gorm:"column:start_time;not null" json:"start_time"`
	EndTime       time.Time `gorm:"column:end_time" json:"end_time"`
	OperatorID    string    `gorm:"column:operator_id;type:varchar(32)" json:"operator_id"`
}

// TableName 指定数据库表名
func (ArchiveTask) TableName() string {
	return "t_archive_task"
}
