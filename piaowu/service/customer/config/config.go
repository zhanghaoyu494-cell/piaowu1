// Package config 提供 Customer RPC 服务的配置管理
// 使用 Viper 加载 YAML 配置文件，支持以下配置项：
// - Server: 服务器配置（服务名称、监听地址）
// - MySQL: 数据库连接配置
// - Redis: 缓存连接配置
// - Trace: 链路追踪配置（Jaeger）
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ============ 配置结构体定义 ============

// Config 服务总配置结构
// 包含服务器、MySQL、Redis、链路追踪四部分配置
type Config struct {
	Server ServerConfig `mapstructure:"server"` // 服务器配置
	MySQL  MySQLConfig  `mapstructure:"mysql"`  // MySQL数据库配置
	Redis  RedisConfig  `mapstructure:"redis"`  // Redis缓存配置
	Trace  TraceConfig  `mapstructure:"trace"`  // 链路追踪配置
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Name    string `mapstructure:"name"`    // 服务名称（如 "customer"）
	Address string `mapstructure:"address"` // 监听地址（如 ":8082"）
}

// MySQLConfig MySQL数据库配置
type MySQLConfig struct {
	Host         string `mapstructure:"host"`           // 数据库主机
	Port         int    `mapstructure:"port"`           // 数据库端口
	User         string `mapstructure:"user"`           // 数据库用户名
	Password     string `mapstructure:"password"`       // 数据库密码
	Database     string `mapstructure:"database"`       // 数据库名
	Charset      string `mapstructure:"charset"`        // 字符集（如 utf8mb4）
	ParseTime    bool   `mapstructure:"parse_time"`     // 是否解析时间类型
	Loc          string `mapstructure:"loc"`            // 时区设置
	MaxIdleConns int    `mapstructure:"max_idle_conns"` // 最大空闲连接数
	MaxOpenConns int    `mapstructure:"max_open_conns"` // 最大打开连接数
}

// RedisConfig Redis缓存配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`     // Redis主机
	Port     int    `mapstructure:"port"`     // Redis端口
	Password string `mapstructure:"password"` // Redis密码（无密码留空）
	DB       int    `mapstructure:"db"`       // 数据库索引（默认0）
}

// TraceConfig 链路追踪配置
// 支持 Jaeger 导出，可通过 enabled 字段控制开关
type TraceConfig struct {
	Enabled       bool          `mapstructure:"enabled"`        // 是否启用链路追踪
	ServiceName   string        `mapstructure:"service_name"`   // 服务名称（显示在 Jaeger UI）
	JaegerEnabled bool          `mapstructure:"jaeger_enabled"` // 是否启用 Jaeger 导出
	Endpoint      string        `mapstructure:"endpoint"`       // Jaeger Collector 地址
	SampleRate    float64       `mapstructure:"sample_rate"`    // 采样率 (0.0~1.0)
	BatchSize     int           `mapstructure:"batch_size"`     // 批量发送大小
	FlushInterval time.Duration `mapstructure:"flush_interval"` // 刷新间隔
}

// ============ 全局配置实例 ============

// GlobalConfig 全局配置实例
// 在 InitConfig 后可以通过该变量访问配置
var GlobalConfig *Config

// ============ 配置初始化函数 ============

// InitConfig 初始化配置
// 从指定的 YAML 文件加载配置并解析到 GlobalConfig
// 参数:
//   - configPath: 配置文件路径
//
// 返回:
//   - error: 配置加载或解析失败时返回错误
func InitConfig(configPath string) error {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	_ = viper.BindEnv("server.name")
	_ = viper.BindEnv("server.address")
	_ = viper.BindEnv("mysql.host")
	_ = viper.BindEnv("mysql.port")
	_ = viper.BindEnv("mysql.user")
	_ = viper.BindEnv("mysql.password")
	_ = viper.BindEnv("mysql.database")
	_ = viper.BindEnv("mysql.charset")
	_ = viper.BindEnv("mysql.parse_time")
	_ = viper.BindEnv("mysql.loc")
	_ = viper.BindEnv("mysql.max_idle_conns")
	_ = viper.BindEnv("mysql.max_open_conns")
	_ = viper.BindEnv("redis.host")
	_ = viper.BindEnv("redis.port")
	_ = viper.BindEnv("redis.password")
	_ = viper.BindEnv("redis.db")
	_ = viper.BindEnv("trace.enabled")
	_ = viper.BindEnv("trace.service_name")
	_ = viper.BindEnv("trace.jaeger_enabled")
	_ = viper.BindEnv("trace.endpoint")
	_ = viper.BindEnv("trace.sample_rate")
	_ = viper.BindEnv("trace.batch_size")
	_ = viper.BindEnv("trace.flush_interval")

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	GlobalConfig = &Config{}
	if err := viper.Unmarshal(GlobalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// ============ 配置工具方法 ============

// DSN 生成 MySQL 数据源名称字符串
// 格式: user:password@tcp(host:port)/database?charset=xxx&parseTime=xxx&loc=xxx
// 返回:
//   - string: 可用于 GORM 连接的 DSN 字符串
func (c *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.Charset, c.ParseTime, c.Loc)
}
