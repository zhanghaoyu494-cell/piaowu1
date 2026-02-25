-- 修复 MySQL 字符集乱码问题
-- 1. 设置连接字符集
SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;
SET character_set_connection=utf8mb4;

-- 2. 删除存储过程生成的乱码数据（保留基础系统数据）
USE ccc;

-- 禁用外键检查
SET FOREIGN_KEY_CHECKS = 0;

-- 删除消息相关数据
TRUNCATE TABLE t_conv_message;
TRUNCATE TABLE t_conversation;
TRUNCATE TABLE t_conv_transfer;
TRUNCATE TABLE t_conversation_summary;
TRUNCATE TABLE ai_jobs;
TRUNCATE TABLE t_ai_job;
TRUNCATE TABLE conversation_summaries;

-- 删除排班和请假数据
TRUNCATE TABLE t_schedule;
TRUNCATE TABLE t_leave_transfer;
TRUNCATE TABLE t_leave_audit_log;

-- 删除存储过程创建的用户数据（保留基础数据）
DELETE FROM sys_users WHERE user_name LIKE 'user_%';
DELETE FROM t_customer_service WHERE cs_id LIKE 'CS%' AND cs_id != 'CS9999';

-- 3. 重新启用外键检查
SET FOREIGN_KEY_CHECKS = 1;

-- 4. 重新插入正确编码的客服数据
INSERT INTO t_customer_service (cs_id, cs_name, dept_id, team_id, skill_tags, status, current_status, is_online, role, password_hash, create_time, update_time) VALUES
('CS001', '张伟', 'D001', 'T001', '购票,改签,退票', 1, 1, 1, 0, '', NOW(), NOW()),
('CS002', '李娟', 'D001', 'T001', '退票,投诉处理', 1, 1, 1, 0, '', NOW(), NOW()),
('CS003', '王强', 'D001', 'T002', '团体票,企业服务', 1, 1, 1, 0, '', NOW(), NOW()),
('CS004', '刘芳', 'D002', 'T003', 'VIP服务,高端客户', 1, 1, 1, 0, '', NOW(), NOW()),
('CS005', '陈明', 'D002', 'T003', '技术支持,系统问题', 1, 1, 1, 0, '', NOW(), NOW()),
('CS9999', 'AI助理', 'AI', 'AI', 'AI自动回复,智能客服', 1, 1, 1, 2, '', NOW(), NOW())
ON DUPLICATE KEY UPDATE cs_name=VALUES(cs_name), skill_tags=VALUES(skill_tags);

-- 5. 重新插入正确编码的用户数据  
INSERT INTO sys_users (user_name, password, real_name, phone, role_code, status, created_at, updated_at) VALUES
('user_001', '$2a$10$abcdefghijklmnopqrstuvwxyz123456', '张三', '13800138001', 'ROLE_USER', 1, NOW(), NOW()),
('user_002', '$2a$10$abcdefghijklmnopqrstuvwxyz123456', '李四', '13800138002', 'ROLE_USER', 1, NOW(), NOW()),
('user_003', '$2a$10$abcdefghijklmnopqrstuvwxyz123456', '王五', '13800138003', 'ROLE_USER', 1, NOW(), NOW()),
('user_004', '$2a$10$abcdefghijklmnopqrstuvwxyz123456', '赵六', '13800138004', 'ROLE_USER', 1, NOW(), NOW()),
('user_005', '$2a$10$abcdefghijklmnopqrstuvwxyz123456', '孙七', '13800138005', 'ROLE_USER', 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE real_name=VALUES(real_name);

-- 6. 插入一些示例对话（使用正确的表结构）
INSERT INTO t_conversation (conv_id, user_id, user_nickname, cs_id, source, start_time, status, is_manual_adjust, create_time, update_time) VALUES
('CONV001', 'user_001', '张三', 'CS9999', 'web', NOW(), 1, 0, NOW(), NOW()),
('CONV002', 'user_002', '李四', 'CS001', 'web', NOW(), 1, 0, NOW(), NOW()),
('CONV003', 'user_003', '王五', 'CS002', 'web', NOW(), 1, 0, NOW(), NOW());

-- 7. 插入示例消息
INSERT INTO t_conv_message (conv_id, sender_type, sender_id, msg_content, send_time, create_time, update_time) VALUES
('CONV001', 1, 'user_001', '您好，我想咨询一下购票问题', NOW(), NOW(), NOW()),
('CONV001', 2, 'CS9999', '您好！我是AI客服助理，请问有什么可以帮您的？', NOW(), NOW(), NOW()),
('CONV002', 1, 'user_002', '请问怎么退票？', NOW(), NOW(), NOW()),
('CONV002', 3, 'CS001', '您好，退票需要提供订单号，请问您的订单号是多少？', NOW(), NOW(), NOW()),
('CONV003', 1, 'user_003', '我的票改签可以吗？', NOW(), NOW(), NOW()),
('CONV003', 3, 'CS002', '可以的，请告诉我您想改签到什么时间？', NOW(), NOW(), NOW());

SELECT '数据修复完成！' as result;
