-- ============================================================
-- 第四层：排班与请假数据生成脚本
-- 生成100万排班 + 10万请假 + 15万审批日志
-- ============================================================

SET autocommit = 0;
SET unique_checks = 0;
SET foreign_key_checks = 0;

-- ============================================================
-- 生成排班数据
-- ============================================================
DROP PROCEDURE IF EXISTS generate_schedules;
DELIMITER $$
CREATE PROCEDURE generate_schedules(IN total_count INT, IN batch_size INT)
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE batch_start INT DEFAULT 0;
    DECLARE v_cs_id VARCHAR(32);
    DECLARE v_shift_id BIGINT;
    DECLARE v_schedule_date DATE;
    DECLARE v_status TINYINT;
    DECLARE progress_msg VARCHAR(100);
    
    DELETE FROM t_schedule WHERE 1=1;
    
    WHILE i < total_count DO
        SET batch_start = i;
        
        START TRANSACTION;
        
        WHILE i < total_count AND i < batch_start + batch_size DO
            -- 随机客服ID
            SET v_cs_id = CONCAT('CS', FLOOR(100 + RAND() * 900));
            
            -- 随机班次ID (1-10)
            SET v_shift_id = FLOOR(1 + RAND() * 10);
            
            -- 随机日期（过去一年到未来一年）
            SET v_schedule_date = DATE_ADD(CURDATE(), INTERVAL FLOOR(-365 + RAND() * 730) DAY);
            
            -- 状态: 0待确认 1已确认 2已完成 3请假 4调班
            SET v_status = FLOOR(RAND() * 5);
            
            INSERT INTO t_schedule (
                cs_id, shift_id, schedule_date, status,
                replace_cs_id, create_time, update_time
            ) VALUES (
                v_cs_id,
                v_shift_id,
                v_schedule_date,
                v_status,
                IF(v_status = 4, CONCAT('CS', FLOOR(100 + RAND() * 900)), NULL),
                DATE_SUB(v_schedule_date, INTERVAL FLOOR(1 + RAND() * 30) DAY),
                NOW()
            );
            
            SET i = i + 1;
        END WHILE;
        
        COMMIT;
        
        IF i % 100000 = 0 THEN
            SET progress_msg = CONCAT('✓ 排班: ', i, ' / ', total_count, ' (', ROUND(i * 100 / total_count, 1), '%)');
            SELECT progress_msg as progress;
        END IF;
    END WHILE;
    
    SELECT CONCAT('✓ 排班生成完成: ', (SELECT COUNT(*) FROM t_schedule), ' 条') as result;
END$$
DELIMITER ;

-- ============================================================
-- 生成请假/调班申请数据
-- ============================================================
DROP PROCEDURE IF EXISTS generate_leave_transfers;
DELIMITER $$
CREATE PROCEDURE generate_leave_transfers(IN total_count INT, IN batch_size INT)
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE batch_start INT DEFAULT 0;
    DECLARE v_cs_id VARCHAR(32);
    DECLARE v_apply_type TINYINT;
    DECLARE v_leave_type TINYINT;
    DECLARE v_target_date DATE;
    DECLARE v_approval_status TINYINT;
    DECLARE v_reason VARCHAR(256);
    DECLARE progress_msg VARCHAR(100);
    
    SET @reasons = '家中有事需要请假,身体不适需要休息,参加培训学习,个人事务处理,陪同就医,临时有急事';
    
    DELETE FROM t_leave_transfer WHERE 1=1;
    
    WHILE i < total_count DO
        SET batch_start = i;
        
        START TRANSACTION;
        
        WHILE i < total_count AND i < batch_start + batch_size DO
            SET v_cs_id = CONCAT('CS', FLOOR(100 + RAND() * 900));
            
            -- 申请类型: 1请假 2调班
            SET v_apply_type = IF(RAND() < 0.7, 1, 2);
            
            -- 请假类型: 1事假 2病假 3年假 4调休
            SET v_leave_type = FLOOR(1 + RAND() * 4);
            
            SET v_target_date = DATE_ADD(CURDATE(), INTERVAL FLOOR(-180 + RAND() * 360) DAY);
            
            -- 审批状态: 0待审批 1已批准 2已拒绝 3已取消
            SET v_approval_status = CASE 
                WHEN RAND() < 0.1 THEN 0
                WHEN RAND() < 0.8 THEN 1
                WHEN RAND() < 0.95 THEN 2
                ELSE 3
            END;
            
            SET v_reason = SUBSTRING_INDEX(SUBSTRING_INDEX(@reasons, ',', FLOOR(1 + RAND() * 6)), ',', -1);
            
            INSERT INTO t_leave_transfer (
                cs_id, apply_type, leave_type, target_date,
                start_date, end_date, start_period, end_period,
                shift_id, target_cs_id, approval_status,
                approver_id, approver_name, approval_time, approval_remark,
                reason, attachments, approver_role, create_time, update_time
            ) VALUES (
                v_cs_id,
                v_apply_type,
                IF(v_apply_type = 1, v_leave_type, 0),
                v_target_date,
                v_target_date,
                DATE_ADD(v_target_date, INTERVAL FLOOR(RAND() * 3) DAY),
                FLOOR(RAND() * 2),
                FLOOR(RAND() * 2),
                FLOOR(1 + RAND() * 10),
                IF(v_apply_type = 2, CONCAT('CS', FLOOR(100 + RAND() * 900)), NULL),
                v_approval_status,
                IF(v_approval_status > 0, CONCAT('admin_', FLOOR(1 + RAND() * 10)), NULL),
                IF(v_approval_status > 0, CONCAT('管理员', FLOOR(1 + RAND() * 10)), NULL),
                IF(v_approval_status > 0, DATE_ADD(v_target_date, INTERVAL FLOOR(-5 + RAND() * 3) DAY), NULL),
                IF(v_approval_status = 2, '与工作安排冲突，请另择日期', NULL),
                v_reason,
                NULL,
                IF(v_approval_status > 0, 'admin', NULL),
                DATE_SUB(v_target_date, INTERVAL FLOOR(5 + RAND() * 10) DAY),
                NOW()
            );
            
            SET i = i + 1;
        END WHILE;
        
        COMMIT;
        
        IF i % 50000 = 0 THEN
            SET progress_msg = CONCAT('✓ 请假/调班: ', i, ' / ', total_count, ' (', ROUND(i * 100 / total_count, 1), '%)');
            SELECT progress_msg as progress;
        END IF;
    END WHILE;
    
    SELECT CONCAT('✓ 请假/调班生成完成: ', (SELECT COUNT(*) FROM t_leave_transfer), ' 条') as result;
END$$
DELIMITER ;

-- ============================================================
-- 生成审批日志数据
-- ============================================================
DROP PROCEDURE IF EXISTS generate_leave_audit_logs;
DELIMITER $$
CREATE PROCEDURE generate_leave_audit_logs(IN total_count INT, IN batch_size INT)
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE batch_start INT DEFAULT 0;
    DECLARE v_apply_id BIGINT;
    DECLARE v_action VARCHAR(32);
    DECLARE v_operator_id VARCHAR(32);
    DECLARE v_operator_name VARCHAR(64);
    DECLARE progress_msg VARCHAR(100);
    
    SET @actions = '提交申请,审批通过,审批拒绝,撤回申请,修改申请,催办提醒';
    SET @operators = '张经理,李主管,王组长,赵审批员,刘管理员';
    
    DELETE FROM t_leave_audit_log WHERE 1=1;
    
    WHILE i < total_count DO
        SET batch_start = i;
        
        START TRANSACTION;
        
        WHILE i < total_count AND i < batch_start + batch_size DO
            SET v_apply_id = FLOOR(1 + RAND() * 100000);
            SET v_action = SUBSTRING_INDEX(SUBSTRING_INDEX(@actions, ',', FLOOR(1 + RAND() * 6)), ',', -1);
            SET v_operator_name = SUBSTRING_INDEX(SUBSTRING_INDEX(@operators, ',', FLOOR(1 + RAND() * 5)), ',', -1);
            SET v_operator_id = CONCAT('OP', FLOOR(1000 + RAND() * 9000));
            
            INSERT INTO t_leave_audit_log (
                apply_id, action, operator_id, operator_name,
                operator_role, remark, create_time
            ) VALUES (
                v_apply_id,
                v_action,
                v_operator_id,
                v_operator_name,
                IF(RAND() < 0.5, 'admin', 'approver'),
                IF(RAND() < 0.3, '无备注', NULL),
                DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 365) DAY)
            );
            
            SET i = i + 1;
        END WHILE;
        
        COMMIT;
        
        IF i % 50000 = 0 THEN
            SET progress_msg = CONCAT('✓ 审批日志: ', i, ' / ', total_count, ' (', ROUND(i * 100 / total_count, 1), '%)');
            SELECT progress_msg as progress;
        END IF;
    END WHILE;
    
    SELECT CONCAT('✓ 审批日志生成完成: ', (SELECT COUNT(*) FROM t_leave_audit_log), ' 条') as result;
END$$
DELIMITER ;

-- 执行生成
CALL generate_schedules(1000000, 10000);
CALL generate_leave_transfers(100000, 5000);
CALL generate_leave_audit_logs(150000, 5000);

SET autocommit = 1;
SET unique_checks = 1;
SET foreign_key_checks = 1;

SELECT '========== 第四层排班与请假数据生成完成 ==========' as final_status;
