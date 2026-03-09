# Simple Example

最简单的 GoReAct 使用示例，使用 Mock LLM 进行快速测试。

## 功能

- 使用 Mock LLM（无需真实 LLM 服务）
- 注册 Echo 工具
- 执行简单任务

## 运行

```bash
go run main.go
```

## 代码说明

这个示例展示了 GoReAct 的最小化配置：

1. 创建 Engine（默认使用 Mock LLM）
2. 注册 Echo 工具
3. 执行任务并获取结果

适合用于：
- 快速了解 GoReAct 基本用法
- 测试框架功能
- 开发调试

## 下一步

- 查看 [calculator](../calculator/) 示例了解更复杂的工具使用
- 查看 [ollama](../ollama/) 示例了解真实 LLM 集成
