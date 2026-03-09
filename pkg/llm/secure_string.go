package llm

// SecureString 安全字符串，防止敏感信息泄露
type SecureString struct {
	value string
}

// NewSecureString 创建安全字符串
func NewSecureString(value string) SecureString {
	return SecureString{value: value}
}

// String 实现 Stringer 接口，返回脱敏字符串
func (s SecureString) String() string {
	if s.value == "" {
		return ""
	}
	// 只显示前4个字符和后4个字符
	if len(s.value) <= 8 {
		return "***REDACTED***"
	}
	return s.value[:4] + "..." + s.value[len(s.value)-4:]
}

// Value 获取实际值（仅在需要时调用）
func (s SecureString) Value() string {
	return s.value
}

// IsEmpty 检查是否为空
func (s SecureString) IsEmpty() bool {
	return s.value == ""
}
