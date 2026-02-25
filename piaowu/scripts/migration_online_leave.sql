-- 客服排班管理系统功能增强 - 数据库迁移脚本
-- 执行时间: 2026-01-23
-- 功能: 在线状态显示、请假日期范围、角色权限、审批人记录

-- ==============================================
-- 1. t_customer_service 表新增字段
-- ==============================================

-- 在线状态: 0=离线, 1=在线
ALTER TABLE t_customer_service 
ADD COLUMN is_online TINYINT(1) NOT NULL DEFAULT 0 COMMENT '在线状态: 0=离线, 1=在线';

-- 最后心跳时间
ALTER TABLE t_customer_service 
ADD COLUMN last_heartbeat DATETIME COMMENT '最后心跳时间';

-- 角色: 0=客服, 1=部门经理, 2=管理员
ALTER TABLE t_customer_service 
ADD COLUMN role TINYINT(1) NOT NULL DEFAULT 0 COMMENT '角色: 0=客服, 1=部门经理, 2=管理员';

-- 密码哈希(用于登录验证)
ALTER TABLE t_customer_service 
ADD COLUMN password_hash VARCHAR(128) COMMENT '密码哈希';

-- ==============================================
-- 2. t_leave_transfer 表新增字段
-- ==============================================

-- 开始日期（支持日期范围）
ALTER TABLE t_leave_transfer 
ADD COLUMN start_date DATE COMMENT '开始日期';

-- 结束日期
ALTER TABLE t_leave_transfer 
ADD COLUMN end_date DATE COMMENT '结束日期';

-- 开始时段: 0=全天, 1=上午, 2=下午
ALTER TABLE t_leave_transfer 
ADD COLUMN start_period TINYINT(1) DEFAULT 0 COMMENT '开始时段: 0=全天, 1=上午, 2=下午';

-- 结束时段
ALTER TABLE t_leave_transfer 
ADD COLUMN end_period TINYINT(1) DEFAULT 0 COMMENT '结束时段: 0=全天, 1=上午, 2=下午';

-- 审批人姓名（便于显示）
ALTER TABLE t_leave_transfer 
ADD COLUMN approver_name VARCHAR(64) COMMENT '审批人姓名';

-- 审批备注
ALTER TABLE t_leave_transfer 
ADD COLUMN approval_remark VARCHAR(256) COMMENT '审批备注';

-- ==============================================
-- 3. 历史数据迁移（将target_date同步到start_date/end_date）
-- ==============================================

UPDATE t_leave_transfer 
SET start_date = target_date, end_date = target_date 
WHERE start_date IS NULL AND target_date IS NOT NULL;

-- ==============================================
-- 4. 设置默认管理员角色（可选）
-- ==============================================

-- 将现有的ADMIN用户设置为管理员角色
-- UPDATE t_customer_service SET role = 2 WHERE cs_id = 'ADMIN';
