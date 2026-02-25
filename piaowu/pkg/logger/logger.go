// Package logger 提供统一的日志记录功能
// 基于 uber-go/zap 实现，支持：
// 1. 控制台输出
// 2. 文件输出（可配置服务名和日志目录）
// 3. JSON 格式日志
// 4. 多级别日志（Debug/Info/Warn/Error/Fatal）
// 5. 链路追踪 ID (TraceID) 支持
//
// 使用方式：
//
//	// 初始化日志（仅控制台）
//	logger.InitLogger()
//
//	// 初始化日志（控制台+文件）
//	logger.InitLoggerWithFile("gateway", "./logs")
//
//	// 记录日志
//	logger.Info("服务启动", zap.String("port", "8080"))
package logger

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ============ 全局变量 ============

var (
	// Logger 全局 logger 实例
	// 在 InitLogger 或 InitLoggerWithFile 初始化后可用
	Logger *zap.Logger
)

// ============ 日志初始化函数 ============

// InitLogger 初始化 zap logger（仅输出到控制台）
// 适用于开发环境或不需要文件日志的场景
// 日志级别: Info
// 输出格式: JSON
// 返回:
//   - error: 初始化失败时返回错误
func InitLogger() error {
	config := zap.NewProductionConfig()

	// 配置日志级别
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)

	// 配置时间格式
	config.EncoderConfig.TimeKey = "ts"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// 配置调用者信息
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// 配置日志级别格式
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder

	// 配置消息键
	config.EncoderConfig.MessageKey = "msg"

	// 构建 logger
	var err error
	Logger, err = config.Build(
		zap.AddCallerSkip(1), // 跳过一层调用栈，显示实际调用位置
	)
	if err != nil {
		return err
	}

	return nil
}

// InitLoggerWithFile 初始化 zap logger（同时输出到控制台和文件）
// 适用于生产环境，支持日志持久化
// 控制台日志级别: Info
// 文件日志级别: Debug（更详细）
// 输出格式: JSON
//
// 参数:
//   - serviceName: 服务名称（如 "gateway", "customer"），用于日志文件名
//   - logDir: 日志目录路径（如 "./logs"），不存在将自动创建
//
// 返回:
//   - error: 初始化失败时返回错误
func InitLoggerWithFile(serviceName, logDir string) error {
	// 创建日志目录
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// 日志文件路径
	logFile := filepath.Join(logDir, serviceName+".log")

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建文件输出
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// 创建两个 core：一个输出到控制台，一个输出到文件
	consoleEncoder := zapcore.NewJSONEncoder(encoderConfig)
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// 控制台输出（所有级别）
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		zapcore.InfoLevel,
	)

	// 文件输出（所有级别）
	fileCore := zapcore.NewCore(
		fileEncoder,
		zapcore.AddSync(file),
		zapcore.DebugLevel, // 文件中记录更详细的日志
	)

	// 合并两个 core
	core := zapcore.NewTee(consoleCore, fileCore)

	// 创建 logger
	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return nil
}

// ============ 日志管理函数 ============

// Sync 刷新日志缓冲区
// 应在程序退出前调用，确保所有日志写入文件
// 建议使用 defer logger.Sync() 调用
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}

// ============ 日志记录函数 ============

// Info 记录 Info 级别日志
// 用于记录正常的业务运行信息
// 参数:
//   - msg: 日志消息
//   - fields: 可选的结构化字段
func Info(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Info(msg, fields...)
	}
}

// Warn 记录 Warn 级别日志
// 用于记录需要注意但不影响正常运行的信息
// 参数:
//   - msg: 日志消息
//   - fields: 可选的结构化字段
func Warn(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Warn(msg, fields...)
	}
}

// Error 记录 Error 级别日志
// 用于记录错误信息，应包含错误详情便于排查
// 参数:
//   - msg: 日志消息
//   - fields: 可选的结构化字段（建议包含 zap.Error(err)）
func Error(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Error(msg, fields...)
	}
}

// Fatal 记录 Fatal 级别日志并退出程序
// 调用后程序将立即退出（os.Exit(1)）
// 仅在不可恢复的错误时使用
// 参数:
//   - msg: 日志消息
//   - fields: 可选的结构化字段
func Fatal(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Fatal(msg, fields...)
	}
}

// Debug 记录 Debug 级别日志
// 用于记录调试信息，仅在设置 Debug 级别时输出
// 参数:
//   - msg: 日志消息
//   - fields: 可选的结构化字段
func Debug(msg string, fields ...zap.Field) {
	if Logger != nil {
		Logger.Debug(msg, fields...)
	}
}
