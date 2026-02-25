SET NAMES utf8mb4;
USE ccc;

-- 删除旧的班次配置并重新插入
DELETE FROM t_shift_config;

INSERT INTO t_shift_config (shift_id, shift_name, start_time, end_time, min_staff, is_holiday, create_by, create_time, update_time) VALUES
(1, '早班', '2026-01-01 08:00:00', '2026-01-01 12:00:00', 2, 0, 'admin', NOW(), NOW()),
(2, '午班', '2026-01-01 12:00:00', '2026-01-01 18:00:00', 3, 0, 'admin', NOW(), NOW()),
(3, '晚班', '2026-01-01 18:00:00', '2026-01-01 22:00:00', 2, 0, 'admin', NOW(), NOW()),
(4, '夜班', '2026-01-01 22:00:00', '2026-01-02 06:00:00', 1, 0, 'admin', NOW(), NOW()),
(5, '周末早班', '2026-01-01 09:00:00', '2026-01-01 15:00:00', 2, 1, 'admin', NOW(), NOW()),
(6, '周末晚班', '2026-01-01 15:00:00', '2026-01-01 21:00:00', 2, 1, 'admin', NOW(), NOW());

SELECT shift_id, shift_name FROM t_shift_config;
