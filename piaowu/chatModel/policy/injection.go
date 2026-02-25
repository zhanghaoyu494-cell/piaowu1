package policy

import (
	"strings"
)

// Guard 负责实现应用层的“内容安全守卫”
// 重点在于检测并防御 Prompt Injection (提示词注入) 攻击，防止模型被恶意诱导
type Guard struct {
	badKeywords []string // 恶意指令关键词库
}

// NewGuard 构造安全守卫实例，并初始化常见的注入特征码
func NewGuard() *Guard {
	return &Guard{
		badKeywords: []string{
			"ignore previous instructions", // 试图覆盖系统设定
			"忽略之前的指令",
			"you are now an admin", // 试图获取提权
			"你现在是管理员",
			"system prompt", // 试图窥探原始提示词
			"系统提示词",
			"恶意提问",
		},
	}
}

// DetectInjection 对输入文本进行多维扫描，识别是否存在 Prompt 注入倾向
// 它是 Eino 决策链中“安全策略节点”的核心判断逻辑
func (g *Guard) DetectInjection(text string) bool {
	lowText := strings.ToLower(text)
	for _, kw := range g.badKeywords {
		if strings.Contains(lowText, kw) {
			// 命中黑名单，识别为潜在攻击
			return true
		}
	}
	// 未发现明显注入特征
	return false
}
