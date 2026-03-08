package memory

// MemoryManager 内存管理器接口
type MemoryManager interface {
	// Store 存储内存数据
	Store(sessionId string, key string, value interface{})
	// Retrieve 检索内存数据
	Retrieve(sessionId string, key string) interface{}
	// Compress 压缩内存数据
	Compress(sessionId string) error
	// Persist 持久化内存数据
	Persist(sessionId string) error
	// Load 加载内存数据
	Load(sessionId string) error
}
