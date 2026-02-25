package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 定义了 chatModel 服务的所有核心配置项
// 使用 mapstructure 标签使其能够与 YAML 字段精确映射
type Config struct {
	App struct {
		Name string // 应用名称
		Port int    // HTTP 服务运行端口
	}
	Database struct {
		DSN string // 数据库连接字符串 (Data Source Name)
	}
	AI struct {
		Provider string `mapstructure:"provider"` // AI 供应商 (ark, qianfan, openai)
		APIKey   string `mapstructure:"api_key"`  // LLM API 密钥
		BaseURL  string `mapstructure:"base_url"` // LLM API 接入地址
		Model    string `mapstructure:"model"`    // 使用的模型名称
	}
	Embedding struct {
		Provider  string `mapstructure:"provider"`
		APIKey    string `mapstructure:"api_key"`
		Model     string `mapstructure:"model"`
		Dimension int    `mapstructure:"dimension"` // 向量维度，初始化 Milvus 时使用
	}
	Milvus struct {
		Host       string `mapstructure:"host"`
		Port       int    `mapstructure:"port"`
		Collection string `mapstructure:"collection"` // 向量集合名称
		TopK       int    `mapstructure:"top_k"`      // 检索时召回的最相似数量
	}
}

// GlobalConfig 持有了通过 InitConfig 加载后的全局配置状态
var GlobalConfig Config

// InitConfig 使用 Viper 库加载并解析配置文件或环境变量
func InitConfig() error {
	v := viper.New()

	// 1. 设置配置查找策略：按优先级尝试不同的查找路径
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("chatModel/config") // 兼容从项目根目录启动
	v.AddConfigPath("config")           // 兼容从子目录启动
	v.AddConfigPath(".")                // 兼容当前目录启动

	// 2. 注入默认值，确保在配置文件缺失时服务仍能尝试以基准参数启动
	v.SetDefault("app.name", "chatModel")
	v.SetDefault("app.port", 8082)
	v.SetDefault("database.dsn", "root:Zhyzhy666888@tcp(121.5.9.239:3306)/ccc?charset=utf8mb4&parseTime=True&loc=Local")
	v.SetDefault("ai.provider", "qianfan")
	v.SetDefault("ai.base_url", "https://qianfan.baidubce.com/v2")
	v.SetDefault("ai.model", "ernie-4.0-8k-latest")
	v.SetDefault("ai.api_key", "bce-v3/ALTAK-l2Yn6ovAqIV9SXzzFn4x0/39ec057f4bc153eae1ccad9c6223f242de0a9842")

	// 3. 读取配置文件内容
	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("💡 提示: 未发现物理配置文件，将启用环境变量/默认值兜底: %v\n", err)
	}

	// 4. 读取环境变量 (优先级最高，常用于 Docker/K8s 部署)
	// 转换规则：APP_PORT -> app.port
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 5. 将解析结果反序列化填充到 GlobalConfig 结构体
	if err := v.Unmarshal(&GlobalConfig); err != nil {
		return fmt.Errorf("配置解析映射失败: %w", err)
	}

	// 简单的初始化检查与日志打印
	keyPrefix := ""
	if len(GlobalConfig.AI.APIKey) >= 10 {
		keyPrefix = GlobalConfig.AI.APIKey[:10]
	}

	fmt.Printf("✅ 配置加载完成: Port=%d, Provider=%s, Model=%s, APIKey预览=%s...\n",
		GlobalConfig.App.Port, GlobalConfig.AI.Provider, GlobalConfig.AI.Model, keyPrefix)

	return nil
}
