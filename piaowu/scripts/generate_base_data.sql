-- ============================================================
-- 百万条测试数据生成脚本
-- 为票务客服系统生成符合业务逻辑的测试数据
-- ============================================================

-- 配置优化：提升批量插入性能
SET autocommit = 0;
SET unique_checks = 0;
SET foreign_key_checks = 0;
SET sql_log_bin = 0;

-- ============================================================
-- 第一层：基础配置数据
-- ============================================================

-- 1.1 角色数据（5条）
INSERT IGNORE INTO sys_roles (role_code, role_name, remark) VALUES
('admin', '管理员', '拥有系统所有权限'),
('customer_service', '客服', '客服人员，负责接待用户咨询'),
('user', '普通用户', '系统用户，可以发起咨询'),
('approver', '审批员', '负责审批请假和调班申请'),
('super_admin', '超级管理员', '最高权限管理员');

SELECT '✓ 角色数据已生成' as progress;

-- 1.2 班次配置（10条）
TRUNCATE TABLE t_shift_config;
INSERT INTO t_shift_config (shift_name, start_time, end_time, min_staff, is_holiday, create_time, update_time, create_by) VALUES
('早班A', '2026-01-01 08:00:00', '2026-01-01 12:00:00', 5, 0, NOW(), NOW(), 'admin'),
('早班B', '2026-01-01 08:30:00', '2026-01-01 12:30:00', 4, 0, NOW(), NOW(), 'admin'),
('午班A', '2026-01-01 12:00:00', '2026-01-01 18:00:00', 6, 0, NOW(), NOW(), 'admin'),
('午班B', '2026-01-01 12:30:00', '2026-01-01 18:30:00', 5, 0, NOW(), NOW(), 'admin'),
('晚班A', '2026-01-01 18:00:00', '2026-01-01 22:00:00', 4, 0, NOW(), NOW(), 'admin'),
('晚班B', '2026-01-01 18:30:00', '2026-01-01 23:00:00', 3, 0, NOW(), NOW(), 'admin'),
('夜班', '2026-01-01 22:00:00', '2026-01-02 06:00:00', 2, 0, NOW(), NOW(), 'admin'),
('周末早班', '2026-01-01 09:00:00', '2026-01-01 15:00:00', 3, 1, NOW(), NOW(), 'admin'),
('周末晚班', '2026-01-01 15:00:00', '2026-01-01 21:00:00', 3, 1, NOW(), NOW(), 'admin'),
('节假日班', '2026-01-01 10:00:00', '2026-01-01 18:00:00', 2, 1, NOW(), NOW(), 'admin');

SELECT '✓ 班次配置已生成' as progress;

-- 1.3 消息分类（20条）
TRUNCATE TABLE t_msg_category;
INSERT INTO t_msg_category (category_name, keywords, sort_no, create_by, create_time, update_time) VALUES
('购票咨询', '买票,购票,订票,怎么买,票价', 1, 'admin', NOW(), NOW()),
('退票服务', '退票,退款,取消订单,不想要了', 2, 'admin', NOW(), NOW()),
('改签服务', '改签,换票,更改日期,改时间', 3, 'admin', NOW(), NOW()),
('行程查询', '查询,订单,行程,我的票', 4, 'admin', NOW(), NOW()),
('座位选择', '选座,换座,靠窗,过道', 5, 'admin', NOW(), NOW()),
('支付问题', '支付,付款,扣款,支付失败', 6, 'admin', NOW(), NOW()),
('发票服务', '发票,报销,开票,电子发票', 7, 'admin', NOW(), NOW()),
('会员服务', '会员,积分,等级,权益', 8, 'admin', NOW(), NOW()),
('优惠活动', '优惠,折扣,满减,促销', 9, 'admin', NOW(), NOW()),
('账号问题', '登录,注册,密码,账号', 10, 'admin', NOW(), NOW()),
('投诉建议', '投诉,建议,不满意,差评', 11, 'admin', NOW(), NOW()),
('技术故障', '故障,bug,闪退,打不开', 12, 'admin', NOW(), NOW()),
('行李托运', '行李,托运,超重,尺寸', 13, 'admin', NOW(), NOW()),
('特殊服务', '轮椅,婴儿,老人,特殊', 14, 'admin', NOW(), NOW()),
('保险咨询', '保险,理赔,延误险,取消险', 15, 'admin', NOW(), NOW()),
('接送服务', '接机,送机,接站,送站', 16, 'admin', NOW(), NOW()),
('证件问题', '身份证,护照,证件,实名', 17, 'admin', NOW(), NOW()),
('儿童票', '儿童,小孩,未成年,学生', 18, 'admin', NOW(), NOW()),
('团体票', '团购,团体,公司,单位', 19, 'admin', NOW(), NOW()),
('其他问题', '其他,咨询,问题,帮助', 20, 'admin', NOW(), NOW());

SELECT '✓ 消息分类已生成' as progress;

-- 1.4 会话分类（20条）
TRUNCATE TABLE t_conv_category;
INSERT INTO t_conv_category (category_name, sort_no, create_by, create_time, update_time) VALUES
('售前咨询', 1, 'admin', NOW(), NOW()),
('售后服务', 2, 'admin', NOW(), NOW()),
('退换货处理', 3, 'admin', NOW(), NOW()),
('投诉处理', 4, 'admin', NOW(), NOW()),
('技术支持', 5, 'admin', NOW(), NOW()),
('账户问题', 6, 'admin', NOW(), NOW()),
('支付问题', 7, 'admin', NOW(), NOW()),
('物流查询', 8, 'admin', NOW(), NOW()),
('产品咨询', 9, 'admin', NOW(), NOW()),
('优惠活动', 10, 'admin', NOW(), NOW()),
('会员服务', 11, 'admin', NOW(), NOW()),
('建议反馈', 12, 'admin', NOW(), NOW()),
('紧急事务', 13, 'admin', NOW(), NOW()),
('预约服务', 14, 'admin', NOW(), NOW()),
('合作咨询', 15, 'admin', NOW(), NOW()),
('媒体咨询', 16, 'admin', NOW(), NOW()),
('法务咨询', 17, 'admin', NOW(), NOW()),
('人工转接', 18, 'admin', NOW(), NOW()),
('AI无法解决', 19, 'admin', NOW(), NOW()),
('其他', 20, 'admin', NOW(), NOW());

SELECT '✓ 会话分类已生成' as progress;

-- 1.5 会话标签（50条）
TRUNCATE TABLE t_conv_tag;
INSERT INTO t_conv_tag (tag_name, tag_color, sort_no, create_by, create_time, update_time) VALUES
('VIP客户', '#FF4D4F', 1, 'admin', NOW(), NOW()),
('高优先级', '#FA541C', 2, 'admin', NOW(), NOW()),
('中优先级', '#FA8C16', 3, 'admin', NOW(), NOW()),
('低优先级', '#52C41A', 4, 'admin', NOW(), NOW()),
('投诉中', '#F5222D', 5, 'admin', NOW(), NOW()),
('已解决', '#52C41A', 6, 'admin', NOW(), NOW()),
('待跟进', '#1890FF', 7, 'admin', NOW(), NOW()),
('需回访', '#722ED1', 8, 'admin', NOW(), NOW()),
('紧急处理', '#EB2F96', 9, 'admin', NOW(), NOW()),
('转接人工', '#13C2C2', 10, 'admin', NOW(), NOW()),
('AI处理', '#2F54EB', 11, 'admin', NOW(), NOW()),
('复杂问题', '#FAAD14', 12, 'admin', NOW(), NOW()),
('简单咨询', '#A0D911', 13, 'admin', NOW(), NOW()),
('重复来电', '#FFD666', 14, 'admin', NOW(), NOW()),
('首次咨询', '#87E8DE', 15, 'admin', NOW(), NOW()),
('老客户', '#B7EB8F', 16, 'admin', NOW(), NOW()),
('新客户', '#91D5FF', 17, 'admin', NOW(), NOW()),
('企业客户', '#D3ADF7', 18, 'admin', NOW(), NOW()),
('个人客户', '#FFADD2', 19, 'admin', NOW(), NOW()),
('需升级', '#FFA39E', 20, 'admin', NOW(), NOW()),
('退款申请', '#FF7A45', 21, 'admin', NOW(), NOW()),
('改签申请', '#FFC53D', 22, 'admin', NOW(), NOW()),
('出票成功', '#73D13D', 23, 'admin', NOW(), NOW()),
('出票失败', '#FF4D4F', 24, 'admin', NOW(), NOW()),
('等待支付', '#40A9FF', 25, 'admin', NOW(), NOW()),
('支付成功', '#52C41A', 26, 'admin', NOW(), NOW()),
('支付失败', '#F5222D', 27, 'admin', NOW(), NOW()),
('已退款', '#722ED1', 28, 'admin', NOW(), NOW()),
('部分退款', '#EB2F96', 29, 'admin', NOW(), NOW()),
('团购订单', '#13C2C2', 30, 'admin', NOW(), NOW()),
('散客订单', '#2F54EB', 31, 'admin', NOW(), NOW()),
('节假日出行', '#FAAD14', 32, 'admin', NOW(), NOW()),
('商务出行', '#A0D911', 33, 'admin', NOW(), NOW()),
('休闲旅游', '#87E8DE', 34, 'admin', NOW(), NOW()),
('学生票', '#B7EB8F', 35, 'admin', NOW(), NOW()),
('儿童票', '#91D5FF', 36, 'admin', NOW(), NOW()),
('老人票', '#D3ADF7', 37, 'admin', NOW(), NOW()),
('残疾人票', '#FFADD2', 38, 'admin', NOW(), NOW()),
('军人票', '#FFA39E', 39, 'admin', NOW(), NOW()),
('国内航班', '#FF7A45', 40, 'admin', NOW(), NOW()),
('国际航班', '#FFC53D', 41, 'admin', NOW(), NOW()),
('高铁动车', '#73D13D', 42, 'admin', NOW(), NOW()),
('普通火车', '#40A9FF', 43, 'admin', NOW(), NOW()),
('汽车客运', '#9254DE', 44, 'admin', NOW(), NOW()),
('轮渡船票', '#F759AB', 45, 'admin', NOW(), NOW()),
('景区门票', '#36CFC9', 46, 'admin', NOW(), NOW()),
('演出票务', '#597EF7', 47, 'admin', NOW(), NOW()),
('体育赛事', '#F9A825', 48, 'admin', NOW(), NOW()),
('会议活动', '#8BC34A', 49, 'admin', NOW(), NOW()),
('其他类型', '#78909C', 50, 'admin', NOW(), NOW());

SELECT '✓ 会话标签已生成' as progress;

-- 1.6 快捷回复（500条）
TRUNCATE TABLE t_quick_reply;
INSERT INTO t_quick_reply (reply_type, reply_content, create_by, is_public, create_time, update_time) VALUES
-- 问候语（1-20）
(1, '您好，欢迎咨询票务服务，请问有什么可以帮您？', 'admin', 1, NOW(), NOW()),
(1, '您好，我是您的专属客服，很高兴为您服务！', 'admin', 1, NOW(), NOW()),
(1, '感谢您的耐心等待，请问有什么需要帮助的吗？', 'admin', 1, NOW(), NOW()),
(1, '您好，请问您需要咨询什么业务呢？', 'admin', 1, NOW(), NOW()),
(1, '欢迎回来，请问这次需要什么帮助？', 'admin', 1, NOW(), NOW()),
-- 确认回复（21-40）
(2, '好的，我已经收到您的问题，正在为您查询。', 'admin', 1, NOW(), NOW()),
(2, '收到，请您稍等，我马上为您处理。', 'admin', 1, NOW(), NOW()),
(2, '明白了，让我来帮您解决这个问题。', 'admin', 1, NOW(), NOW()),
(2, '好的，我已经记录下来了，请稍候。', 'admin', 1, NOW(), NOW()),
(2, '了解，我这就为您查询相关信息。', 'admin', 1, NOW(), NOW()),
-- 购票帮助（41-80）
(3, '请问您需要购买哪个日期的票呢？', 'admin', 1, NOW(), NOW()),
(3, '您可以在APP首页直接搜索目的地进行购票。', 'admin', 1, NOW(), NOW()),
(3, '购票流程：选择日期→选择座位→确认订单→完成支付', 'admin', 1, NOW(), NOW()),
(3, '目前该日期还有余票，建议您尽快下单。', 'admin', 1, NOW(), NOW()),
(3, '抱歉，该日期的票已售完，建议您换个日期。', 'admin', 1, NOW(), NOW()),
-- 退票帮助（81-120）
(4, '退票手续费按照退票时间计算，越早退票费用越低。', 'admin', 1, NOW(), NOW()),
(4, '您可以在"我的订单"中找到该订单，点击"申请退票"。', 'admin', 1, NOW(), NOW()),
(4, '退款将在3-5个工作日内原路返还到您的支付账户。', 'admin', 1, NOW(), NOW()),
(4, '已为您提交退票申请，请保持手机畅通。', 'admin', 1, NOW(), NOW()),
(4, '退票成功后，退款金额已原路退回，请注意查收。', 'admin', 1, NOW(), NOW()),
-- 结束语（121-150）
(5, '感谢您的咨询，祝您出行愉快！', 'admin', 1, NOW(), NOW()),
(5, '如还有其他问题，随时欢迎咨询，再见！', 'admin', 1, NOW(), NOW()),
(5, '很高兴能帮到您，祝您旅途愉快！', 'admin', 1, NOW(), NOW()),
(5, '感谢您的理解与支持，期待下次为您服务！', 'admin', 1, NOW(), NOW()),
(5, '问题已解决，如有需要可随时联系我们，再见！', 'admin', 1, NOW(), NOW());

-- 继续插入更多快捷回复...
INSERT INTO t_quick_reply (reply_type, reply_content, create_by, is_public, create_time, update_time)
SELECT 
    FLOOR(1 + RAND() * 5) as reply_type,
    CASE FLOOR(1 + RAND() * 20)
        WHEN 1 THEN '请问还有其他需要帮助的吗？'
        WHEN 2 THEN '您的问题已经记录，我们会尽快处理。'
        WHEN 3 THEN '感谢您的反馈，我们会持续改进服务。'
        WHEN 4 THEN '已为您转接专业客服，请稍候。'
        WHEN 5 THEN '您的满意是我们最大的追求！'
        WHEN 6 THEN '请问您的订单号是多少？'
        WHEN 7 THEN '请提供您的手机号以便查询。'
        WHEN 8 THEN '系统正在处理中，请稍后刷新查看。'
        WHEN 9 THEN '这个问题我需要进一步确认，请稍等。'
        WHEN 10 THEN '您可以拨打客服热线400-xxx-xxxx获得更快服务。'
        WHEN 11 THEN '建议您使用最新版本的APP以获得最佳体验。'
        WHEN 12 THEN '该功能预计下个版本上线，敬请期待！'
        WHEN 13 THEN '您的问题比较特殊，我为您转接专家处理。'
        WHEN 14 THEN '请问方便提供一下订单截图吗？'
        WHEN 15 THEN '这个情况我需要核实一下，请稍候。'
        WHEN 16 THEN '非常抱歉给您带来不便，我们马上处理。'
        WHEN 17 THEN '您的建议我们已经收到，感谢您的反馈！'
        WHEN 18 THEN '请问您是通过什么渠道购买的票？'
        WHEN 19 THEN '目前系统较忙，可能需要等待一会儿。'
        ELSE '感谢您对我们的支持与信任！'
    END as reply_content,
    'admin' as create_by,
    1 as is_public,
    NOW() as create_time,
    NOW() as update_time
FROM (
    SELECT 1 as n UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5
    UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9 UNION SELECT 10
) t1,
(
    SELECT 1 as n UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5
    UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9 UNION SELECT 10
) t2,
(
    SELECT 1 as n UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5
) t3
LIMIT 475;

COMMIT;
SELECT '✓ 快捷回复已生成' as progress;
SELECT CONCAT('第一层基础配置数据生成完成，共 ', 
    (SELECT COUNT(*) FROM sys_roles), ' 个角色，',
    (SELECT COUNT(*) FROM t_shift_config), ' 个班次，',
    (SELECT COUNT(*) FROM t_msg_category), ' 个消息分类，',
    (SELECT COUNT(*) FROM t_conv_category), ' 个会话分类，',
    (SELECT COUNT(*) FROM t_conv_tag), ' 个标签，',
    (SELECT COUNT(*) FROM t_quick_reply), ' 条快捷回复'
) as summary;
