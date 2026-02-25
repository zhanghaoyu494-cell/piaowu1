// Package config 提供 Gateway 网关服务的配置管理功能
// 主要功能：
// 1. 加载和解析YAML配置文件
// 2. 管理服务地址、JWT等配置项
// 3. 提供全局配置访问入口
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultJWTSecret      = "piaowu-secret-key-2026"
	defaultJWTExpireHours = 24
)

// Config 网关服务的根配置结构
type Config struct {
	Server   ServerConfig             `mapstructure:"server"`
	Services map[string]ServiceConfig `mapstructure:"services"`
	JWT      JWTConfig                `mapstructure:"jwt"`
	Trace    TraceConfig              `mapstructure:"trace"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Address string `mapstructure:"address"`
}

// ServiceConfig 后端服务配置项
type ServiceConfig struct {
	Name    string `mapstructure:"name"`
	Address string `mapstructure:"address"`
}

// JWTConfig JWT配置结构
type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

// TraceConfig 链路追踪配置
// 支持 Jaeger 导出，可通过 enabled 字段控制开关
type TraceConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	ServiceName   string        `mapstructure:"service_name"`
	JaegerEnabled bool          `mapstructure:"jaeger_enabled"`
	Endpoint      string        `mapstructure:"endpoint"`
	SampleRate    float64       `mapstructure:"sample_rate"`
	BatchSize     int           `mapstructure:"batch_size"`
	FlushInterval time.Duration `mapstructure:"flush_interval"`
}

// GlobalConfig 全局配置实例，由InitConfig初始化
var GlobalConfig *Config

// InitConfig 初始化配置
// 从指定YAML文件加载配置，并设置默认值
// 参数:
//   - configPath: 配置文件路径
//
// 返回: 配置加载错误
func InitConfig(configPath string) error {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	viper.SetDefault("jwt.secret", defaultJWTSecret)
	viper.SetDefault("jwt.expire_hours", defaultJWTExpireHours)

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	GlobalConfig = &Config{}
	if err := viper.Unmarshal(GlobalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// GetJWTSecret 获取JWT签名密钥
// 优先从配置文件读取，如果未配置则返回默认值
// 返回: JWT密钥字符串
func GetJWTSecret() string {
	if GlobalConfig != nil && GlobalConfig.JWT.Secret != "" {
		return GlobalConfig.JWT.Secret
	}
	return defaultJWTSecret
}

// GetJWTExpireHours 获取Token过期时间
// 优先从配置文件读取，如果未配置则返回默认24小时
// 返回: 过期时间（小时数）
func GetJWTExpireHours() int {
	if GlobalConfig != nil && GlobalConfig.JWT.ExpireHours > 0 {
		return GlobalConfig.JWT.ExpireHours
	}
	return defaultJWTExpireHours
}
