-- ============================================================
-- 第三层：会话与消息数据生成脚本
-- 生成100万会话 + 100万消息 + 10万转接记录
-- ============================================================

SET autocommit = 0;
SET unique_checks = 0;
SET foreign_key_checks = 0;

-- ============================================================
-- 用户消息模板
-- ============================================================
DROP TEMPORARY TABLE IF EXISTS tmp_user_messages;
CREATE TEMPORARY TABLE tmp_user_messages (id INT PRIMARY KEY, content TEXT);
INSERT INTO tmp_user_messages VALUES
(1, '您好，我想咨询一下购票的问题'),
(2, '请问怎么退票？'),
(3, '我的订单什么时候能出票？'),
(4, '为什么我的支付失败了？'),
(5, '我想改签到明天的票'),
(6, '请问有学生票优惠吗？'),
(7, '我的票丢了怎么办？'),
(8, '请问可以开发票吗？'),
(9, '为什么我收不到短信？'),
(10, '我想投诉你们的服务'),
(11, '请帮我查一下我的订单'),
(12, '票价怎么这么贵？'),
(13, '有没有折扣活动？'),
(14, '我的积分怎么用？'),
(15, '请问在哪里取票？'),
(16, '能不能帮我选个靠窗的位置？'),
(17, '儿童需要买票吗？'),
(18, '带宠物可以上车吗？'),
(19, '行李超重了怎么办？'),
(20, '我想办理会员'),
(21, '密码忘记了怎么找回？'),
(22, '这个APP怎么这么难用'),
(23, '订单取消后多久退款？'),
(24, '我想预约明天的票'),
(25, '你们的客服电话是多少？'),
(26, '我想转人工服务'),
(27, '能帮我催一下出票吗？'),
(28, '为什么我的账号被冻结了？'),
(29, '团体购票有优惠吗？'),
(30, '请问最早一班是几点？');

-- 客服回复模板
DROP TEMPORARY TABLE IF EXISTS tmp_cs_messages;
CREATE TEMPORARY TABLE tmp_cs_messages (id INT PRIMARY KEY, content TEXT);
INSERT INTO tmp_cs_messages VALUES
(1, '您好，很高兴为您服务，请问有什么可以帮您？'),
(2, '好的，我来帮您查询一下。'),
(3, '已经为您处理好了，请刷新查看。'),
(4, '抱歉给您带来不便，我们会尽快处理。'),
(5, '您的问题我已记录，会反馈给相关部门。'),
(6, '请问您的订单号是多少？'),
(7, '感谢您的耐心等待。'),
(8, '这个问题需要进一步核实，请稍候。'),
(9, '已为您提交申请，预计1-3个工作日处理完成。'),
(10, '您可以在APP的"我的订单"中查看详情。'),
(11, '退款预计3-5个工作日到账，请注意查收。'),
(12, '已为您发送短信验证码，请注意查收。'),
(13, '您的问题我已经解决了，还有其他需要帮助的吗？'),
(14, '建议您使用最新版本的APP操作。'),
(15, '非常感谢您的反馈，我们会持续改进。'),
(16, '请问方便留下您的联系方式吗？'),
(17, '您的满意是我们最大的追求！'),
(18, '这边已经为您加急处理了。'),
(19, '请您不要担心，我们一定会妥善处理。'),
(20, '感谢您的咨询，祝您出行愉快！');

SELECT '✓ 消息模板已创建' as progress;

-- ============================================================
-- 生成会话数据存储过程
-- ============================================================
DROP PROCEDURE IF EXISTS generate_conversations;
DELIMITER $$
CREATE PROCEDURE generate_conversations(IN total_count INT, IN batch_size INT)
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE batch_start INT DEFAULT 0;
    DECLARE v_user_id VARCHAR(64);
    DECLARE v_cs_id VARCHAR(32);
    DECLARE v_conv_id VARCHAR(64);
    DECLARE v_start_time DATETIME;
    DECLARE v_status TINYINT;
    DECLARE v_source VARCHAR(32);
    DECLARE progress_msg VARCHAR(100);
    
    -- 来源列表
    SET @sources = 'APP,微信小程序,网页,H5,客服电话';
    
    -- 清空现有会话数据
    DELETE FROM t_conversation WHERE 1=1;
    
    WHILE i < total_count DO
        SET batch_start = i;
        
        START TRANSACTION;
        
        WHILE i < total_count AND i < batch_start + batch_size DO
            -- 生成会话ID
            SET v_conv_id = CONCAT('CONV', LPAD(i, 10, '0'));
            
            -- 随机用户ID
            SET v_user_id = CONCAT('user_', FLOOR(RAND() * 1000000));
            
            -- 随机客服ID（70% AI客服，30% 人工客服）
            IF RAND() < 0.7 THEN
                SET v_cs_id = 'CS9999';
            ELSE
                SET v_cs_id = CONCAT('CS', FLOOR(100 + RAND() * 900));
            END IF;
            
            -- 随机开始时间（过去一年内）
            SET v_start_time = DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 365) DAY);
            SET v_start_time = DATE_ADD(v_start_time, INTERVAL FLOOR(RAND() * 24) HOUR);
            SET v_start_time = DATE_ADD(v_start_time, INTERVAL FLOOR(RAND() * 60) MINUTE);
            
            -- 随机状态 0:进行中 1:已结束 2:待评价
            SET v_status = CASE 
                WHEN RAND() < 0.1 THEN 0
                WHEN RAND() < 0.85 THEN 1
                ELSE 2
            END;
            
            -- 随机来源
            SET v_source = SUBSTRING_INDEX(SUBSTRING_INDEX(@sources, ',', FLOOR(1 + RAND() * 5)), ',', -1);
            
            INSERT INTO t_conversation (
                conv_id, user_id, user_nickname, cs_id, source,
                start_time, end_time, last_msg_time, msg_type,
                is_manual_adjust, category_id, tags, is_core, status,
                version, create_time, update_time
            ) VALUES (
                v_conv_id,
                v_user_id,
                NULL,
                v_cs_id,
                v_source,
                v_start_time,
                IF(v_status > 0, DATE_ADD(v_start_time, INTERVAL FLOOR(5 + RAND() * 55) MINUTE), NULL),
                DATE_ADD(v_start_time, INTERVAL FLOOR(1 + RAND() * 60) MINUTE),
                FLOOR(RAND() * 3),
                IF(RAND() < 0.1, 1, 0),
                FLOOR(1 + RAND() * 20),
                NULL,
                IF(RAND() < 0.05, 1, 0),
                v_status,
                0,
                v_start_time,
                NOW()
            );
            
            SET i = i + 1;
        END WHILE;
        
        COMMIT;
        
        IF i % 100000 = 0 THEN
            SET progress_msg = CONCAT('✓ 会话: ', i, ' / ', total_count, ' (', ROUND(i * 100 / total_count, 1), '%)');
            SELECT progress_msg as progress;
        END IF;
    END WHILE;
    
    SELECT CONCAT('✓ 会话生成完成: ', (SELECT COUNT(*) FROM t_conversation), ' 条') as result;
END$$
DELIMITER ;

-- ============================================================
-- 生成消息数据存储过程
-- ============================================================
DROP PROCEDURE IF EXISTS generate_messages;
DELIMITER $$
CREATE PROCEDURE generate_messages(IN total_count INT, IN batch_size INT)
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE batch_start INT DEFAULT 0;
    DECLARE v_conv_id VARCHAR(64);
    DECLARE v_sender_type TINYINT;
    DECLARE v_sender_id VARCHAR(64);
    DECLARE v_content TEXT;
    DECLARE v_send_time DATETIME;
    DECLARE progress_msg VARCHAR(100);
    
    -- 清空现有消息数据
    DELETE FROM t_conv_message WHERE 1=1;
    
    WHILE i < total_count DO
        SET batch_start = i;
        
        START TRANSACTION;
        
        WHILE i < total_count AND i < batch_start + batch_size DO
            -- 随机会话ID
            SET v_conv_id = CONCAT('CONV', LPAD(FLOOR(RAND() * 1000000), 10, '0'));
            
            -- 随机发送者类型 1:用户 2:客服
            SET v_sender_type = IF(RAND() < 0.5, 1, 2);
            
            IF v_sender_type = 1 THEN
                SET v_sender_id = CONCAT('user_', FLOOR(RAND() * 1000000));
                SELECT content INTO v_content FROM tmp_user_messages WHERE id = FLOOR(1 + RAND() * 30) LIMIT 1;
            ELSE
                SET v_sender_id = IF(RAND() < 0.7, 'CS9999', CONCAT('CS', FLOOR(100 + RAND() * 900)));
                SELECT content INTO v_content FROM tmp_cs_messages WHERE id = FLOOR(1 + RAND() * 20) LIMIT 1;
            END IF;
            
            SET v_send_time = DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 365) DAY);
            SET v_send_time = DATE_ADD(v_send_time, INTERVAL FLOOR(RAND() * 24) HOUR);
            
            INSERT INTO t_conv_message (
                conv_id, sender_type, sender_id, msg_content,
                is_quick_reply, quick_reply_id, send_time, create_time, update_time
            ) VALUES (
                v_conv_id,
                v_sender_type,
                v_sender_id,
                v_content,
                IF(v_sender_type = 2 AND RAND() < 0.3, 1, 0),
                IF(RAND() < 0.2, FLOOR(1 + RAND() * 500), NULL),
                v_send_time,
                v_send_time,
                NOW()
            );
            
            SET i = i + 1;
        END WHILE;
        
        COMMIT;
        
        IF i % 100000 = 0 THEN
            SET progress_msg = CONCAT('✓ 消息: ', i, ' / ', total_count, ' (', ROUND(i * 100 / total_count, 1), '%)');
            SELECT progress_msg as progress;
        END IF;
    END WHILE;
    
    SELECT CONCAT('✓ 消息生成完成: ', (SELECT COUNT(*) FROM t_conv_message), ' 条') as result;
END$$
DELIMITER ;

-- ============================================================
-- 生成转接记录存储过程
-- ============================================================
DROP PROCEDURE IF EXISTS generate_transfers;
DELIMITER $$
CREATE PROCEDURE generate_transfers(IN total_count INT, IN batch_size INT)
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE batch_start INT DEFAULT 0;
    DECLARE v_conv_id VARCHAR(64);
    DECLARE v_from_cs_id VARCHAR(32);
    DECLARE v_to_cs_id VARCHAR(32);
    DECLARE v_transfer_time DATETIME;
    DECLARE v_reason VARCHAR(256);
    DECLARE progress_msg VARCHAR(100);
    
    SET @reasons = '客户要求转人工,问题超出处理范围,需要专业客服处理,AI无法解决,VIP客户专属服务,复杂投诉处理';
    
    DELETE FROM t_conv_transfer WHERE 1=1;
    
    WHILE i < total_count DO
        SET batch_start = i;
        
        START TRANSACTION;
        
        WHILE i < total_count AND i < batch_start + batch_size DO
            SET v_conv_id = CONCAT('CONV', LPAD(FLOOR(RAND() * 1000000), 10, '0'));
            SET v_from_cs_id = IF(RAND() < 0.8, 'CS9999', CONCAT('CS', FLOOR(100 + RAND() * 900)));
            SET v_to_cs_id = CONCAT('CS', FLOOR(100 + RAND() * 900));
            SET v_transfer_time = DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 365) DAY);
            SET v_reason = SUBSTRING_INDEX(SUBSTRING_INDEX(@reasons, ',', FLOOR(1 + RAND() * 6)), ',', -1);
            
            INSERT INTO t_conv_transfer (
                conv_id, from_cs_id, from_cs_name, to_cs_id, to_cs_name,
                transfer_reason, context_remark, transfer_time, accept_time, status
            ) VALUES (
                v_conv_id,
                v_from_cs_id,
                IF(v_from_cs_id = 'CS9999', '智能助理', CONCAT('客服', FLOOR(RAND() * 1000))),
                v_to_cs_id,
                CONCAT('客服', FLOOR(RAND() * 1000)),
                v_reason,
                '用户咨询内容摘要...',
                v_transfer_time,
                IF(RAND() < 0.9, DATE_ADD(v_transfer_time, INTERVAL FLOOR(1 + RAND() * 10) MINUTE), NULL),
                IF(RAND() < 0.9, 1, 0)
            );
            
            SET i = i + 1;
        END WHILE;
        
        COMMIT;
        
        IF i % 50000 = 0 THEN
            SET progress_msg = CONCAT('✓ 转接: ', i, ' / ', total_count, ' (', ROUND(i * 100 / total_count, 1), '%)');
            SELECT progress_msg as progress;
        END IF;
    END WHILE;
    
    SELECT CONCAT('✓ 转接记录生成完成: ', (SELECT COUNT(*) FROM t_conv_transfer), ' 条') as result;
END$$
DELIMITER ;

-- 执行生成
CALL generate_conversations(1000000, 10000);
CALL generate_messages(1000000, 10000);
CALL generate_transfers(100000, 5000);

SET autocommit = 1;
SET unique_checks = 1;
SET foreign_key_checks = 1;

SELECT '========== 第三层会话与消息数据生成完成 ==========' as final_status;
