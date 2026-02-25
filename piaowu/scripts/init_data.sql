SET NAMES utf8mb4;
USE ccc;

-- 插入客服数据 (role字段是tinyint: 0=普通, 1=客服, 2=管理员)
INSERT INTO t_customer_service (cs_id, cs_name, dept_id, team_id, skill_tags, status, current_status, is_online, role, password_hash, create_time, update_time) VALUES
('CS001', '张三', '1', '1', '售前,购票', 1, 0, 0, 1, 'hash123', NOW(), NOW()),
('CS002', '李四', '1', '1', '售后,退票', 1, 0, 0, 1, 'hash123', NOW(), NOW()),
('CS003', '王五', '2', '1', '技术支持', 1, 0, 0, 1, 'hash123', NOW(), NOW()),
('CS9999', 'AI智能助手', '0', '0', 'AI,智能客服', 1, 0, 1, 1, 'hash123', NOW(), NOW())
ON DUPLICATE KEY UPDATE cs_name=VALUES(cs_name);

-- 插入角色数据
INSERT IGNORE INTO sys_roles (id, role_code, role_name, remark, created_at, updated_at) VALUES
(1, 'admin', '管理员', '系统管理员', NOW(), NOW()),
(2, 'customer_service', '客服', '客服人员', NOW(), NOW()),
(3, 'user', '普通用户', '用户端权限', NOW(), NOW());

-- 插入班次配置
INSERT IGNORE INTO t_shift_config (shift_id, shift_name, start_time, end_time, min_staff, is_holiday, create_by, create_time, update_time) VALUES
(1, '早班', '2026-01-01 08:00:00', '2026-01-01 12:00:00', 2, 0, 'admin', NOW(), NOW()),
(2, '午班', '2026-01-01 12:00:00', '2026-01-01 18:00:00', 3, 0, 'admin', NOW(), NOW()),
(3, '晚班', '2026-01-01 18:00:00', '2026-01-01 22:00:00', 2, 0, 'admin', NOW(), NOW());

-- 插入会话分类
INSERT IGNORE INTO t_conv_category (category_id, category_name, sort_no, create_by, create_time, update_time) VALUES
(1, '售前咨询', 1, 'admin', NOW(), NOW()),
(2, '售后服务', 2, 'admin', NOW(), NOW()),
(3, '退换货处理', 3, 'admin', NOW(), NOW()),
(4, '投诉处理', 4, 'admin', NOW(), NOW());

-- 插入快捷回复
INSERT IGNORE INTO t_quick_reply (reply_id, reply_type, reply_content, create_by, is_public, create_time, update_time) VALUES
(1, 1, '您好，欢迎咨询票务服务，请问有什么可以帮您？', 'admin', 1, NOW(), NOW()),
(2, 2, '好的，我已经收到您的问题，正在为您查询。', 'admin', 1, NOW(), NOW()),
(3, 3, '感谢您的咨询，祝您旅途愉快！', 'admin', 1, NOW(), NOW());

SELECT 'Init Complete!' as result;
SELECT COUNT(*) as cs_count FROM t_customer_service;
