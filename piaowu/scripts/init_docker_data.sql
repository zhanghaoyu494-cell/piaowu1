-- ============================================================
-- Docker 环境数据库初始化脚本
-- 用于初始化用户、角色、客服等基础数据
-- ============================================================

-- 1. 创建系统角色表
CREATE TABLE IF NOT EXISTS sys_roles (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    role_code VARCHAR(32) NOT NULL UNIQUE,
    role_name VARCHAR(32) NOT NULL,
    remark VARCHAR(255),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 插入角色数据
INSERT IGNORE INTO sys_roles (role_code, role_name, remark) VALUES
('admin', '管理员', '拥有所有权限'),
('customer_service', '客服', '会话管理和提交申请权限'),
('user', '普通用户', '用户端权限');

-- 2. 创建系统用户表
CREATE TABLE IF NOT EXISTS sys_users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_name VARCHAR(32) NOT NULL UNIQUE,
    password VARCHAR(128) NOT NULL,
    real_name VARCHAR(32) NOT NULL,
    phone VARCHAR(11),
    role_code VARCHAR(32) NOT NULL,
    status TINYINT(1) DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    INDEX idx_role_code (role_code)
);

-- 插入用户数据
-- 密码使用 bcrypt 加密:
--   admin123 => $2a$10$8J.vXPyob0MwDJPoJ0bnzeCDxHL0I3dzeFzxy7ipru9mjqdyf0f6i
--   123456   => $2a$10$89y8.Zg5l6plG95BSpa6t.z1UcxH0H1OKvZs2kUZpzaeb/w3sybSy

-- 管理员账号
INSERT INTO sys_users (user_name, password, real_name, phone, role_code, status) VALUES
('admin', '$2a$10$8J.vXPyob0MwDJPoJ0bnzeCDxHL0I3dzeFzxy7ipru9mjqdyf0f6i', '管理员', '13800138000', 'admin', 1)
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- 客服账号 (用于测试转接人工)
INSERT INTO sys_users (user_name, password, real_name, phone, role_code, status) VALUES
('cs001', '$2a$10$89y8.Zg5l6plG95BSpa6t.z1UcxH0H1OKvZs2kUZpzaeb/w3sybSy', '客服小张', '13800138001', 'customer_service', 1)
ON DUPLICATE KEY UPDATE updated_at = NOW();

INSERT INTO sys_users (user_name, password, real_name, phone, role_code, status) VALUES
('cs002', '$2a$10$89y8.Zg5l6plG95BSpa6t.z1UcxH0H1OKvZs2kUZpzaeb/w3sybSy', '客服小李', '13800138002', 'customer_service', 1)
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- 测试用户账号 (用于用户端测试)
INSERT INTO sys_users (user_name, password, real_name, phone, role_code, status) VALUES
('u_test1', '$2a$10$89y8.Zg5l6plG95BSpa6t.z1UcxH0H1OKvZs2kUZpzaeb/w3sybSy', '测试用户1', '13900139001', 'user', 1)
ON DUPLICATE KEY UPDATE updated_at = NOW();

-- 3. 创建客服信息表记录
CREATE TABLE IF NOT EXISTS t_customer_service (
    cs_id VARCHAR(32) PRIMARY KEY,
    cs_name VARCHAR(64) NOT NULL,
    dept_id VARCHAR(32) NOT NULL DEFAULT 'DEFAULT',
    team_id VARCHAR(32) DEFAULT '',
    skill_tags VARCHAR(128) DEFAULT '',
    status TINYINT(1) NOT NULL DEFAULT 1,
    current_status TINYINT(1) NOT NULL DEFAULT 0,
    is_online TINYINT(1) NOT NULL DEFAULT 0,
    last_heartbeat DATETIME,
    role TINYINT(1) NOT NULL DEFAULT 0,
    password_hash VARCHAR(128) DEFAULT '',
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 插入 AI 助理客服 (CS9999)
INSERT INTO t_customer_service (cs_id, cs_name, dept_id, status, current_status, is_online, create_time, update_time) VALUES
('CS9999', '智能助理', 'AI', 1, 0, 1, NOW(), NOW())
ON DUPLICATE KEY UPDATE cs_name = '智能助理', is_online = 1, update_time = NOW();

-- 插入人工客服 (关联 sys_users 中的客服账号)
-- 根据 sys_users 的 id 生成 cs_id (CS + id)
INSERT INTO t_customer_service (cs_id, cs_name, dept_id, status, current_status, is_online, last_heartbeat, create_time, update_time)
SELECT 
    CONCAT('CS', u.id) as cs_id,
    u.real_name as cs_name,
    'DEFAULT' as dept_id,
    1 as status,
    0 as current_status,
    1 as is_online,
    NOW() as last_heartbeat,
    NOW() as create_time,
    NOW() as update_time
FROM sys_users u
WHERE u.role_code = 'customer_service' AND u.status = 1
ON DUPLICATE KEY UPDATE 
    cs_name = VALUES(cs_name),
    is_online = 1,
    last_heartbeat = NOW(),
    update_time = NOW();

-- 4. 创建 AI Job 表 (chatModel 使用)
CREATE TABLE IF NOT EXISTS ai_jobs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    job_id VARCHAR(128) NOT NULL UNIQUE,
    conversation_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    result TEXT,
    error TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_job_id (job_id),
    INDEX idx_conv_id (conversation_id)
);

-- 5. 创建会话摘要表 (chatModel 使用)
CREATE TABLE IF NOT EXISTS conversation_summaries (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    conversation_id VARCHAR(64) NOT NULL,
    summary TEXT,
    trace_id VARCHAR(128),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_conv_id (conversation_id)
);

-- 完成提示
SELECT 'Docker environment database initialized successfully!' as message;
SELECT CONCAT('Total users created: ', COUNT(*)) as user_count FROM sys_users;
SELECT CONCAT('Total customer service agents: ', COUNT(*)) as cs_count FROM t_customer_service;
