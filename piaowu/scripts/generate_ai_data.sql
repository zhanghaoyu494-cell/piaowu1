-- ============================================================
-- 第五层：AI相关数据生成脚本
-- 生成100万AI任务 + 100万会话摘要
-- ============================================================

SET autocommit = 0;
SET unique_checks = 0;
SET foreign_key_checks = 0;

-- ============================================================
-- AI摘要模板
-- ============================================================
DROP TEMPORARY TABLE IF EXISTS tmp_summaries;
CREATE TEMPORARY TABLE tmp_summaries (id INT PRIMARY KEY, content TEXT);
INSERT INTO tmp_summaries VALUES
(1, '用户咨询购票流程，客服详细介绍了APP购票步骤，用户表示满意。'),
(2, '用户反映退票失败问题，经核实为网络原因，已指导用户重新操作成功。'),
(3, '用户询问儿童票政策，客服解答了儿童购票规则和优惠政策。'),
(4, '用户投诉出票慢，已反馈给技术部门处理，承诺24小时内解决。'),
(5, '用户咨询改签服务，告知改签流程和手续费标准，用户已完成改签。'),
(6, '用户查询订单状态，确认订单已出票成功，短信已发送。'),
(7, '用户反馈APP闪退问题，建议更新至最新版本，问题已解决。'),
(8, '用户咨询积分兑换规则，详细介绍积分获取和使用方式。'),
(9, '用户要求开具发票，已指导在APP中申请电子发票。'),
(10, '用户对服务表示满意，给予好评，已记录客户反馈。'),
(11, '用户咨询团体票优惠，介绍了团购折扣和申请流程。'),
(12, '用户报告支付失败，排查为银行限额问题，建议换卡支付。'),
(13, '用户询问行李托运规定，告知免费额度和超重收费标准。'),
(14, '用户要求转人工服务，已转接专业客服处理。'),
(15, '用户咨询会员等级权益，介绍了各等级享受的服务内容。'),
(16, '用户反映重复扣款，已核实并提交退款申请。'),
(17, '用户询问学生票购买条件，说明了学生认证流程。'),
(18, '用户咨询保险理赔流程，指导提交理赔材料。'),
(19, '用户反馈短信未收到，已重新发送并确认收到。'),
(20, '用户表扬客服态度好，已记录表扬并反馈。');

SELECT '✓ 摘要模板已创建' as progress;

-- ============================================================
-- 生成AI任务数据
-- ============================================================
DROP PROCEDURE IF EXISTS generate_ai_jobs;
DELIMITER $$
CREATE PROCEDURE generate_ai_jobs(IN total_count INT, IN batch_size INT)
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE batch_start INT DEFAULT 0;
    DECLARE v_job_id VARCHAR(128);
    DECLARE v_conv_id VARCHAR(64);
    DECLARE v_status VARCHAR(32);
    DECLARE v_result TEXT;
    DECLARE v_created_at DATETIME;
    DECLARE progress_msg VARCHAR(100);
    
    SET @statuses = 'pending,processing,completed,failed';
    SET @results = '处理成功,用户问题已解决,已转接人工,需要进一步跟进,自动回复已发送';
    SET @errors = '服务超时,模型调用失败,参数错误,会话已结束';
    
    DELETE FROM ai_jobs WHERE 1=1;
    
    WHILE i < total_count DO
        SET batch_start = i;
        
        START TRANSACTION;
        
        WHILE i < total_count AND i < batch_start + batch_size DO
            SET v_job_id = CONCAT('JOB_', UUID());
            SET v_conv_id = CONCAT('CONV', LPAD(FLOOR(RAND() * 1000000), 10, '0'));
            SET v_created_at = DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 365) DAY);
            
            -- 状态分布: 5% pending, 5% processing, 85% completed, 5% failed
            SET v_status = CASE 
                WHEN RAND() < 0.05 THEN 'pending'
                WHEN RAND() < 0.10 THEN 'processing'
                WHEN RAND() < 0.95 THEN 'completed'
                ELSE 'failed'
            END;
            
            SET v_result = IF(v_status = 'completed', 
                SUBSTRING_INDEX(SUBSTRING_INDEX(@results, ',', FLOOR(1 + RAND() * 5)), ',', -1),
                NULL);
            
            INSERT INTO ai_jobs (
                job_id, conversation_id, status, result, error, created_at, updated_at
            ) VALUES (
                v_job_id,
                v_conv_id,
                v_status,
                v_result,
                IF(v_status = 'failed', SUBSTRING_INDEX(SUBSTRING_INDEX(@errors, ',', FLOOR(1 + RAND() * 4)), ',', -1), NULL),
                v_created_at,
                IF(v_status IN ('completed', 'failed'), DATE_ADD(v_created_at, INTERVAL FLOOR(1 + RAND() * 60) SECOND), v_created_at)
            );
            
            SET i = i + 1;
        END WHILE;
        
        COMMIT;
        
        IF i % 100000 = 0 THEN
            SET progress_msg = CONCAT('✓ AI任务: ', i, ' / ', total_count, ' (', ROUND(i * 100 / total_count, 1), '%)');
            SELECT progress_msg as progress;
        END IF;
    END WHILE;
    
    SELECT CONCAT('✓ AI任务生成完成: ', (SELECT COUNT(*) FROM ai_jobs), ' 条') as result;
END$$
DELIMITER ;

-- ============================================================
-- 生成会话摘要数据
-- ============================================================
DROP PROCEDURE IF EXISTS generate_conversation_summaries;
DELIMITER $$
CREATE PROCEDURE generate_conversation_summaries(IN total_count INT, IN batch_size INT)
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE batch_start INT DEFAULT 0;
    DECLARE v_conv_id VARCHAR(64);
    DECLARE v_summary TEXT;
    DECLARE v_trace_id VARCHAR(128);
    DECLARE progress_msg VARCHAR(100);
    
    DELETE FROM conversation_summaries WHERE 1=1;
    
    WHILE i < total_count DO
        SET batch_start = i;
        
        START TRANSACTION;
        
        WHILE i < total_count AND i < batch_start + batch_size DO
            SET v_conv_id = CONCAT('CONV', LPAD(FLOOR(RAND() * 1000000), 10, '0'));
            SELECT content INTO v_summary FROM tmp_summaries WHERE id = FLOOR(1 + RAND() * 20) LIMIT 1;
            SET v_trace_id = CONCAT('TRACE_', UUID());
            
            INSERT INTO conversation_summaries (
                conversation_id, summary, trace_id, created_at
            ) VALUES (
                v_conv_id,
                v_summary,
                v_trace_id,
                DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 365) DAY)
            );
            
            SET i = i + 1;
        END WHILE;
        
        COMMIT;
        
        IF i % 100000 = 0 THEN
            SET progress_msg = CONCAT('✓ 会话摘要: ', i, ' / ', total_count, ' (', ROUND(i * 100 / total_count, 1), '%)');
            SELECT progress_msg as progress;
        END IF;
    END WHILE;
    
    SELECT CONCAT('✓ 会话摘要生成完成: ', (SELECT COUNT(*) FROM conversation_summaries), ' 条') as result;
END$$
DELIMITER ;

-- 执行生成
CALL generate_ai_jobs(1000000, 10000);
CALL generate_conversation_summaries(1000000, 10000);

SET autocommit = 1;
SET unique_checks = 1;
SET foreign_key_checks = 1;

SELECT '========== 第五层AI数据生成完成 ==========' as final_status;
