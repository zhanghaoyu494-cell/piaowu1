-- ============================================================
-- 第二层：用户与客服数据生成脚本（修正版）
-- 生成100万系统用户 + 1000名客服
-- ============================================================

SET autocommit = 0;
SET unique_checks = 0;
SET foreign_key_checks = 0;

-- 删除已有的测试用户（保留基础用户）
DELETE FROM sys_users WHERE id > 100;

SELECT '✓ 准备生成用户数据...' as progress;

-- ============================================================
-- 使用简化的批量插入方式生成用户数据
-- ============================================================

-- 生成10万用户（速度优化版本）
DROP PROCEDURE IF EXISTS generate_users_batch;
DELIMITER $$
CREATE PROCEDURE generate_users_batch()
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE batch INT DEFAULT 0;
    DECLARE v_surname VARCHAR(4);
    DECLARE v_given_name VARCHAR(8);
    DECLARE v_phone_prefix VARCHAR(3);
    DECLARE v_role VARCHAR(32);
    
    -- 姓氏数组（直接在过程中定义）
    SET @surnames = '张,王,李,赵,刘,陈,杨,黄,周,吴,徐,孙,马,胡,朱,郭,何,林,罗,高,郑,梁,谢,宋,唐,许,韩,冯,邓,曹,彭,曾,肖,田,董,袁,潘,于,蒋,蔡,余,杜,叶,程,苏,魏,吕,丁,任,沈';
    SET @given_names = '伟,芳,娜,敏,静,丽,强,磊,洋,艳,勇,军,杰,娟,涛,明,超,英,霞,平,刚,华,玲,红,春,飞,兰,健,云,波,燕,辉,凤,斌,婷,宇,鹏,倩,雪,萍,浩,雯,琳,欣,佳,梅,俊,阳,璐,凯';
    SET @prefixes = '138,139,150,151,152,158,159,186,187,188';
    
    WHILE i < 1000000 DO
        SET batch = 0;
        START TRANSACTION;
        
        WHILE batch < 10000 AND i < 1000000 DO
            -- 随机选择姓氏
            SET v_surname = SUBSTRING_INDEX(SUBSTRING_INDEX(@surnames, ',', 1 + FLOOR(RAND() * 50)), ',', -1);
            -- 随机选择名字
            SET v_given_name = SUBSTRING_INDEX(SUBSTRING_INDEX(@given_names, ',', 1 + FLOOR(RAND() * 50)), ',', -1);
            -- 随机手机号前缀
            SET v_phone_prefix = SUBSTRING_INDEX(SUBSTRING_INDEX(@prefixes, ',', 1 + FLOOR(RAND() * 10)), ',', -1);
            
            -- 角色分布
            SET v_role = CASE 
                WHEN i < 40000 THEN 'customer_service'  -- 4万客服
                ELSE 'user'
            END;
            
            INSERT INTO sys_users (user_name, password, real_name, phone, role_code, status, created_at, updated_at)
            VALUES (
                CONCAT('user_', i),
                '$2a$10$89y8.Zg5l6plG95BSpa6t.z1UcxH0H1OKvZs2kUZpzaeb/w3sybSy',
                CONCAT(v_surname, v_given_name),
                CONCAT(v_phone_prefix, LPAD(FLOOR(RAND() * 100000000), 8, '0')),
                v_role,
                1,
                DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 730) DAY),
                NOW()
            );
            
            SET i = i + 1;
            SET batch = batch + 1;
        END WHILE;
        
        COMMIT;
        
        -- 每10万条输出进度
        IF i % 100000 = 0 THEN
            SELECT CONCAT('✓ 用户: ', i, ' / 1000000 (', ROUND(i * 100.0 / 1000000, 1), '%)') as progress;
        END IF;
    END WHILE;
    
    SELECT CONCAT('✓ 用户生成完成，共 ', (SELECT COUNT(*) FROM sys_users), ' 条') as result;
END$$
DELIMITER ;

-- 执行用户生成
CALL generate_users_batch();

-- ============================================================
-- 生成客服数据
-- ============================================================
SELECT '✓ 开始生成客服数据...' as progress;

-- 清空已有客服数据（保留AI助理）
DELETE FROM t_customer_service WHERE cs_id != 'CS9999';

-- 确保AI助理存在
INSERT IGNORE INTO t_customer_service (cs_id, cs_name, dept_id, status, current_status, is_online, create_time, update_time)
VALUES ('CS9999', '智能助理', 'AI', 1, 0, 1, NOW(), NOW());

-- 从客服用户中创建客服记录
INSERT INTO t_customer_service (
    cs_id, cs_name, dept_id, team_id, skill_tags,
    status, current_status, is_online, last_heartbeat,
    role, password_hash, create_time, update_time
)
SELECT 
    CONCAT('CS', u.id) as cs_id,
    u.real_name as cs_name,
    CASE FLOOR(1 + RAND() * 5)
        WHEN 1 THEN 'DEPT_TICKET'
        WHEN 2 THEN 'DEPT_REFUND'
        WHEN 3 THEN 'DEPT_CONSULT'
        WHEN 4 THEN 'DEPT_COMPLAINT'
        ELSE 'DEPT_VIP'
    END as dept_id,
    CASE FLOOR(1 + RAND() * 3)
        WHEN 1 THEN 'TEAM_A'
        WHEN 2 THEN 'TEAM_B'
        ELSE 'TEAM_C'
    END as team_id,
    CASE FLOOR(1 + RAND() * 6)
        WHEN 1 THEN '购票,改签,退票'
        WHEN 2 THEN '投诉处理,客户安抚'
        WHEN 3 THEN 'VIP服务,高端客户'
        WHEN 4 THEN '技术支持,系统问题'
        WHEN 5 THEN '商务咨询,企业客户'
        ELSE '综合服务,多技能'
    END as skill_tags,
    1 as status,
    FLOOR(RAND() * 3) as current_status,
    IF(RAND() > 0.3, 1, 0) as is_online,
    DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 60) MINUTE) as last_heartbeat,
    0 as role,
    '' as password_hash,
    DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 365) DAY) as create_time,
    NOW() as update_time
FROM sys_users u
WHERE u.role_code = 'customer_service'
LIMIT 1000;

SET autocommit = 1;
SET unique_checks = 1;
SET foreign_key_checks = 1;

SELECT CONCAT('✓ 客服生成完成，共 ', (SELECT COUNT(*) FROM t_customer_service), ' 条') as result;
SELECT '========== 第二层用户与客服数据生成完成 ==========' as final_status;
