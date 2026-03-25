// Package builtin 提供一组轻量、实用而强大的内置工具集
//
// 设计理念（参考 Claude Code）：
// - 职责单一：每个工具只做一件事，做到极致
// - 组合强大：通过工具组合实现复杂功能
// - 覆盖全面：文件操作、搜索、执行三大核心场景
//
// 工具分类：
// Tier 1 - 文件操作：Read, Write, Edit
// Tier 2 - 搜索浏览：Glob, Grep, LS
// Tier 3 - 执行扩展：Bash, Calculator, DateTime, Cron
//
// 已移除：
// - Email → 移至独立插件包 (goreact-plugins/email)
// - Docker, Git → 低频使用，移至独立插件包
// - HTTP, Curl → 使用 bash curl 替代
// - Filesystem → 拆分为 Read/Write/Edit
// - Echo, Port → 无价值，直接删除
package builtin
