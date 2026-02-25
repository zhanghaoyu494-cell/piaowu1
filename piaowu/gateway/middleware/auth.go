// Package middleware 提供 Gateway 网关的 HTTP 中间件能力。
//
// 本文件聚焦“HTTP 鉴权与权限控制”：
// - AuthMiddleware / AuthMiddlewareFunc：解析 Authorization 头，校验 JWT，并把用户信息写入 request context
// - CheckAdminPermission：基于 context 中的 RoleCode 做管理员权限校验
//
// 返回策略（与很多 REST 习惯不同）：
// - 鉴权失败/权限不足时，HTTP 状态仍返回 200（StatusOK）
// - 业务错误码通过 JSON 的 code 字段表达（401/403）
// 该约定需要前端/调用方按 code 字段判断是否需要跳转登录或提示无权限。
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// contextKey 用于避免 context.WithValue 的 key 与其他包发生冲突。
// 通过自定义类型替代 string，可减少误用/覆盖的风险。
type contextKey string

const (
	// ContextKeyUserID 存储用户 ID（int64）。
	ContextKeyUserID contextKey = "user_id"
	// ContextKeyUserName 存储用户名（string）。
	ContextKeyUserName contextKey = "user_name"
	// ContextKeyRoleCode 存储角色编码（string）。
	ContextKeyRoleCode contextKey = "role_code"
)

// 角色编码常量（需与后端/数据模型保持一致）。
// 说明：IsCustomerService 将 admin 视作具备客服权限的超集角色。
const (
	RoleAdmin           = "admin"
	RoleCustomerService = "customer_service"
)

// Response 统一响应结构
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// respondJSON 返回JSON响应
//
// status 用于 HTTP 状态码（本项目多数情况下使用 200），data 为任意可 JSON 序列化对象。
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// AuthMiddleware 登录校验中间件
// 验证请求头中的Token，将用户信息存入上下文
// 返回包装后的Handler
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 约定：Token 放在 Authorization 头中，格式为：
		// - "Bearer <token>"（推荐）
		// - "<token>"（兼容）
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondJSON(w, http.StatusOK, Response{Code: 401, Msg: "请先登录"})
			return
		}

		// ParseToken 内部会处理 Bearer 前缀裁剪，并验证签名与过期时间等。
		claims, err := ParseToken(authHeader)
		if err != nil {
			respondJSON(w, http.StatusOK, Response{Code: 401, Msg: "登录已过期，请重新登录"})
			return
		}

		// 将用户信息写入 context，供后续 Handler/中间件读取。
		// 注意：context 存储的是“认证结果”，不能替代后端的最终权限校验。
		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, ContextKeyUserName, claims.UserName)
		ctx = context.WithValue(ctx, ContextKeyRoleCode, claims.RoleCode)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AuthMiddlewareFunc 登录校验中间件（函数版本）
// 用于包装 http.HandlerFunc
func AuthMiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondJSON(w, http.StatusOK, Response{Code: 401, Msg: "请先登录"})
			return
		}

		claims, err := ParseToken(authHeader)
		if err != nil {
			respondJSON(w, http.StatusOK, Response{Code: 401, Msg: "登录已过期，请重新登录"})
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, ContextKeyUserName, claims.UserName)
		ctx = context.WithValue(ctx, ContextKeyRoleCode, claims.RoleCode)

		next(w, r.WithContext(ctx))
	}
}

// CheckAdminPermission 管理员权限校验中间件
// 仅允许管理员访问，客服访问返回403
func CheckAdminPermission(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 依赖 AuthMiddleware 先写入 context；若未经过鉴权中间件，这里会视为未登录。
		roleCode, ok := r.Context().Value(ContextKeyRoleCode).(string)
		if !ok || roleCode == "" {
			respondJSON(w, http.StatusOK, Response{Code: 401, Msg: "请先登录"})
			return
		}

		if roleCode != RoleAdmin {
			respondJSON(w, http.StatusOK, Response{Code: 403, Msg: "权限不足，仅管理员可操作"})
			return
		}

		next(w, r)
	}
}

// GetUserIDFromContext 从上下文获取用户ID
//
// 返回 0 表示未登录或未注入（调用方可据此做兜底处理）。
func GetUserIDFromContext(ctx context.Context) int64 {
	if userID, ok := ctx.Value(ContextKeyUserID).(int64); ok {
		return userID
	}
	return 0
}

// GetUserNameFromContext 从上下文获取用户名
func GetUserNameFromContext(ctx context.Context) string {
	if userName, ok := ctx.Value(ContextKeyUserName).(string); ok {
		return userName
	}
	return ""
}

// GetRoleCodeFromContext 从上下文获取角色编码
func GetRoleCodeFromContext(ctx context.Context) string {
	if roleCode, ok := ctx.Value(ContextKeyRoleCode).(string); ok {
		return roleCode
	}
	return ""
}

// IsAdmin 判断当前用户是否为管理员
func IsAdmin(ctx context.Context) bool {
	return GetRoleCodeFromContext(ctx) == RoleAdmin
}

// IsCustomerService 判断当前用户是否为客服
//
// 说明：管理员默认拥有客服权限（可用于客服相关接口与后台管理接口）。
func IsCustomerService(ctx context.Context) bool {
	roleCode := GetRoleCodeFromContext(ctx)
	return roleCode == RoleCustomerService || roleCode == RoleAdmin
}

// extractBearerToken 从Authorization头中提取Token
//
// 兼容两种输入：
// - "Bearer <token>"
// - "<token>"
//
// 注意：本文件目前未直接使用该函数（ParseToken 已支持裁剪 "Bearer " 前缀），
// 仍保留在此以便其他模块（例如 WebSocket 握手）复用。
func extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		return parts[1]
	}
	return authHeader
}
