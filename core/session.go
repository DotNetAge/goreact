package core

type SessionStore interface {
	Append(sessionID string, message Message) error      // 将消息插入到指定会话的尾部(System消息会自动插入到头部)
	Get(sessionID string) ([]*Message, error)            // 获取指定会话的所有消息
	CurrentContext(sessionID string) ([]*Message, error) // 获取适合当前上下文窗口的消息
	Delete(timestamp int64, sessionID string) error      // 删除指定时间戳的消息
}
