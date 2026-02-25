//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	passwords := []string{"admin123", "123456", "password"}

	fmt.Println("=== Bcrypt Password Generator ===")
	fmt.Println("Use these in your SQL INSERT statements")
	fmt.Println()

	for _, pwd := range passwords {
		hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		if err != nil {
			fmt.Printf("Error hashing '%s': %v\n", pwd, err)
			continue
		}
		fmt.Printf("Password: %-12s => Hash: %s\n", pwd, string(hash))
	}

	fmt.Println()
	fmt.Println("=== SQL Example ===")
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	fmt.Println(`
-- 1. 创建角色表 (如果不存在)
CREATE TABLE IF NOT EXISTS sys_roles (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    role_code VARCHAR(32) NOT NULL UNIQUE,
    role_name VARCHAR(32) NOT NULL,
    remark VARCHAR(255),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 2. 插入角色
INSERT IGNORE INTO sys_roles (role_code, role_name, remark) VALUES
('admin', '管理员', '拥有所有权限'),
('customer_service', '客服', '会话管理和提交申请权限');

-- 3. 创建用户表 (如果不存在)
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

-- 4. 插入管理员用户 (密码: admin123)
INSERT INTO sys_users (user_name, password, real_name, phone, role_code, status) VALUES
('admin', '` + string(hash) + `', '管理员', '13800138000', 'admin', 1)
ON DUPLICATE KEY UPDATE updated_at = NOW();
`)
}
