# Calculator Example

演示如何使用 Calculator 工具进行数学计算。

## 功能

- 使用 Mock LLM 模拟推理过程
- Calculator 工具支持四则运算
- 展示 ReAct 循环的完整流程

## 运行

```bash
go run main.go
```

## 代码说明

这个示例展示了：

1. **工具注册**：注册 Calculator 工具
2. **任务执行**：执行数学计算任务
3. **结果追踪**：查看完整的执行轨迹

Calculator 工具支持的操作：
- `add` - 加法
- `subtract` - 减法
- `multiply` - 乘法
- `divide` - 除法

## 示例输出

```
Task: Calculate 15 * 23 + 7
Success: true
Output: 352
```

## 下一步

- 查看 [ollama](../ollama/) 示例了解真实 LLM 集成
- 查看 [with_cache](../with_cache/) 示例了解缓存优化
