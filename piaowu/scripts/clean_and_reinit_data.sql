-- 清理乱码数据并重新初始化
-- 执行方式: docker exec -i piaowu-mysql mysql -uroot -pZhyzhy666888 < scripts/clean_and_reinit_data.sql

SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;
SET character_set_connection=utf8mb4;

USE ccc;

-- 禁用外键检查
SET FOREIGN_KEY_CHECKS = 0;

-- =====================================================
-- 1. 清理所有有乱码的表
-- =====================================================

-- 清理角色表（保留前两个正常的角色，删除乱码的）
DELETE FROM sys_roles WHERE id > 2;

-- 清理分类和标签表
TRUNCATE TABLE t_conv_category;
TRUNCATE TABLE t_conv_tag;
TRUNCATE TABLE t_msg_category;
TRUNCATE TABLE t_quick_reply;
TRUNCATE TABLE t_shift_config;
TRUNCATE TABLE t_schedule;

-- 清理对话相关表
TRUNCATE TABLE t_conv_message;
TRUNCATE TABLE t_conversation;
TRUNCATE TABLE t_conv_transfer;
TRUNCATE TABLE t_conversation_summary;
TRUNCATE TABLE ai_jobs;
TRUNCATE TABLE t_ai_job;
TRUNCATE TABLE conversation_summaries;
TRUNCATE TABLE t_ai_suggestion_log;
TRUNCATE TABLE t_ai_tool_audit_log;
TRUNCATE TABLE t_classify_adjust_log;
TRUNCATE TABLE t_leave_audit_log;
TRUNCATE TABLE t_leave_transfer;
TRUNCATE TABLE t_risk_alert_log;

-- =====================================================
-- 2. 重新插入角色数据
-- =====================================================
INSERT INTO sys_roles (id, role_code, role_name, remark, created_at, updated_at) VALUES
(5, 'user', '普通用户', '用户端权限', NOW(), NOW()),
(6, 'approver', '审批员', '负责审批请假和调班申请', NOW(), NOW()),
(7, 'super_admin', '超级管理员', '最高权限管理员', NOW(), NOW())
ON DUPLICATE KEY UPDATE role_name=VALUES(role_name), remark=VALUES(remark);

-- =====================================================
-- 3. 重新插入会话分类数据 t_conv_category
-- =====================================================
INSERT INTO t_conv_category (category_id, category_name, sort_no, create_by, create_time, update_time) VALUES
(1, '售前咨询', 1, 'admin', NOW(), NOW()),
(2, '售后服务', 2, 'admin', NOW(), NOW()),
(3, '退换货处理', 3, 'admin', NOW(), NOW()),
(4, '投诉处理', 4, 'admin', NOW(), NOW()),
(5, '技术支持', 5, 'admin', NOW(), NOW()),
(6, '账户问题', 6, 'admin', NOW(), NOW()),
(7, '支付问题', 7, 'admin', NOW(), NOW()),
(8, '物流查询', 8, 'admin', NOW(), NOW()),
(9, '产品咨询', 9, 'admin', NOW(), NOW()),
(10, '优惠活动', 10, 'admin', NOW(), NOW());

-- =====================================================
-- 4. 重新插入会话标签数据 t_conv_tag
-- =====================================================
INSERT INTO t_conv_tag (tag_id, tag_name, tag_color, sort_no, create_by, create_time, update_time) VALUES
(1, 'VIP客户', '#FF4D4F', 1, 'admin', NOW(), NOW()),
(2, '高优先级', '#FA541C', 2, 'admin', NOW(), NOW()),
(3, '中优先级', '#FA8C16', 3, 'admin', NOW(), NOW()),
(4, '低优先级', '#52C41A', 4, 'admin', NOW(), NOW()),
(5, '投诉中', '#F5222D', 5, 'admin', NOW(), NOW()),
(6, '已解决', '#52C41A', 6, 'admin', NOW(), NOW()),
(7, '待跟进', '#1890FF', 7, 'admin', NOW(), NOW()),
(8, '需回访', '#722ED1', 8, 'admin', NOW(), NOW()),
(9, '紧急处理', '#EB2F96', 9, 'admin', NOW(), NOW()),
(10, '转接人工', '#13C2C2', 10, 'admin', NOW(), NOW());

-- =====================================================
-- 5. 重新插入消息分类数据 t_msg_category
-- =====================================================
INSERT INTO t_msg_category (category_id, category_name, keywords, sort_no, create_by, create_time, update_time) VALUES
(1, '购票咨询', '买票,购票,订票,怎么买,票价', 1, 'admin', NOW(), NOW()),
(2, '退票服务', '退票,退款,取消订单,不想要了', 2, 'admin', NOW(), NOW()),
(3, '改签服务', '改签,换票,更改日期,改时间', 3, 'admin', NOW(), NOW()),
(4, '行程查询', '查询,订单,行程,我的票', 4, 'admin', NOW(), NOW()),
(5, '座位选择', '选座,换座,靠窗,过道', 5, 'admin', NOW(), NOW()),
(6, '支付问题', '支付,付款,扣款,支付失败', 6, 'admin', NOW(), NOW()),
(7, '发票服务', '发票,报销,开票,电子发票', 7, 'admin', NOW(), NOW()),
(8, '会员服务', '会员,积分,等级,权益', 8, 'admin', NOW(), NOW()),
(9, '优惠活动', '优惠,折扣,满减,促销', 9, 'admin', NOW(), NOW()),
(10, '账号问题', '登录,注册,密码,账号', 10, 'admin', NOW(), NOW());

-- =====================================================
-- 6. 重新插入快捷回复数据 t_quick_reply
-- =====================================================
INSERT INTO t_quick_reply (reply_id, reply_type, reply_content, create_by, is_public, create_time, update_time) VALUES
-- 问候语 (type=1)
(1, 1, '您好，欢迎咨询票务服务，请问有什么可以帮您？', 'admin', 1, NOW(), NOW()),
(2, 1, '您好，我是您的专属客服，很高兴为您服务！', 'admin', 1, NOW(), NOW()),
(3, 1, '感谢您的耐心等待，请问有什么需要帮助的吗？', 'admin', 1, NOW(), NOW()),
(4, 1, '您好，请问您需要咨询什么业务呢？', 'admin', 1, NOW(), NOW()),
(5, 1, '欢迎回来，请问这次需要什么帮助？', 'admin', 1, NOW(), NOW()),
-- 确认语 (type=2)
(6, 2, '好的，我已经收到您的问题，正在为您查询。', 'admin', 1, NOW(), NOW()),
(7, 2, '收到，请您稍等，我马上为您处理。', 'admin', 1, NOW(), NOW()),
(8, 2, '明白了，让我来帮您解决这个问题。', 'admin', 1, NOW(), NOW()),
(9, 2, '好的，我已经记录下来了，请稍候。', 'admin', 1, NOW(), NOW()),
(10, 2, '了解，我这就为您查询相关信息。', 'admin', 1, NOW(), NOW()),
-- 结束语 (type=3)
(11, 3, '感谢您的咨询，祝您旅途愉快！', 'admin', 1, NOW(), NOW()),
(12, 3, '如有其他问题，欢迎随时咨询我们。', 'admin', 1, NOW(), NOW()),
(13, 3, '感谢您的信任，期待再次为您服务！', 'admin', 1, NOW(), NOW()),
(14, 3, '祝您生活愉快，再见！', 'admin', 1, NOW(), NOW()),
(15, 3, '感谢您的耐心配合，再见！', 'admin', 1, NOW(), NOW()),
-- 购票相关 (type=4)
(16, 4, '请问您需要购买哪天的票？', 'admin', 1, NOW(), NOW()),
(17, 4, '您好，请问需要购买几张票？', 'admin', 1, NOW(), NOW()),
(18, 4, '请提供您的出发日期和目的地。', 'admin', 1, NOW(), NOW()),
-- 退票相关 (type=5)
(19, 5, '请提供您的订单号，我为您查询退票信息。', 'admin', 1, NOW(), NOW()),
(20, 5, '退票手续费按照票价的5%-15%收取。', 'admin', 1, NOW(), NOW()),
(21, 5, '退票成功后，款项将在3-7个工作日内退回。', 'admin', 1, NOW(), NOW()),
-- 改签相关 (type=6)
(22, 6, '请问您想改签到什么时间？', 'admin', 1, NOW(), NOW()),
(23, 6, '改签需要补差价或退差价，具体以实际为准。', 'admin', 1, NOW(), NOW()),
(24, 6, '已为您改签成功，请注意查收新的行程信息。', 'admin', 1, NOW(), NOW());

-- =====================================================
-- 7. 重新插入班次配置数据 t_shift_config
-- =====================================================
INSERT INTO t_shift_config (shift_id, shift_name, start_time, end_time, min_staff, is_holiday, create_by, create_time, update_time) VALUES
(1, '早班A', '2026-01-01 08:00:00', '2026-01-01 12:00:00', 5, 0, 'admin', NOW(), NOW()),
(2, '早班B', '2026-01-01 08:30:00', '2026-01-01 12:30:00', 4, 0, 'admin', NOW(), NOW()),
(3, '午班A', '2026-01-01 12:00:00', '2026-01-01 18:00:00', 6, 0, 'admin', NOW(), NOW()),
(4, '午班B', '2026-01-01 12:30:00', '2026-01-01 18:30:00', 5, 0, 'admin', NOW(), NOW()),
(5, '晚班A', '2026-01-01 18:00:00', '2026-01-01 22:00:00', 4, 0, 'admin', NOW(), NOW()),
(6, '晚班B', '2026-01-01 18:30:00', '2026-01-01 23:00:00', 3, 0, 'admin', NOW(), NOW()),
(7, '夜班', '2026-01-01 22:00:00', '2026-01-02 06:00:00', 2, 0, 'admin', NOW(), NOW()),
(8, '周末早班', '2026-01-01 09:00:00', '2026-01-01 15:00:00', 3, 1, 'admin', NOW(), NOW()),
(9, '周末晚班', '2026-01-01 15:00:00', '2026-01-01 21:00:00', 3, 1, 'admin', NOW(), NOW()),
(10, '节假日班', '2026-01-01 10:00:00', '2026-01-01 18:00:00', 2, 1, 'admin', NOW(), NOW());

-- =====================================================
-- 8. 重新插入排班数据 t_schedule
-- =====================================================
INSERT INTO t_schedule (cs_id, shift_id, schedule_date, status, replace_cs_id, create_time, update_time) VALUES
-- 今天的排班 (2026-01-30)
('CS001', 1, '2026-01-30', 0, '', NOW(), NOW()),
('CS002', 2, '2026-01-30', 0, '', NOW(), NOW()),
('CS003', 3, '2026-01-30', 0, '', NOW(), NOW()),
('CS004', 4, '2026-01-30', 0, '', NOW(), NOW()),
('CS005', 5, '2026-01-30', 0, '', NOW(), NOW()),
('CS9999', 1, '2026-01-30', 0, '', NOW(), NOW()),
-- 明天的排班 (2026-01-31)
('CS001', 3, '2026-01-31', 0, '', NOW(), NOW()),
('CS002', 4, '2026-01-31', 0, '', NOW(), NOW()),
('CS003', 5, '2026-01-31', 0, '', NOW(), NOW()),
('CS004', 6, '2026-01-31', 0, '', NOW(), NOW()),
('CS005', 1, '2026-01-31', 0, '', NOW(), NOW()),
('CS9999', 1, '2026-01-31', 0, '', NOW(), NOW()),
-- 后天的排班 (2026-02-01)
('CS001', 5, '2026-02-01', 0, '', NOW(), NOW()),
('CS002', 6, '2026-02-01', 0, '', NOW(), NOW()),
('CS003', 1, '2026-02-01', 0, '', NOW(), NOW()),
('CS004', 2, '2026-02-01', 0, '', NOW(), NOW()),
('CS005', 3, '2026-02-01', 0, '', NOW(), NOW()),
('CS9999', 1, '2026-02-01', 0, '', NOW(), NOW());

-- =====================================================
-- 9. 重新插入示例对话数据
-- =====================================================
INSERT INTO t_conversation (conv_id, user_id, user_nickname, cs_id, source, start_time, status, is_manual_adjust, create_time, update_time) VALUES
('CONV001', 'user_001', '张三', 'CS9999', 'web', NOW(), 1, 0, NOW(), NOW()),
('CONV002', 'user_002', '李四', 'CS001', 'web', NOW(), 1, 0, NOW(), NOW()),
('CONV003', 'user_003', '王五', 'CS002', 'web', NOW(), 1, 0, NOW(), NOW()),
('CONV004', 'U_123', '刘德华', 'CS003', 'app', NOW(), 1, 0, NOW(), NOW()),
('CONV005', 'user_004', '赵六', 'CS9999', 'wechat', NOW(), 1, 0, NOW(), NOW());

-- =====================================================
-- 10. 重新插入示例消息数据
-- =====================================================
INSERT INTO t_conv_message (conv_id, sender_type, sender_id, msg_content, send_time, create_time, update_time) VALUES
-- 对话1: 用户咨询AI
('CONV001', 1, 'user_001', '您好，我想咨询一下购票问题', NOW(), NOW(), NOW()),
('CONV001', 2, 'CS9999', '您好！我是AI客服助理，请问有什么可以帮您的？', NOW(), NOW(), NOW()),
('CONV001', 1, 'user_001', '我想买一张明天去北京的高铁票', NOW(), NOW(), NOW()),
('CONV001', 2, 'CS9999', '好的，请问您从哪个城市出发呢？我来帮您查询。', NOW(), NOW(), NOW()),
-- 对话2: 退票咨询
('CONV002', 1, 'user_002', '请问怎么退票？', NOW(), NOW(), NOW()),
('CONV002', 3, 'CS001', '您好，退票需要提供订单号，请问您的订单号是多少？', NOW(), NOW(), NOW()),
('CONV002', 1, 'user_002', '我的订单号是202601290001', NOW(), NOW(), NOW()),
('CONV002', 3, 'CS001', '好的，我来帮您查询这个订单。退票手续费为票价的5%，预计3-7个工作日退回原支付账户。', NOW(), NOW(), NOW()),
-- 对话3: 改签咨询
('CONV003', 1, 'user_003', '我的票可以改签吗？', NOW(), NOW(), NOW()),
('CONV003', 3, 'CS002', '可以的，请告诉我您想改签到什么时间？', NOW(), NOW(), NOW()),
('CONV003', 1, 'user_003', '想改到后天下午的车', NOW(), NOW(), NOW()),
('CONV003', 3, 'CS002', '好的，后天下午有14:30和16:00两趟车，请问您选择哪一趟？', NOW(), NOW(), NOW()),
-- 对话4: VIP咨询
('CONV004', 1, 'U_123', '你好，我是VIP会员，想咨询一下积分兑换', NOW(), NOW(), NOW()),
('CONV004', 3, 'CS003', '您好刘先生，感谢您的支持！您目前有12000积分，可以兑换价值120元的代金券。', NOW(), NOW(), NOW()),
-- 对话5: AI对话
('CONV005', 1, 'user_004', '晚上好，有优惠活动吗？', NOW(), NOW(), NOW()),
('CONV005', 2, 'CS9999', '晚上好！目前我们有春节返乡优惠活动，部分线路享8折优惠，需要我为您推荐吗？', NOW(), NOW(), NOW());

-- 重新启用外键检查
SET FOREIGN_KEY_CHECKS = 1;

SELECT '数据清理并重新初始化完成！' as result;

-- 显示各表数据量统计
SELECT 'sys_roles' as table_name, COUNT(*) as count FROM sys_roles
UNION ALL
SELECT 't_conv_category', COUNT(*) FROM t_conv_category
UNION ALL
SELECT 't_conv_tag', COUNT(*) FROM t_conv_tag
UNION ALL
SELECT 't_msg_category', COUNT(*) FROM t_msg_category
UNION ALL
SELECT 't_quick_reply', COUNT(*) FROM t_quick_reply
UNION ALL
SELECT 't_shift_config', COUNT(*) FROM t_shift_config
UNION ALL
SELECT 't_schedule', COUNT(*) FROM t_schedule
UNION ALL
SELECT 't_conversation', COUNT(*) FROM t_conversation
UNION ALL
SELECT 't_conv_message', COUNT(*) FROM t_conv_message
UNION ALL
SELECT 't_customer_service', COUNT(*) FROM t_customer_service
UNION ALL
SELECT 'sys_users', COUNT(*) FROM sys_users;
