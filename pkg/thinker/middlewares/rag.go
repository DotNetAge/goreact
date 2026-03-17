package middlewares

import (
	"fmt"

	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/types"
)

// RAGRetriever RAG 检索器接口
type RAGRetriever interface {
	// Retrieve 检索相关文档
	Retrieve(query string, topK int) ([]Document, error)
}

// Document 文档
type Document struct {
	ID       string                 // 文档 ID
	Content  string                 // 文档内容
	Score    float64                // 相关性得分
	Metadata map[string]interface{} // 元数据
}

// RAGMiddleware RAG 增强中间件
// 在 Think 之前检索相关知识，增强输入
func RAGMiddleware(retriever RAGRetriever, topK int) thinker.ThinkMiddleware {
	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			// 检索相关文档
			docs, err := retriever.Retrieve(task, topK)
			if err != nil {
				// RAG 失败不影响主流程，记录错误继续
				ctx.Set("rag_error", err.Error())
			} else if len(docs) > 0 {
				// 将检索到的文档注入 context
				ctx.Set("rag_documents", docs)

				// 构建增强的任务描述
				enhancedTask := buildEnhancedTask(task, docs)
				task = enhancedTask

				// 记录 RAG 信息
				ctx.Set("rag_enhanced", true)
				ctx.Set("rag_doc_count", len(docs))
			}

			return next(task, ctx)
		}
	}
}

// buildEnhancedTask 构建增强的任务描述
func buildEnhancedTask(task string, docs []Document) string {
	if len(docs) == 0 {
		return task
	}

	enhanced := fmt.Sprintf("%s\n\nRelevant context from knowledge base:\n", task)
	for i, doc := range docs {
		enhanced += fmt.Sprintf("\n[Document %d] (relevance: %.2f)\n%s\n", i+1, doc.Score, doc.Content)
	}

	return enhanced
}
