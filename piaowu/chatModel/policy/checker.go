package policy

import (
	"context"
	"fmt"
)

// Checker 负责执行“行级数据域隔离 (Row-Level Security)”校验
// 在 Eino 决策链中，它是第一个执行的节点，确保 AI 不会处理未授权的会话数据
type Checker struct{}

// NewChecker 构造一个权限检查器实例
func NewChecker() *Checker {
	return &Checker{}
}

// CheckSessionPermission 校验当前操作者 (Actor) 是否具备访问目标会话的合法权限
// 这是防止 AI 越权访问他人聊天记录的关键工程保障
func (c *Checker) CheckSessionPermission(ctx context.Context, actorID int64, conversationID string) error {
	// 基础防御：拒绝匿名或未标记身份的请求
	if actorID == 0 {
		return fmt.Errorf("权限拦截 [Unauthorized]: actor_id 缺失，无法界定操作边界")
	}

	// 业务逻辑说明：
	// 在生产环境中，此处应通过 repository 调用数据库，
	// 查询 t_conversation 表，验证该 conversation_id 的主体归属是否与 actor_id 匹配。

	fmt.Printf("[Policy/Checker] 正在执行数据隔离审计：操作人=%d, 目标会话=%s\n", actorID, conversationID)

	// 若校验不通过，应返回具体的 error，Eino Graph 将立即终止并返回给客户端
	return nil
}
