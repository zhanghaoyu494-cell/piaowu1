// Package main 是 Gateway 网关服务的入口
// 该服务作为系统的API网关，负责：
// 1. 接收前端HTTP请求并转发到后端RPC服务
// 2. 处理WebSocket实时通信
// 3. JWT身份认证和权限校验
// 4. 跨域(CORS)处理
// 5. 请求链路追踪(TraceID)
package main // 包声明

import ( // 导入依赖列表开始
	"net/http"      // HTTP服务：Handler/请求/响应等
	"net/url"       // URL编码：拼接安全的查询/缓存键参数
	"os"            // 依赖导入
	"path/filepath" // 依赖导入
	"strings"       // 字符串处理：Trim/Contains/Split等

	"example_shop/gateway/config"     // 网关配置管理
	"example_shop/gateway/handler"    // HTTP请求处理器
	"example_shop/gateway/middleware" // 中间件（认证、链路追踪等）
	"example_shop/gateway/router"     // 路由配置
	"example_shop/gateway/rpc"        // RPC客户端
	"example_shop/gateway/ws"         // WebSocket模块
	"example_shop/pkg/logger"         // 日志工具
	"example_shop/pkg/trace"          // 链路追踪

	"go.uber.org/zap" // 依赖导入
) // 导入依赖列表结束/或参数列表结束

func main() { // 函数定义/HTTP处理入口
	// 初始化日志系统（输出到控制台和文件）
	if err := logger.InitLoggerWithFile("gateway", "./logs"); err != nil { // 条件判断
		panic("Failed to init logger: " + err.Error()) // 逻辑处理
	} // 代码块结束
	defer logger.Sync() // 延迟执行清理逻辑

	// 初始化配置
	if err := config.InitConfig(mustResolveConfigPath( // 条件判断
		"gateway/config/config.yaml", // 依赖导入
		"config/config.yaml",         // 依赖导入
	)); err != nil { // 逻辑处理
		logger.Fatal("Failed to init config", zap.Error(err)) // 逻辑处理
	} // 代码块结束

	// 同步 JWT 配置到 middleware（确保登录和WebSocket验证使用同一密钥）
	middleware.SetJWTSecret(config.GetJWTSecret())          // 逻辑处理
	middleware.SetJWTExpireTime(config.GetJWTExpireHours()) // 逻辑处理

	// 初始化链路追踪（Jaeger）
	initTracing()                // 逻辑处理
	defer trace.ShutdownJaeger() // 延迟执行清理逻辑

	// 初始化 RPC 客户端
	customerService := config.GlobalConfig.Services["customer"]                                 // 逻辑处理
	customerClient, err := rpc.NewCustomerClient(customerService.Name, customerService.Address) // 调用并接收错误
	if err != nil {                                                                             // 条件判断
		logger.Fatal("Failed to create customer client", zap.Error(err)) // 逻辑处理
	} // 代码块结束
	defer customerClient.Close() // 延迟执行清理逻辑

	// 初始化 Handler
	customerHandler := handler.NewCustomerHandler(customerClient) // 逻辑处理

	// 初始化 WebSocket Hub
	chatModelService := config.GlobalConfig.Services["chatModel"]
	hub := ws.NewHub(customerClient, chatModelService.Address) // 传入 AI 服务地址
	go hub.Run()                                               // 启动并发协程

	// 设置路由
	mux := router.SetupRoutes(customerHandler, hub) // 逻辑处理

	// 应用中间件：TraceID -> Auth (选择性) -> CORS
	// 注意：中间件按逆序包装，所以执行顺序是 CORS -> Auth -> TraceID -> Handler
	httpHandler := middleware.TraceMiddleware(mux) // 逻辑处理
	httpHandler = withSelectiveAuth(httpHandler)   // 选择性认证中间件
	httpHandler = withCORS(httpHandler)            // 逻辑处理

	// 启动 HTTP 服务器
	logger.Info("Gateway started", zap.String("address", config.GlobalConfig.Server.Address))    // 逻辑处理
	if err := http.ListenAndServe(config.GlobalConfig.Server.Address, httpHandler); err != nil { // 条件判断
		logger.Fatal("Gateway stopped with error", zap.Error(err)) // 逻辑处理
	} // 代码块结束
} // 代码块结束

// withCORS 跨域处理中间件
// 功能说明：
// 1. 检查请求来源是否在允许列表中（localhost/127.0.0.1）
// 2. 设置CORS响应头，允许跨域请求
// 3. 处理OPTIONS预检请求
// 参数:
//   - next: 下一个HTTP处理器
//
// 返回: 包装后的HTTP处理器
func withCORS(next http.Handler) http.Handler { // 函数定义/HTTP处理入口
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // 返回结果/结束处理
		origin := r.Header.Get("Origin")   // 逻辑处理
		allowed := isAllowedOrigin(origin) // 逻辑处理

		// 设置CORS响应头
		if allowed { // 条件判断
			w.Header().Set("Access-Control-Allow-Origin", origin)                           // 设置响应头
			w.Header().Set("Vary", "Origin")                                                // 设置响应头
			w.Header().Set("Access-Control-Allow-Credentials", "true")                      // 设置响应头
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type") // 设置响应头
		} // 代码块结束

		// 处理OPTIONS预检请求
		if r.Method == http.MethodOptions { // 条件判断
			if allowed { // 条件判断
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS") // 设置响应头
				reqHeaders := r.Header.Get("Access-Control-Request-Headers")                             // 逻辑处理
				if strings.TrimSpace(reqHeaders) == "" {                                                 // 条件判断
					reqHeaders = "Content-Type, Authorization" // 逻辑处理
				} // 代码块结束
				w.Header().Set("Access-Control-Allow-Headers", reqHeaders) // 设置响应头
				w.Header().Set("Access-Control-Max-Age", "600")            // 预检结果缓存10分钟
			} // 代码块结束
			w.WriteHeader(http.StatusNoContent) // 写入HTTP状态码
			return                              // 返回结果/结束处理
		} // 代码块结束

		next.ServeHTTP(w, r) // 逻辑处理
	}) // 逻辑处理
} // 代码块结束

// withSelectiveAuth 选择性认证中间件
// 对公共接口（登录、注册、健康检查等）跳过认证
// 对其他接口应用 AuthMiddleware 进行 JWT 验证
// 参数:
//   - next: 下一个HTTP处理器
//
// 返回: 包装后的HTTP处理器
func withSelectiveAuth(next http.Handler) http.Handler { // 函数定义/HTTP处理入口
	// 公共路径列表（不需要登录认证）
	publicPaths := []string{
		"/api/v1/user/login",    // 登录
		"/api/v1/user/register", // 注册
		"/api/v1/user/logout",   // 退出
		"/health",               // 健康检查
		"/ws",                   // WebSocket（自己处理token验证）
		"/api/stats/online",     // 在线状态统计
		"/api/shift/list",       // 班次列表（用于请假表单下拉）
	}

	// 用 AuthMiddleware 包装
	authHandler := middleware.AuthMiddleware(next) // 逻辑处理

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // 返回结果/结束处理
		path := r.URL.Path // 逻辑处理

		// 检查是否为公共路径
		for _, p := range publicPaths { // 循环处理
			if path == p { // 条件判断
				// 公共路径，跳过认证，直接传递给下一个处理器
				next.ServeHTTP(w, r) // 逻辑处理
				return               // 返回结果/结束处理
			} // 代码块结束
		} // 代码块结束

		// 非公共路径，应用认证中间件
		authHandler.ServeHTTP(w, r) // 逻辑处理
	}) // 逻辑处理
} // 代码块结束

// isAllowedOrigin 检查请求来源是否在允许的白名单中
// 目前仅允许本地开发环境（localhost/127.0.0.1）
// 生产环境应根据实际域名配置
// 参数:
//   - origin: HTTP请求的Origin头
//
// 返回: true=允许跨域, false=拒绝
func isAllowedOrigin(origin string) bool { // 函数定义/HTTP处理入口
	origin = strings.TrimSpace(origin) // 逻辑处理
	if origin == "" {                  // 条件判断
		return false // 返回结果/结束处理
	} // 代码块结束
	u, err := url.Parse(origin) // 调用并接收错误
	if err != nil {             // 条件判断
		return false // 返回结果/结束处理
	} // 代码块结束
	if u.Scheme != "http" && u.Scheme != "https" { // 条件判断
		return false // 返回结果/结束处理
	} // 代码块结束
	host := u.Hostname()                              // 逻辑处理
	return host == "localhost" || host == "127.0.0.1" // 返回结果/结束处理
} // 代码块结束

// mustResolveConfigPath 解析配置文件路径
// 按顺序尝试多个候选路径，返回第一个存在的文件路径
// 支持从当前目录或项目根目录查找
// 参数:
//   - candidates: 候选配置文件路径列表
//
// 返回: 找到的配置文件路径，如果都不存在则返回第一个候选路径
func mustResolveConfigPath(candidates ...string) string { // 函数定义/HTTP处理入口
	// 首先尝试候选路径
	for _, p := range candidates { // 循环处理
		if fileExists(p) { // 条件判断
			return p // 返回结果/结束处理
		} // 代码块结束
	} // 代码块结束

	// 如果候选路径都不存在，尝试从项目根目录查找
	root, ok := findProjectRoot() // 逻辑处理
	if ok {                       // 条件判断
		for _, p := range candidates { // 循环处理
			if fileExists(filepath.Join(root, p)) { // 条件判断
				return filepath.Join(root, p) // 返回结果/结束处理
			} // 代码块结束
		} // 代码块结束
	} // 代码块结束

	return candidates[0] // 返回结果/结束处理
} // 代码块结束

// fileExists 检查文件是否存在
// 参数:
//   - path: 文件路径
//
// 返回: true=文件存在, false=文件不存在或路径为空
func fileExists(path string) bool { // 函数定义/HTTP处理入口
	if path == "" { // 条件判断
		return false // 返回结果/结束处理
	} // 代码块结束
	_, err := os.Stat(path) // 调用并接收错误
	return err == nil       // 返回结果/结束处理
} // 代码块结束

// findProjectRoot 查找项目根目录
// 从当前工作目录向上遍历，找到包含go.mod的目录即为项目根目录
// 返回:
//   - string: 项目根目录路径
//   - bool: true=找到, false=未找到
func findProjectRoot() (string, bool) { // 函数定义/HTTP处理入口
	wd, err := os.Getwd() // 调用并接收错误
	if err != nil {       // 条件判断
		return "", false // 返回结果/结束处理
	} // 代码块结束
	dir := wd // 逻辑处理
	for {     // 循环处理
		// 找到go.mod文件，说明是项目根目录
		if fileExists(filepath.Join(dir, "go.mod")) { // 条件判断
			return dir, true // 返回结果/结束处理
		} // 代码块结束
		// 向上遍历
		parent := filepath.Dir(dir) // 逻辑处理
		if parent == dir {          // 条件判断
			return "", false // 返回结果/结束处理
		} // 代码块结束
		dir = parent // 逻辑处理
	} // 代码块结束
} // 代码块结束

// initTracing 初始化链路追踪
func initTracing() { // 函数定义/HTTP处理入口
	cfg := config.GlobalConfig.Trace // 逻辑处理
	if !cfg.Enabled {                // 条件判断
		logger.Info("Tracing is disabled") // 逻辑处理
		return                             // 返回结果/结束处理
	} // 代码块结束

	// 配置 Jaeger Exporter
	jaegerCfg := &trace.JaegerConfig{ // 逻辑处理
		Enabled:       cfg.JaegerEnabled, // 逻辑处理
		ServiceName:   cfg.ServiceName,   // 逻辑处理
		Endpoint:      cfg.Endpoint,      // 逻辑处理
		SampleRate:    cfg.SampleRate,    // 逻辑处理
		BatchSize:     cfg.BatchSize,     // 逻辑处理
		FlushInterval: cfg.FlushInterval, // 逻辑处理
	} // 代码块结束

	// 设置默认值
	if jaegerCfg.ServiceName == "" { // 条件判断
		jaegerCfg.ServiceName = "gateway" // 逻辑处理
	} // 代码块结束
	if jaegerCfg.Endpoint == "" { // 条件判断
		jaegerCfg.Endpoint = "http://jaeger:4318/v1/traces"
	} // 代码块结束
	if jaegerCfg.SampleRate <= 0 { // 条件判断
		jaegerCfg.SampleRate = 1.0 // Gateway 默认全量采样
	} // 代码块结束

	trace.InitJaeger(jaegerCfg)        // 逻辑处理
	logger.Info("Tracing initialized", // 逻辑处理
		zap.String("service", jaegerCfg.ServiceName),     // 逻辑处理
		zap.Bool("jaeger_enabled", jaegerCfg.Enabled),    // 逻辑处理
		zap.Float64("sample_rate", jaegerCfg.SampleRate), // 逻辑处理
	) // 导入依赖列表结束/或参数列表结束
} // 代码块结束
