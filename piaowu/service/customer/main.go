// Package main 是 Customer RPC 服务的入口
// 该服务基于 Kitex 框架，提供客服排班会话管理的核心业务功能：
// 1. 客服信息管理（查询客服信息、客服列表）
// 2. 班次配置管理（创建、更新、删除班次模板）
// 3. 排班管理（手动/自动排班、排班表查询）
// 4. 请假调班（申请、审批、查询）
// 5. 会话管理（会话列表、消息收发、快捷回复）
// 6. 会话分类与标签（分类管理、标签管理）
// 7. 统计看板（会话统计数据）
// 8. 用户认证（登录、注册、获取用户信息）
//
// 服务启动流程：
// 1. 初始化日志系统
// 2. 加载配置文件
// 3. 初始化链路追踪（Jaeger）
// 4. 初始化数据库连接（MySQL + Redis）
// 5. 执行数据表迁移
// 6. 启动 Kitex RPC 服务器（带追踪中间件）
package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"example_shop/pkg/logger"
	"example_shop/pkg/trace"
	"example_shop/service/customer/config"
	"example_shop/service/customer/handler"
	"example_shop/service/customer/kitex_gen/customer/customerservice"
	"example_shop/service/customer/repository"

	"github.com/cloudwego/kitex/server"
	"go.uber.org/zap"
)

// main 服务入口函数
// 按顺序执行：日志初始化 -> 配置加载 -> 链路追踪初始化 -> 数据库初始化 -> 表迁移 -> 启动服务
func main() {

	// 1. 初始化日志系统（输出到控制台和文件）
	if err := logger.InitLoggerWithFile("customer", "./logs"); err != nil {
		panic("Failed to init logger: " + err.Error())
	}
	defer logger.Sync() // 确保日志缓冲区刷新

	// 2. 初始化配置，支持多个配置文件路径候选
	if err := config.InitConfig(mustResolveConfigPath(
		"service/customer/config/config.yaml",
		"config/config.yaml",
	)); err != nil {
		logger.Fatal("Failed to init config", zap.Error(err))
	}

	// 3. 初始化链路追踪（Jaeger）
	initTracing()
	defer trace.ShutdownJaeger()

	// 4. 初始化数据库连接（包括 MySQL 和 Redis）
	if err := repository.InitDB(); err != nil {
		logger.Fatal("Failed to init database", zap.Error(err))
	}

	// 5. 执行数据表迁移（自动创建/更新表结构）
	logger.Info("Executing database migrations...")
	if err := repository.MigrateTables(); err != nil {
		logger.Fatal("Failed to migrate tables", zap.Error(err))
	}
	logger.Info("Database migrations completed successfully")
	fmt.Println("Customer RPC Service Starting...")

	// 6. 解析服务监听地址
	addr, err := net.ResolveTCPAddr("tcp", config.GlobalConfig.Server.Address)
	if err != nil {
		logger.Fatal("Failed to resolve service address", zap.Error(err))
	}

	// 7. 获取服务名称
	serviceName := config.GlobalConfig.Server.Name
	if serviceName == "" {
		serviceName = "CustomerService"
	}

	// 8. 创建并启动 Kitex RPC 服务器（带追踪中间件）
	svr := customerservice.NewServer(
		handler.NewCustomerServiceHandler(),                             // 业务处理器
		server.WithServiceAddr(addr),                                    // 服务地址
		server.WithMiddleware(trace.ServerTraceMiddleware(serviceName)), // 链路追踪中间件
	)

	logger.Info("Customer service started",
		zap.String("address", config.GlobalConfig.Server.Address),
		zap.String("service", serviceName),
		zap.Bool("tracing_enabled", config.GlobalConfig.Trace.Enabled),
	)
	if err := svr.Run(); err != nil {
		logger.Fatal("Server stopped with error", zap.Error(err))
	}
}

// initTracing 初始化链路追踪
func initTracing() {
	cfg := config.GlobalConfig.Trace
	if !cfg.Enabled {
		logger.Info("Tracing is disabled")
		return
	}

	// 配置 Jaeger Exporter
	jaegerCfg := &trace.JaegerConfig{
		Enabled:       cfg.JaegerEnabled,
		ServiceName:   cfg.ServiceName,
		Endpoint:      cfg.Endpoint,
		SampleRate:    cfg.SampleRate,
		BatchSize:     cfg.BatchSize,
		FlushInterval: cfg.FlushInterval,
	}

	// 设置默认值
	if jaegerCfg.ServiceName == "" {
		jaegerCfg.ServiceName = "customer-service"
	}
	if jaegerCfg.Endpoint == "" {
		jaegerCfg.Endpoint = "http://jaeger:4318/v1/traces" // OTLP HTTP 端点
	}
	if jaegerCfg.SampleRate <= 0 {
		jaegerCfg.SampleRate = 0.1 // 默认 10% 采样
	}

	trace.InitJaeger(jaegerCfg)
	logger.Info("Tracing initialized",
		zap.String("service", jaegerCfg.ServiceName),
		zap.Bool("jaeger_enabled", jaegerCfg.Enabled),
		zap.Float64("sample_rate", jaegerCfg.SampleRate),
	)
}

// ============ 配置文件路径解析工具函数 ============

// mustResolveConfigPath 解析配置文件路径
// 支持多个候选路径，依次尝试直到找到存在的文件
// 如果相对路径找不到，会尝试从项目根目录查找
// 参数:
//   - candidates: 配置文件路径候选列表
//
// 返回:
//   - string: 找到的配置文件路径，如果都不存在则返回第一个候选
func mustResolveConfigPath(candidates ...string) string {
	for _, p := range candidates {
		if fileExists(p) {
			return p
		}
	}

	root, ok := findProjectRoot()
	if ok {
		for _, p := range candidates {
			if fileExists(filepath.Join(root, p)) {
				return filepath.Join(root, p)
			}
		}
	}

	return candidates[0]
}

// fileExists 检查文件是否存在
// 参数:
//   - path: 文件路径
//
// 返回:
//   - bool: 文件存在返回 true，否则返回 false
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// findProjectRoot 查找项目根目录
// 从当前工作目录向上遍历，查找包含 go.mod 文件的目录
// 返回:
//   - string: 项目根目录路径
//   - bool: 是否找到项目根目录
func findProjectRoot() (string, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	dir := wd
	for {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
