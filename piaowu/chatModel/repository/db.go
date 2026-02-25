package repository

import (
	"example_shop/chatModel/config"
	"example_shop/chatModel/model"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 是全局数据库操作句柄，供全项目引用
var DB *gorm.DB

// InitDB 初始化 MySQL 数据库连接池，并执行 Schema 自动迁移
func InitDB() error {
	// 从全局配置获取数据源名称 (DSN)
	dsn := config.GlobalConfig.Database.DSN
	if dsn == "" {
		return fmt.Errorf("database DSN is empty, please check config.yaml")
	}

	// 配置 GORM 日志器，使其在控制台输出带颜色的 SQL 语句，方便开发调试
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // 慢 SQL 阈值
			LogLevel:                  logger.Info, // 日志级别
			IgnoreRecordNotFoundError: true,        // 忽略 ErrRecordNotFound 错误
			Colorful:                  true,        // 启用彩色打印
		},
	)

	// 开启数据库连接
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to mysql: %w", err)
	}

	// 核心步骤：自动迁移 (AutoMigrate)
	// 确保项目所需的 AI 任务表、审计表、摘要表在数据库中自动建立
	err = db.AutoMigrate(
		&model.ConversationSummary{},
		&model.AISuggestionLog{},
		&model.RiskAlertLog{},
		&model.AIToolAuditLog{},
		&model.AIJob{},
		&model.ConvMessage{}, // 业务消息表映射
	)
	if err != nil {
		return fmt.Errorf("database auto migrate failed: %w", err)
	}

	DB = db
	log.Println("✅ MySQL 数据库连接并初始化成功")
	return nil
}
