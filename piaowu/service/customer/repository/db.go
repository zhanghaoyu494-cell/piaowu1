// Package repository 提供数据访问层（Repository Layer）功能
// 包括：
// - MySQL 数据库连接与操作（使用 GORM）
// - Redis 缓存连接与操作
// - 数据表自动迁移
// - 缓存工具函数（JSON序列化/反序列化）
// - 默认数据初始化（角色、管理员账号）
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"example_shop/pkg/logger"
	"example_shop/service/customer/config"
	"example_shop/service/customer/model"
	"example_shop/service/customer/repository/plugin"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ============ 全局数据库实例 ============

var (
	DB    *gorm.DB      // MySQL 数据库连接实例
	Redis *redis.Client // Redis 客户端实例
)

// ============ 数据库初始化 ============

// InitDB 初始化数据库连接
// 功能：
// 1. 使用配置建立 MySQL 连接
// 2. 配置连接池参数
// 3. 注册 TracePlugin 实现日志链路追踪
// 4. 初始化 Redis 连接
// 返回:
//   - error: 初始化失败时返回错误
func InitDB() error {
	cfg := config.GlobalConfig.MySQL
	dsn := cfg.DSN()

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		// 使用自定义的 TraceLogger
		Logger: plugin.NewTraceLogger(),
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	// 注册 TracePlugin
	if err := DB.Use(&plugin.TracePlugin{}); err != nil {
		return fmt.Errorf("failed to register trace plugin: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	// 连接的最大生命周期，防止连接长时间未使用被MySQL服务器关闭
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
	// 连接的最大空闲时间，超过此时间的空闲连接会被关闭
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	logger.Info("Database connected successfully")

	if err := InitRedis(); err != nil {
		logger.Warn("Redis init failed", zap.Error(err))
	} else {
		logger.Info("Redis connected successfully")
	}

	return nil
}

// InitRedis 初始化 Redis 连接
// 使用配置建立 Redis 连接并测试连通性
// 返回:
//   - error: 连接失败时返回错误
func InitRedis() error {
	cfg := config.GlobalConfig.Redis
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	c := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		return err
	}

	Redis = c
	return nil
}

// ============ Redis 缓存工具函数 ============

// CacheGetJSON 从 Redis 获取 JSON 数据并反序列化
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//   - dest: 目标对象指针（将解析JSON到该对象）
//
// 返回:
//   - bool: 是否命中缓存
//   - error: 操作错误信息
func CacheGetJSON(ctx context.Context, key string, dest interface{}) (bool, error) {
	if Redis == nil {
		return false, nil
	}
	val, err := Redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		logger.Error("Redis get failed", zap.Error(err))
		return false, err
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		logger.Error("JSON unmarshal failed", zap.Error(err))
		return false, err
	}
	return true, nil
}

// CacheSetJSON 将数据序列化为 JSON 并存入 Redis
// 参数:
//   - ctx: 上下文
//   - key: 缓存键
//   - src: 源数据（将被序列化为JSON）
//   - ttl: 过期时间
//
// 返回:
//   - error: 操作错误信息
func CacheSetJSON(ctx context.Context, key string, src interface{}, ttl time.Duration) error {
	if Redis == nil {
		return nil
	}
	bytes, err := json.Marshal(src)
	if err != nil {
		logger.Error("JSON marshal failed", zap.Error(err))
		return err
	}
	if err := Redis.Set(ctx, key, bytes, ttl).Err(); err != nil {
		logger.Error("Redis set failed", zap.Error(err))
		return err
	}
	return nil
}

// CacheDel 删除 Redis 缓存键
// 支持批量删除多个键
// 参数:
//   - ctx: 上下文
//   - keys: 要删除的缓存键列表
//
// 返回:
//   - error: 删除失败时返回错误
func CacheDel(ctx context.Context, keys ...string) error {
	if Redis == nil || len(keys) == 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return Redis.Del(ctx, keys...).Err()
}

// ============ 数据表迁移 ============

// MigrateTables 执行数据表自动迁移
// 功能：
// 1. 根据 Model 结构体自动创建/更新表结构
// 2. 修复 approval_time 字段允许为 NULL
// 3. 初始化默认角色和管理员账号
// 返回:
//   - error: 迁移失败时返回错误
func MigrateTables() error {
	logger.Info("Starting database migration...")
	err := DB.AutoMigrate(
		&model.ShiftConfig{},
		&model.CustomerService{},
		&model.Schedule{},
		&model.Conversation{},
		&model.ConvCategory{},
		&model.ConvMessage{},
		&model.QuickReply{},
		&model.LeaveTransfer{},
		&model.LeaveAuditLog{}, // 请假调班审批日志
		&model.ConvTag{},
		&model.ConvTransfer{},      // 会话转接记录表
		&model.MsgCategory{},       // 消息分类维度表
		&model.ClassifyAdjustLog{}, // 分类调整日志表
		// 新增用户认证相关表
		&model.SysRole{},
		&model.SysUser{},
	)
	if err != nil {
		// 如果只是提示表已存在，记录警告但继续运行
		if strings.Contains(err.Error(), "already exists") {
			logger.Warn("AutoMigrate reported table already exists, ignoring...", zap.Error(err))
		} else {
			return fmt.Errorf("failed to migrate tables: %w", err)
		}
	}

	if err := DB.Exec("ALTER TABLE t_leave_transfer MODIFY COLUMN approval_time DATETIME NULL DEFAULT NULL").Error; err != nil {
		logger.Warn("migrate approval_time column failed", zap.Error(err))
	}

	logger.Info("Tables migrated successfully")

	// 初始化默认角色和管理员账号
	if err := InitDefaultRolesAndAdmin(); err != nil {
		logger.Warn("init default roles and admin failed", zap.Error(err))
	}

	return nil
}

// InitDefaultRolesAndAdmin 初始化默认角色和管理员账号
// 创建以下默认数据：
// 1. 管理员角色（admin）
// 2. 客服角色（customer_service）
// 3. 默认管理员账号（用户名: admin, 密码: admin123）
// 返回:
//   - error: 初始化失败时返回错误
func InitDefaultRolesAndAdmin() error {
	// 初始化管理员角色
	var adminRole model.SysRole
	result := DB.Where("role_code = ?", model.RoleAdmin).First(&adminRole)
	if result.Error != nil {
		adminRole = model.SysRole{
			RoleCode: model.RoleAdmin,
			RoleName: "系统管理员",
			Remark:   "拥有系统全部权限",
		}
		if err := DB.Create(&adminRole).Error; err != nil {
			return fmt.Errorf("create admin role failed: %w", err)
		}
		logger.Info("Created default admin role")
	}

	// 初始化客服角色
	var csRole model.SysRole
	result = DB.Where("role_code = ?", model.RoleCustomerService).First(&csRole)
	if result.Error != nil {
		csRole = model.SysRole{
			RoleCode: model.RoleCustomerService,
			RoleName: "客服专员",
			Remark:   "仅拥有提交申请和会话管理权限",
		}
		if err := DB.Create(&csRole).Error; err != nil {
			return fmt.Errorf("create customer_service role failed: %w", err)
		}
		logger.Info("Created default customer_service role")
	}

	// 初始化默认管理员账号
	var adminUser model.SysUser
	result = DB.Where("user_name = ?", "admin").First(&adminUser)
	if result.Error != nil {
		// 使用bcrypt加密默认密码
		hashedPassword, err := model.HashPassword("admin123")
		if err != nil {
			return fmt.Errorf("hash password failed: %w", err)
		}
		adminUser = model.SysUser{
			UserName: "admin",
			Password: hashedPassword,
			RealName: "系统管理员",
			Phone:    "",
			RoleCode: model.RoleAdmin,
			Status:   1,
		}
		if err := DB.Create(&adminUser).Error; err != nil {
			return fmt.Errorf("create admin user failed: %w", err)
		}
		logger.Info("Created default admin user (username: admin, password: admin123)")
	}

	return nil
}
