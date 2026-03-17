# RAG 集成指南

> 将 GoRAG 作为 Tool 集成到 GoReact Agent 中

## 📖 概述

GoReact 提供了与 RAG 系统集成的标准接口，可以将任何符合 `Retriever` 或 `RAG` 接口的 RAG 引擎（如 GoRAG）作为 Agent 的工具使用。

---

## 🔗 接口定义

### Retriever 接口

```go
// pkg/rag/rag.go
type Retriever interface {
    // Retrieve 检索与查询相关的文档
    Retrieve(query string, topK int) ([]Document, error)
}
```

### RAG 接口

```go
type RAG interface {
    // Generate 生成增强响应
    Generate(query string) (string, error)
    
    // AddDocument 添加文档到 RAG 系统
    AddDocument(doc Document) error
}
```

### Document 接口

```go
type Document interface {
    ID() string
    Content() string
    Metadata() map[string]interface{}
}
```

---

## 🚀 快速开始

### 1. 将 GoRAG 封装为 Tool

```go
package main

import (
    "context"
    
    "github.com/DotNetAge/goreact/pkg/rag"
    "github.com/DotNetAge/gorag/infra/searcher/native"
)

// RAGToolAdapter 将 GoRAG Searcher 适配为 GoReact Tool
type RAGToolAdapter struct {
    searcher *native.Searcher
}

func NewRAGToolAdapter(searcher *native.Searcher) *RAGToolAdapter {
    return &RAGToolAdapter{searcher: searcher}
}

// 实现 rag.Retriever 接口
func (a *RAGToolAdapter) Retrieve(query string, topK int) ([]rag.Document, error) {
    result, err := a.searcher.Search(context.Background(), query)
    if err != nil {
        return nil, err
    }
    
    // 转换为 rag.Document
    docs := make([]rag.Document, len(result.Chunks))
    for i, chunk := range result.Chunks {
        docs[i] = &ChunkDocument{chunk: chunk}
    }
    return docs, nil
}

// ChunkDocument 适配器
type ChunkDocument struct {
    chunk *entity.Chunk
}

func (d *ChunkDocument) ID() string {
    return d.chunk.ID
}

func (d *ChunkDocument) Content() string {
    return d.chunk.Content
}

func (d *ChunkDocument) Metadata() map[string]interface{} {
    return d.chunk.Metadata
}
```

### 2. 在 Agent 中使用 RAG Tool

```go
package main

import (
    "context"
    
    "github.com/DotNetAge/goreact/pkg/agent"
    "github.com/DotNetAge/goreact/pkg/core/thinker/presets"
    "github.com/DotNetAge/goreact/pkg/tool"
)

func main() {
    // 1. 创建 GoRAG Searcher
    searcher := native.NewSearcher(...)
    
    // 2. 创建 RAG Tool
    ragTool := NewRAGToolAdapter(searcher)
    
    // 3. 创建 Agent
    ag := agent.NewAgent(
        agent.WithName("ResearchAssistant"),
        agent.WithThinker(presets.NewReActThinker(llm)),
        agent.WithTools([]tool.Tool{
            tool.FromRetriever(ragTool), // ← 使用 RAG Tool
            tool.Calculator(),           // 计算器
            tool.WebSearch(),            // 网络搜索
        }),
    )
    
    // 4. 执行任务
    response, err := ag.Think(context.Background(),
        "分析特斯拉 2024 年 Q2 财报的关键数据")
    
    fmt.Println(response)
}
```

---

## 💡 高级用法

### 1. RAG + 多工具协作

```go
// 创建多个工具
tools := []tool.Tool{
    tool.FromRetriever(ragTool),      // RAG 检索专业知识
    tool.Calculator(),                // 计算财务比率
    tool.CodeExecutor(),              // 运行 Python 分析代码
    tool.WebSearch(),                 // 获取最新新闻
}

// Agent 会自动决定何时使用哪个工具
/*
Thought: 我需要先查找特斯拉 2024 年 Q2 财报的数据
Action: RAG_Tool
Observation: 找到财报数据...

Thought: 现在我需要计算同比增长率
Action: Calculator
Observation: 增长率 = 15.3%

Thought: 我还需要最新的股价信息
Action: WebSearch
Observation: 当前股价 $245...

Thought: 现在我有足够的信息来生成完整分析报告
Action: Finish
*/
```

### 2. 带置信度的 RAG Tool

```go
type ConfidenceRAGTool struct {
    searcher *native.Searcher
}

func (t *ConfidenceRAGTool) Retrieve(query string, topK int) ([]Document, error) {
    result, err := t.searcher.Search(ctx, query)
    if err != nil {
        return nil, err
    }
    
    // 附加 Self-RAG 置信度评分
    docs := make([]Document, 0)
    for _, chunk := range result.Chunks {
        if chunk.Score >= 0.8 { // 只返回高置信度结果
            docs = append(docs, chunk)
        }
    }
    
    return docs, nil
}
```

### 3. 级联 RAG（多级检索）

```go
type CascadeRAGTool struct {
    primary   *hybrid.Searcher    // 混合检索（快）
    secondary *native.Searcher    // 深度检索（慢）
}

func (t *CascadeRAGTool) Retrieve(query string, topK int) ([]Document, error) {
    // 先尝试快速检索
    result, _ := t.primary.Search(ctx, query)
    
    if len(result.Chunks) < topK {
        // 不够，使用深度检索补充
        deepResult, _ := t.secondary.Search(ctx, query)
        result.Chunks = append(result.Chunks, deepResult.Chunks...)
    }
    
    return convertToDocuments(result.Chunks[:topK]), nil
}
```

---

## 📊 性能优化建议

### 1. 缓存 RAG 结果

```go
type CachedRAGTool struct {
    searcher *native.Searcher
    cache    *cache.MemoryCache
}

func (t *CachedRAGTool) Retrieve(query string, topK int) ([]Document, error) {
    // 先查缓存
    if cached, ok := t.cache.Get(query); ok {
        return cached, nil
    }
    
    // 未命中，检索并缓存
    result, err := t.searcher.Search(ctx, query)
    if err != nil {
        return nil, err
    }
    
    docs := convertToDocuments(result.Chunks)
    t.cache.Set(query, docs, 5*time.Minute)
    return docs, nil
}
```

### 2. 批量检索优化

```go
// Agent 可能需要多次检索相似问题
// 可以批量处理减少向量数据库调用

type BatchRAGTool struct {
    searcher *native.Searcher
    batch    []string
}

func (t *BatchRAGTool) Retrieve(query string, topK int) ([]Document, error) {
    t.batch = append(t.batch, query)
    
    // 等待 100ms 收集更多查询
    time.Sleep(100 * time.Millisecond)
    
    // 批量检索
    allQueries := t.batch
    t.batch = nil
    
    results, err := t.searcher.BatchSearch(ctx, allQueries, topK)
    // ... 处理结果
    
    return results[query], nil
}
```

---

## 🔍 调试和监控

### 1. 日志记录

```go
type LoggingRAGTool struct {
    searcher *native.Searcher
    logger   log.Logger
}

func (t *LoggingRAGTool) Retrieve(query string, topK int) ([]Document, error) {
    t.logger.Info("RAG retrieval", 
        "query", query, 
        "topK", topK)
    
    start := time.Now()
    result, err := t.searcher.Search(ctx, query)
    duration := time.Since(start)
    
    t.logger.Info("RAG retrieval completed",
        "duration_ms", duration.Milliseconds(),
        "chunks_count", len(result.Chunks))
    
    return convertToDocuments(result.Chunks), err
}
```

### 2. 指标监控

```go
type MetricsRAGTool struct {
    searcher *native.Searcher
    metrics  *metrics.Collector
}

func (t *MetricsRAGTool) Retrieve(query string, topK int) ([]Document, error) {
    t.metrics.Inc("rag_retrievals_total")
    
    result, err := t.searcher.Search(ctx, query)
    if err != nil {
        t.metrics.Inc("rag_errors_total")
        return nil, err
    }
    
    t.metrics.Observe("rag_latency_seconds", duration.Seconds())
    t.metrics.Observe("rag_chunks_count", float64(len(result.Chunks)))
    
    return convertToDocuments(result.Chunks), nil
}
```

---

## 📚 完整示例

### 智能研究助手

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/DotNetAge/goreact/pkg/agent"
    "github.com/DotNetAge/goreact/pkg/core/thinker/presets"
    "github.com/DotNetAge/goreact/pkg/tool"
    "github.com/DotNetAge/goreact/pkg/memory"
    
    "github.com/DotNetAge/gorag/infra/searcher/hybrid"
    "github.com/DotNetAge/gorag/infra/vectorstore/faiss"
)

func main() {
    // 1. 初始化 GoRAG
    vectorDB := faiss.NewStore("dim=768")
    searcher := hybrid.NewSearcher(
        hybrid.WithLLM(llm),
        hybrid.WithEmbedder(embedder),
        hybrid.WithVectorStores(vectorDB, sparseDB),
        hybrid.WithFusion(true),
        hybrid.WithRerank(true),
        hybrid.WithSelfRAG(true),
    )
    
    // 2. 创建 RAG Tool
    ragTool := NewRAGToolAdapter(searcher)
    
    // 3. 创建 Agent
    ag := agent.NewAgent(
        agent.WithName("AI Research Assistant"),
        agent.WithThinker(presets.NewReActThinker(llm,
            presets.WithMaxIterations(5),
            presets.WithTemperature(0.7),
        )),
        agent.WithMemory(memory.NewConversationBuffer(10)),
        agent.WithTools([]tool.Tool{
            tool.FromRetriever(ragTool),
            tool.Calculator(),
            tool.WebSearch(),
            tool.CodeExecutor(),
        }),
    )
    
    // 4. 对话循环
    ctx := context.Background()
    for {
        fmt.Print("User: ")
        var input string
        fmt.Scanln(&input)
        
        if input == "exit" {
            break
        }
        
        response, err := ag.Think(ctx, input)
        if err != nil {
            log.Printf("Error: %v", err)
            continue
        }
        
        fmt.Printf("Assistant: %s\n", response)
    }
}
```

---

## 🎯 最佳实践总结

1. **优先使用 GoRAG 独立模式** - 简单场景不需要引入 Agent
2. **清晰的责任边界** - RAG 负责检索，Agent 负责决策
3. **接口兼容性** - 确保 GoRAG 实现 GoReact 的 RAG 接口
4. **性能优化** - 使用缓存、批处理等策略
5. **可观测性** - 添加日志和指标监控

---

## 📖 相关资源

- [GoRAG 联合使用指南](../gorag/docs/GO_REACT_INTEGRATION_GUIDE.md)
- [GoReact Agent 文档](./pkg/agent/README.md)
- [Tool 开发指南](./docs/TOOL_DEVELOPMENT_GUIDE.md)
- [ReAct Thinker 详解](./pkg/core/thinker/presets/react_thinker.go)
