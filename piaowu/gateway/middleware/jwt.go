// Package middleware 提供网关侧的认证与鉴权相关能力。
//
// 本文件聚焦 JWT 的“配置 + 生成 + 解析”：
// - 启动阶段由 main 读取配置并调用 SetJWTSecret / SetJWTExpireTime 注入运行时配置
// - 登录成功后由 GenerateToken 生成 Token 返回前端
// - 鉴权阶段由 ParseToken 解析 Token 并返回自定义 Claims
//
// 约定：
// - Token 采用 HS256 签名
// - 解析时允许传入完整 Authorization 值（"Bearer xxx"）或纯 token 字符串
package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWT 运行时配置（密钥 + 过期时间）。
//
// 说明：
// - 这里用包级变量便于中间件与业务代码共享配置
// - 正式环境务必从配置文件/环境变量注入强随机密钥，避免默认值泄漏导致伪造 Token
var (
	jwtSecret     = []byte("your-secret-key-change-in-production")
	jwtExpireTime = 24 * time.Hour
)

// Claims 是网关侧使用的 JWT 声明（Payload）。
//
// 字段说明：
// - UserID：业务侧用户/客服的主键 ID
// - UserName：登录账号（用于展示或审计）
// - RoleCode：角色编码（admin/customer_service），用于鉴权与路由控制
// - RegisteredClaims：标准 JWT 声明，包含 exp/iat/nbf 等时间字段
type Claims struct {
	UserID   int64  `json:"user_id"`
	UserName string `json:"user_name"`
	RoleCode string `json:"role_code"`
	jwt.RegisteredClaims
}

// SetJWTSecret 设置JWT密钥
// 用于在服务启动时从配置文件加载密钥
// 参数:
//   - secret: 密钥字符串，空字符串不更新
func SetJWTSecret(secret string) {
	if secret != "" {
		jwtSecret = []byte(secret)
	}
}

// SetJWTExpireTime 设置 Token 过期时间（单位：小时）。
//
// 说明：
// - hours <= 0 时忽略，保留默认值
// - 过期时间会影响 GenerateToken 生成的 exp 字段
func SetJWTExpireTime(hours int) {
	if hours > 0 {
		jwtExpireTime = time.Duration(hours) * time.Hour
	}
}

// GenerateToken 生成JWT Token
// 参数: 用户ID、用户名、角色编码
// 返回: Token字符串和错误信息
func GenerateToken(userID int64, userName, roleCode string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		UserName: userName,
		RoleCode: roleCode,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(jwtExpireTime)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析JWT Token
// 参数: Token字符串
// 返回: Claims声明和错误信息
func ParseToken(tokenString string) (*Claims, error) {
	// 允许直接传 token，或传 "Bearer <token>" 形式的 Authorization 值。
	// 这里只做前缀裁剪与空白清理，不做更复杂的头部解析。
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	tokenString = strings.TrimSpace(tokenString)

	if tokenString == "" {
		return nil, errors.New("token is empty")
	}

	// 解析并校验签名与标准 Claims（exp/nbf/iat）。
	// keyFunc 返回签名密钥；如需对 alg 做更严格的约束，可在 keyFunc 内检查 token.Method。
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GetUserIDFromToken 从Token中获取用户ID
//
// 适用场景：
// - 某些非 HTTP 中间件场景只需要 UserID（例如 WebSocket 握手鉴权）
func GetUserIDFromToken(tokenString string) (int64, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

// GetRoleCodeFromToken 从Token中获取角色编码
//
// 说明：
// - RoleCode 由登录时写入 Token，用于后续鉴权（管理员/客服等）
func GetRoleCodeFromToken(tokenString string) (string, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.RoleCode, nil
}
