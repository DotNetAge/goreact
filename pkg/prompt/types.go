package prompt

// Prompt 表示完整的提示词
type Prompt struct {
	System string
	User   string
}

// String 返回完整的提示词字符串
func (p *Prompt) String() string {
	if p.System == "" {
		return p.User
	}
	return p.System + "\n\n" + p.User
}

// TokenCounter Token 计数器接口
type TokenCounter interface {
	Count(text string) int
}

// Tokens 返回估算的 token 数
func (p *Prompt) Tokens(counter TokenCounter) int {
	if counter == nil {
		return len(p.String()) / 4 // 简单估算
	}
	return counter.Count(p.String())
}
