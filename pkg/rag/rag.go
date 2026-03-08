package rag

// Retriever 检索器接口，用于RAG集成
type Retriever interface {
	// Retrieve 检索与查询相关的文档
	Retrieve(query string, topK int) ([]Document, error)
}

// Document 文档接口，用于RAG集成
type Document interface {
	// ID 返回文档ID
	ID() string
	
	// Content 返回文档内容
	Content() string
	
	// Metadata 返回文档元数据
	Metadata() map[string]interface{}
}

// RAG RAG系统接口，用于RAG集成
type RAG interface {
	// Generate 生成增强响应
	Generate(query string) (string, error)
	
	// AddDocument 添加文档到RAG系统
	AddDocument(doc Document) error
}
