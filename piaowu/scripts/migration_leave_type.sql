-- 优化方案：添加请假类型和附件字段
-- 执行方式: mysql -u root -p customer_db < migration_leave_type.sql

-- 添加请假类型字段 (0=事假, 1=病假, 2=年假, 3=调休, 4=其他)
ALTER TABLE `t_leave_transfer`
ADD COLUMN `leave_type` TINYINT(1) DEFAULT 0 COMMENT '请假类型: 0=事假, 1=病假, 2=年假, 3=调休, 4=其他';

-- 添加附件字段 (JSON数组格式)
ALTER TABLE `t_leave_transfer`
ADD COLUMN `attachments` VARCHAR(512) NULL COMMENT '附件URL列表(JSON数组)';
