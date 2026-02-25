// Package model 定义用户认证相关的数据模型
// 包含：
// - SysRole: 系统角色表（管理员、客服）
// - SysUser: 系统用户表（用于登录认证）
// - 密码加密/校验工具函数（bcrypt）
// - 角色常量和角色判断方法
package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ==================== 用户认证相关模型 ====================

// SysRole 系统角色表
// 用于定义系统中的角色类型，当前支持两种固定角色：
// - admin: 管理员（拥有所有权限）
// - customer_service: 客服（仅拥有会话管理和提交申请权限）
type SysRole struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`                                           // 角色ID
	RoleCode  string    `gorm:"column:role_code;type:varchar(32);not null;uniqueIndex" json:"role_code"`       // 角色编码（唯一标识）
	RoleName  string    `gorm:"column:role_name;type:varchar(32);not null" json:"role_name"`                   // 角色名称（显示名称）
	Remark    string    `gorm:"column:remark;type:varchar(255)" json:"remark"`                                 // 备注说明
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`                                           // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`                                           // 更新时间
}

// TableName 指定 SysRole 对应的数据库表名
func (SysRole) TableName() string {
	return "sys_roles"
}

// SysUser 系统用户表
// 用于管理员和客服的登录认证
// 密码字段使用 bcrypt 加密存储，并不返回给前端序列化
type SysUser struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`                                          // 用户ID
	UserName  string         `gorm:"column:user_name;type:varchar(32);not null;uniqueIndex" json:"user_name"`     // 登录账号（唯一）
	Password  string         `gorm:"column:password;type:varchar(128);not null" json:"-"`                         // 密码（bcrypt加密，不返回）
	RealName  string         `gorm:"column:real_name;type:varchar(32);not null" json:"real_name"`                 // 真实姓名
	Phone     string         `gorm:"column:phone;type:varchar(11)" json:"phone"`                                  // 手机号
	RoleCode  string         `gorm:"column:role_code;type:varchar(32);not null;index" json:"role_code"`           // 角色编码
	Status    int8           `gorm:"column:status;type:tinyint(1);default:1" json:"status"`                       // 状态: 1-正常 0-禁用
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`                                         // 创建时间
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`                                         // 更新时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`                                                              // 软删除时间
}

// TableName 指定 SysUser 对应的数据库表名
func (SysUser) TableName() string {
	return "sys_users"
}

// ==================== 密码加密工具函数 ====================

// HashPassword 对明文密码进行 bcrypt 加密
// 使用 bcrypt.DefaultCost 作为计算成本
// 参数:
//   - password: 明文密码
//
// 返回:
//   - string: 加密后的密码字符串
//   - error: 加密失败时返回错误
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 校验密码是否正确
// 比较加密后的密码与明文密码是否匹配
// 参数:
//   - hashedPassword: 数据库中存储的加密密码
//   - password: 用户输入的明文密码
//
// 返回:
//   - bool: 密码匹配返回 true，否则返回 false
func CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// ==================== 角色常量定义 ====================

const (
	RoleAdmin           = "admin"            // 管理员角色编码（拥有所有权限）
	RoleCustomerService = "customer_service" // 客服角色编码（会话管理+提交申请权限）
)

// ==================== 角色判断方法 ====================

// IsAdmin 判断用户是否为管理员
// 返回:
//   - bool: 是管理员返回 true
func (u *SysUser) IsAdmin() bool {
	return u.RoleCode == RoleAdmin
}

// IsCustomerService 判断用户是否为客服
// 返回:
//   - bool: 是客服返回 true
func (u *SysUser) IsCustomerService() bool {
	return u.RoleCode == RoleCustomerService
}
