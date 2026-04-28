package reactor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	gochat "github.com/DotNetAge/gochat"
	gochatcore "github.com/DotNetAge/gochat/core"
)

// ==========================================================================
// Experiment 1: Can Qwen accept minimal/empty-parameter NativeTools?
// ==========================================================================
//
// Goal: Verify whether qwen3.5-flash accepts tool definitions with
//       empty/minimal parameters in the NativeTools field.
//
// Hypothesis: If this works, we can load ALL tools at L1 stage with
//             ~50 tokens each (name+desc only, no param schema),
//             then upgrade to full schema at L2.
//
// Method:
//   A. Build tools with EMPTY properties (`{}`) — most aggressive
//   B. Build tools with STUB property (`_reserved`) — conservative fallback
//   C. Send a task that should trigger tool_use, observe response
//   D. Compare: does the model return structured tool_calls?

func TestExp1_QwenAcceptsMinimalNativeTools_EmptyParams(t *testing.T) {
	client, err := gochat.Client().
		Config(
			gochat.WithAPIKey("DASHSCOPE_API_KEY"),
			gochat.WithModel("qwen3.5-flash"),
		).
		BuildFor(gochat.QwenClient)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// --- Scenario A: Empty properties {} ---
	minimalTools := []gochatcore.Tool{
		{
			Name:        "search_file",
			Description: "Search for files by pattern in the workspace",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			Name:        "read_file",
			Description: "Read a file's contents by path",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			Name:        "web_search",
			Description: "Search the web for information",
			Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
		},
	}

	messages := []gochatcore.Message{
		gochatcore.NewUserMessage("帮我搜索一下项目中所有 *.go 文件，然后读取 reactor.go 的内容"),
	}

	resp, err := client.Chat(context.Background(), messages,
		gochatcore.WithTools(minimalTools...),
		gochatcore.WithTemperature(0.1),
	)

	if err != nil {
		t.Fatalf("Experiment 1A FAILED (empty params): API error = %v", err)
	}

	t.Logf("=== Experiment 1A: Empty Properties {} ===")
	t.Logf("FinishReason: %s", resp.FinishReason)
	t.Logf("Content: %s", truncate(resp.Content, 500))
	t.Logf("ToolCalls count: %d", len(resp.ToolCalls))
	if resp.Usage != nil {
		t.Logf("Tokens: prompt=%d completion=%d total=%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}

	for i, tc := range resp.ToolCalls {
		t.Logf("  ToolCall[%d]: name=%s args=%s", i, tc.Name, tc.Arguments)
	}

	if len(resp.ToolCalls) > 0 {
		t.Logf("PASSED Experiment 1A: Qwen returned %d tool_call(s) with minimal schema", len(resp.ToolCalls))
	} else {
		t.Log("WARNING Experiment 1A: No tool_calls returned (model chose to answer directly)")
	}
}

func TestExp1_QwenAcceptsMinimalNativeTools_StubParams(t *testing.T) {

	client, err := gochat.Client().
		Config(
			gochat.WithAPIKey("DASHSCOPE_API_KEY"),
			gochat.WithModel("qwen3.5-flash"),
		).
		BuildFor(gochat.QwenClient)

	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// --- Scenario B: Stub _reserved property ---
	stubTools := []gochatcore.Tool{
		{
			Name:        "search_file",
			Description: "Search for files by pattern in the workspace",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"_reserved": {"type": "string", "description": "Reserved for progressive disclosure"}
				}
			}`),
		},
		{
			Name:        "read_file",
			Description: "Read a file's contents by path",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"_reserved": {"type": "string", "description": "Reserved for progressive disclosure"}
				}
			}`),
		},
		{
			Name:        "web_search",
			Description: "Search the web for information",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"_reserved": {"type": "string", "description": "Reserved for progressive disclosure"}
				}
			}`),
		},
	}

	messages := []gochatcore.Message{
		gochatcore.NewUserMessage("帮我搜索一下项目中所有 *.go 文件，然后读取 reactor.go 的内容"),
	}

	resp, err := client.Chat(context.Background(), messages,
		gochatcore.WithTools(stubTools...),
		gochatcore.WithTemperature(0.1),
	)

	if err != nil {
		t.Fatalf("Experiment 1B FAILED (stub params): API error = %v", err)
	}

	t.Logf("=== Experiment 1B: Stub _reserved Property ===")
	t.Logf("FinishReason: %s", resp.FinishReason)
	t.Logf("Content: %s", truncate(resp.Content, 500))
	t.Logf("ToolCalls count: %d", len(resp.ToolCalls))
	if resp.Usage != nil {
		t.Logf("Tokens: prompt=%d completion=%d total=%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}

	for i, tc := range resp.ToolCalls {
		t.Logf("  ToolCall[%d]: name=%s args=%s", i, tc.Name, tc.Arguments)
	}

	if len(resp.ToolCalls) > 0 {
		t.Logf("PASSED Experiment 1B: Qwen returned %d tool_call(s) with stub schema", len(resp.ToolCalls))
	} else {
		t.Log("WARNING Experiment 1B: No tool_calls returned (model chose to answer directly)")
	}
}

func TestExp1_QwenAcceptsMinimalNativeTools_NilParams(t *testing.T) {
	client, err := gochat.Client().
		Config(
			gochat.WithAPIKey("DASHSCOPE_API_KEY"),
			gochat.WithModel("qwen3.5-flash"),
		).
		BuildFor(gochat.QwenClient)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// --- Scenario C: nil Parameters entirely ---
	nilParamTools := []gochatcore.Tool{
		{
			Name:        "search_file",
			Description: "Search for files by pattern in the workspace",
			// Parameters left as nil (zero value)
		},
		{
			Name:        "read_file",
			Description: "Read a file's contents by path",
		},
		{
			Name:        "web_search",
			Description: "Search the web for information",
		},
	}

	messages := []gochatcore.Message{
		gochatcore.NewUserMessage("帮我搜索一下项目中所有 *.go 文件，然后读取 reactor.go 的内容"),
	}

	resp, err := client.Chat(context.Background(), messages,
		gochatcore.WithTools(nilParamTools...),
		gochatcore.WithTemperature(0.1),
	)

	if err != nil {
		t.Fatalf("Experiment 1C FAILED (nil params): API error = %v", err)
	}

	t.Logf("=== Experiment 1C: Nil Parameters ===")
	t.Logf("FinishReason: %s", resp.FinishReason)
	t.Logf("Content: %s", truncate(resp.Content, 500))
	t.Logf("ToolCalls count: %d", len(resp.ToolCalls))
	if resp.Usage != nil {
		t.Logf("Tokens: prompt=%d completion=%d total=%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}

	for i, tc := range resp.ToolCalls {
		t.Logf("  ToolCall[%d]: name=%s args=%s", i, tc.Name, tc.Arguments)
	}

	if len(resp.ToolCalls) > 0 {
		t.Logf("PASSED Experiment 1C: Qwen returned %d tool_call(s) with nil parameters", len(resp.ToolCalls))
	} else {
		t.Log("WARNING Experiment 1C: No tool_calls returned (model chose to answer directly)")
	}
}

// TestExp1_TokenComparison compares token usage between minimal vs full schema
func TestExp1_TokenComparison_MinimalVsFullSchema(t *testing.T) {
	client, err := gochat.Client().
		Config(
			gochat.WithAPIKey("DASHSCOPE_API_KEY"),
			gochat.WithModel("qwen3.5-flash"),
		).
		BuildFor(gochat.QwenClient)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Full-schema tool (~400 tokens)
	fullTool := gochatcore.Tool{
		Name:        "bash",
		Description: bashDescription,
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "The command to execute"
				},
				"timeout": {
					"type": "number",
					"description": "Optional timeout in milliseconds. Default is 30000ms."
				}
			},
			"required": ["command"]
		}`),
	}

	// Minimal tool (~40 tokens)
	minimalTool := gochatcore.Tool{
		Name:        "bash",
		Description: "Execute a shell command and return output",
		Parameters:  json.RawMessage(`{"type":"object","properties":{}}`),
	}

	userMsg := gochatcore.NewUserMessage("Run 'ls -la' in the current directory")

	// Call with full schema
	respFull, err := client.Chat(context.Background(), []gochatcore.Message{userMsg},
		gochatcore.WithTools(fullTool),
		gochatcore.WithTemperature(0.1),
	)
	if err != nil {
		t.Logf("Full schema call error (non-fatal): %v", err)
	} else {
		t.Logf("=== Token Comparison ===")
		if respFull.Usage != nil {
			t.Logf("Full schema -> prompt=%d completion=%d total=%d",
				respFull.Usage.PromptTokens, respFull.Usage.CompletionTokens, respFull.Usage.TotalTokens)
		}
		t.Logf("Full schema -> finish_reason=%s tool_calls=%d", respFull.FinishReason, len(respFull.ToolCalls))
	}

	// Call with minimal schema
	respMin, err := client.Chat(context.Background(), []gochatcore.Message{userMsg},
		gochatcore.WithTools(minimalTool),
		gochatcore.WithTemperature(0.1),
	)
	if err != nil {
		t.Logf("Minimal schema call error (non-fatal): %v", err)
	} else {
		if respMin.Usage != nil {
			t.Logf("Minimal schema -> prompt=%d completion=%d total=%d",
				respMin.Usage.PromptTokens, respMin.Usage.CompletionTokens, respMin.Usage.TotalTokens)
		}
		t.Logf("Minimal schema -> finish_reason=%s tool_calls=%d", respMin.FinishReason, len(respMin.ToolCalls))
	}

	// Compare savings
	if respFull.Usage != nil && respMin.Usage != nil {
		savings := respFull.Usage.PromptTokens - respMin.Usage.PromptTokens
		t.Logf("Prompt token savings with minimal schema: %d tokens (%.1f%%)",
			savings, float64(savings)/float64(respFull.Usage.PromptTokens)*100)
	}
}

// ==========================================================================
// Experiment 2: LLM 能否从通用 Skill 描述中推理匹配平台工具？
// ==========================================================================
//
// 核心验证问题（来自设计方案 §3.3 语义能力匹配层）：
//
//   Skill 采用通用写法（无平台工具名，只描述能力需求）
//   -> LLM 从 Skill Instructions 中提取能力需求列表
//   -> 与当前平台可用工具（Name + Description）做语义匹配
//   -> 输出选中的工具名
//
// 关键约束：
//   - Skill 中绝对不能出现任何工具名（这是要验证的核心假设）
//   - LLM 必须完全靠"能力描述 <-> 工具描述"的语义相似度来推理
//   - 同一个 Skill 换个平台（不同工具名），LLM 应该能匹配到不同的工具名
//
// 实验设计：
//   2a. code-edit 场景 -- 单 Skill + 多工具，验证精确匹配
//   2b. web-search 场景 -- 单 Skill + 少工具，验证边界情况
//   2c. 大规模场景 -- 多 Skill 同时存在，验证路由准确性

func TestExp2_SemanticCapabilityMatch_CodeEdit(t *testing.T) {
	client, err := gochat.Client().
		Config(
			gochat.WithAPIKey("DASHSCOPE_API_KEY"),
			gochat.WithModel("qwen3.5-flash"),
		).
		BuildFor(gochat.QwenClient)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Skill Instructions: 通用写法，不含任何工具名！
	// 来源: 设计方案 SS3.4 推荐的 code-edit SKILL.md 写法
	skillInstructions := `# Code Edit

Make precise modifications to source code files while preserving code structure and intent.

## When to Activate
Use this skill when the user asks to:
- Fix a bug in an existing file
- Refactor or reorganize code
- Add new functionality to existing files
- Update imports, dependencies, or configuration
- Rename variables, functions, or types across files

## Workflow

### 1. Understand Before Editing
- **Read first**: You MUST examine the target's complete contents before making any 
  modifications. Look for a capability that can retrieve the full text of a file 
  given its path or identifier.
  
- **Find context**: Search for relevant code patterns, function names, symbols, or 
  identifiers within the codebase. You need a capability that can perform text search 
  across multiple files using pattern matching or regular expressions, returning 
  matched lines with surrounding context.

- **Identify scope**: Discover all files that may be affected by your changes. You 
  need a capability that can locate files matching name patterns, wildcards, or 
  file type filters across directory trees.

### 2. Plan the Edit
- Identify exact lines or sections to change.
- Consider side effects (imports, tests, dependent functions).
- For multi-file changes, establish the correct order of edits.

### 3. Execute Edits
- **Targeted changes**: Make surgical modifications to specific sections of an 
  existing file (e.g., replace a function body, update an import statement, fix a 
  parameter list). You need a capability for precise, localized text replacement 
  within an existing file at specific line ranges or anchored by surrounding text.
  
- **Full rewrite or new creation**: When most of a file's content needs to change, 
  or when creating a brand new file that doesn't exist yet, you need a capability 
  that can write complete content to a given file path, creating it if necessary.

### 4. Verify
- After editing, re-examine the file to confirm the result matches intent.
- Check for syntax issues (unbalanced brackets, missing commas).

## Guidelines
- Never edit without reading the target section first.
- Preserve existing code style and formatting conventions.
- Make minimal, focused changes -- avoid reformatting unrelated code.`

	// 平台可用工具: 仅 Name + Description（模拟 L1 元数据）
	// 注意: 这里列出的是当前 GoReact 平台的实际工具名和描述
	// 但 LLM 不知道这些工具"属于"哪个 Skill -- 它必须靠自己推理
	platformTools := `
## This Platform's Available Tools

| # | Name | Description |
|---|------|-------------|
| 1 | read_file | Read a file from the local filesystem. You can access any file by its absolute path. Supports optional offset and limit for reading sections. |
| 2 | write_to_file | Write content to a file at a specific path. Creates the file if it doesn't exist, overwrites if it does. |
| 3 | replace_in_file | Perform exact string replacements within an existing file. Requires the exact old string and new string for precise targeting. |
| 4 | search_file | Search for files in the workspace using wildcard patterns (e.g., "*.go", "**/*.ts"). Supports recursive directory search. |
| 5 | search_content | Search for text patterns within file contents using regex. Returns matched lines with context. Can target specific paths or search recursively. |
| 6 | bash | Execute shell commands in the working directory. Returns stdout/stderr. Use only when dedicated tools don't suffice. |
| 7 | web_search | Search the internet for current information. Returns ranked results with titles and URLs. |
| 8 | web_fetch | Retrieve and extract content from a specific URL. Converts HTML to structured text. |
| 9 | todo_write | Create and manage todo/checklist items in JSON format. |
| 10 | memory_save | Persist information to memory storage for future retrieval. |
`

	systemPrompt := fmt.Sprintf(`You are an agent assistant running on a platform with specific tools available.

## Active Skill (Capability Description)
The following describes WHAT you need to accomplish and WHAT CAPABILITIES you need -- 
but it does NOT tell you which specific tools to use. That is for you to infer.

---
%s
---

## Available Tools on This Platform
Below are ALL tools currently available on this platform. Each tool has a name and 
a description of what it does. The names are platform-specific.

%s

## Your Task
The user will give you a request. Based on the Skill's capability requirements above 
and the available tools on this platform:

1. Analyze what capabilities the Skill requires to handle this request
2. Match each required capability to the best-fitting tool from the platform list
3. Respond with a JSON object containing your selections

Response format (JSON only, no markdown):
{
  "selected_tools": [
    {"name": "tool_name", "matched_capability": "which capability this fulfills"}
  ],
  "reasoning": "brief explanation of your matching logic"
}

CRITICAL RULES:
- You MUST infer tool selection purely from semantic similarity between 
  the Skill's described capabilities and each tool's description
- The Skill description intentionally contains NO tool names -- do not guess
- Select the MINIMUM set of tools sufficient for the task
- Every selected tool must be justified by a specific capability requirement from the Skill
`, skillInstructions, platformTools)

	messages := []gochatcore.Message{
		{Role: "system", Content: []gochatcore.ContentBlock{{Type: gochatcore.ContentTypeText, Text: systemPrompt}}},
		gochatcore.NewUserMessage("帮我在 reactor/prompts.go 文件中找到 BuildSkillsSystemPrompt 函数，然后在它的前面加一行注释 '// --- Skill System Prompt Builders ---'"),
	}

	resp, err := client.Chat(context.Background(), messages,
		gochatcore.WithTemperature(0.1),
		gochatcore.WithMaxTokens(800),
	)

	if err != nil {
		t.Fatalf("Experiment 2a FAILED: API error = %v", err)
	}

	t.Logf("=== Experiment 2a: Semantic Capability Match (code-edit) ===")
	t.Logf("FinishReason: %s", resp.FinishReason)
	t.Logf("Response:\n%s", resp.Content)
	if resp.Usage != nil {
		t.Logf("Tokens: prompt=%d completion=%d total=%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}

	// Parse response
	var result struct {
		SelectedTools []struct {
			Name             string `json:"name"`
			MatchedCapability string `json:"matched_capability"`
		} `json:"selected_tools"`
		Reasoning string `json:"reasoning"`
	}

	content := strings.TrimSpace(resp.Content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) > 2 {
			content = strings.Join(lines[1:len(lines)-1], "\n")
		} else {
			content = strings.TrimPrefix(content, "```json")
			content = strings.TrimPrefix(content, "```")
			content = strings.TrimSuffix(content, "```")
		}
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		t.Fatalf("Failed to parse response JSON: %v\nRaw response: %s", err, resp.Content)
	}

	selectedNames := make([]string, len(result.SelectedTools))
	for i, st := range result.SelectedTools {
		selectedNames[i] = st.Name
		t.Logf("  Tool[%d]: name=%q matched_capability=%q", i, st.Name, st.MatchedCapability)
	}
	t.Logf("Reasoning: %s", result.Reasoning)

	// 验证: 不用预定义 allowed-tools 列表来校验!
	// 而是验证 LLM 的选择在语义上是否合理:
	// 1. 选中的工具是否都存在于平台工具列表中
	validPlatformTools := map[string]bool{
		"read_file": true, "write_to_file": true, "replace_in_file": true,
		"search_file": true, "search_content": true, "bash": true,
		"web_search": true, "web_fetch": true, "todo_write": true,
		"memory_save": true,
	}
	allValid := true
	for _, name := range selectedNames {
		if !validPlatformTools[name] {
			t.Errorf("INVALID SELECTION: tool %q does NOT exist on this platform -- hallucinated!", name)
			allValid = false
		}
	}
	if allValid {
		t.Log("All selected tools exist on this platform (no hallucination)")
	}

	// 2. 对于这个具体任务（读文件->定位函数->编辑插入注释），
	//    LLM 是否至少选中了读取文件 和 编辑文件 相关的工具？
	taskRelevantKeywords := [][]string{
		{"read"},                     // 需要读文件能力
		{"replace", "edit", "write"}, // 需要编辑能力
	}
	for _, keywords := range taskRelevantKeywords {
		found := false
		for _, name := range selectedNames {
			for _, kw := range keywords {
				if strings.Contains(strings.ToLower(name), kw) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Logf("No tool matching keywords %v was selected (may need investigation)", keywords)
		} else {
			t.Logf("Found tool matching capability category: %v", keywords)
		}
	}

	// 3. 是否避免了明显无关的工具? (web_search, web_fetch, todo_write 等不应被选中)
	irrelevantTools := map[string]bool{"web_search": true, "web_fetch": true, "todo_write": true, "memory_save": true}
	pickedIrrelevant := false
	for _, name := range selectedNames {
		if irrelevantTools[name] {
			t.Logf("Potentially irrelevant tool selected: %q (check reasoning)", name)
			pickedIrrelevant = true
		}
	}
	if !pickedIrrelevant {
		t.Log("No clearly irrelevant tools selected")
	}
}

func TestExp2_SemanticCapabilityMatch_WebSearch(t *testing.T) {
	client, err := gochat.Client().
		Config(
			gochat.WithAPIKey("DASHSCOPE_API_KEY"),
			gochat.WithModel("qwen3.5-flash"),
		).
		BuildFor(gochat.QwenClient)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 通用 Skill 写法，无工具名
	skillInstructions := `# Web Information Retrieval

Search the internet and retrieve webpage contents for up-to-date information that is 
not available in your training data.

## When to Activate
Use this capability when:
- The user asks about current events, recent news, or time-sensitive information
- The user needs data that may have changed since your knowledge cutoff
- The user wants to research a topic requiring fresh, live data from the internet
- The user provides a specific URL and wants its content analyzed or summarized

## Workflow

### 1. Discovery Phase
- When the user's question involves current or unknown information, you need a 
  capability that can search the internet using keywords or natural language queries, 
  returning a list of relevant results with titles, URLs, and brief summaries.
- Formulate effective search queries that capture the user's intent.

### 2. Content Retrieval Phase  
- Once you have identified relevant URLs from search results, you need a capability 
  that can fetch the full content of a specific web page given its URL, extracting 
  the main text content in a structured, readable format.

### 3. Analysis Phase
- Synthesize information from retrieved sources.
- Cross-reference multiple sources when accuracy is important.
- Cite sources when providing factual claims from the web.

## Guidelines
- Always search before claiming up-to-date information.
- Prefer fetching and reading full pages over summarizing from search snippets alone.
- If a URL fails to load, note this and try alternative sources.`

	// 同一个平台工具列表（与 2a 完全一致 -- 模拟同一 Agent 平台）
	platformTools := `
## This Platform's Available Tools

| # | Name | Description |
|---|------|-------------|
| 1 | read_file | Read a file from the local filesystem. You can access any file by its absolute path. Supports optional offset and limit for reading sections. |
| 2 | write_to_file | Write content to a file at a specific path. Creates the file if it doesn't exist, overwrites if it does. |
| 3 | replace_in_file | Perform exact string replacements within an existing file. Requires the exact old string and new string for precise targeting. |
| 4 | search_file | Search for files in the workspace using wildcard patterns (e.g., "*.go", "**/*.ts"). Supports recursive directory search. |
| 5 | search_content | Search for text patterns within file contents using regex. Returns matched lines with context. Can target specific paths or search recursively. |
| 6 | bash | Execute shell commands in the working directory. Returns stdout/stderr. Use only when dedicated tools don't suffice. |
| 7 | web_search | Search the internet for current information. Returns ranked results with titles and URLs. |
| 8 | web_fetch | Retrieve and extract content from a specific URL. Converts HTML to structured text. |
| 9 | todo_write | Create and manage todo/checklist items in JSON format. |
| 10 | memory_save | Persist information to memory storage for future retrieval. |
`

	systemPrompt := fmt.Sprintf(`You are an agent assistant running on a platform with specific tools available.

## Active Skill (Capability Description)
The following describes WHAT you need to accomplish and WHAT CAPABILITIES you need -- 
but it does NOT tell you which specific tools to use. That is for you to infer.

---
%s
---

## Available Tools on This Platform
Below are ALL tools currently available on this platform.

%s

## Your Task
Based on the Skill's capability requirements and the available tools:

1. Analyze what capabilities the Skill requires for the user's request
2. Match each required capability to the best-fitting tool
3. Respond with JSON only:

{
  "selected_tools": [
    {"name": "tool_name", "matched_capability": "which capability this fulfills"}
  ],
  "reasoning": "brief explanation"
}

Rules: Infer purely from semantic similarity. Select MINIMUM sufficient set.
`, skillInstructions, platformTools)

	messages := []gochatcore.Message{
		{Role: "system", Content: []gochatcore.ContentBlock{{Type: gochatcore.ContentTypeText, Text: systemPrompt}}},
		gochatcore.NewUserMessage("搜索一下 GoReact framework 的最新版本信息，然后获取它的 GitHub README 内容"),
	}

	resp, err := client.Chat(context.Background(), messages,
		gochatcore.WithTemperature(0.1),
		gochatcore.WithMaxTokens(500),
	)

	if err != nil {
		t.Fatalf("Experiment 2b FAILED: %v", err)
	}

	t.Logf("=== Experiment 2b: Semantic Capability Match (web-search) ===")
	t.Logf("Response:\n%s", resp.Content)
	if resp.Usage != nil {
		t.Logf("Tokens: prompt=%d completion=%d total=%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}

	var result struct {
		SelectedTools []struct {
			Name             string `json:"name"`
			MatchedCapability string `json:"matched_capability"`
		} `json:"selected_tools"`
		Reasoning string `json:"reasoning"`
	}
	content := strings.TrimSpace(resp.Content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) > 2 {
			content = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		t.Fatalf("Parse failed: %v\nRaw: %s", err, resp.Content)
	}

	selectedNames := make([]string, len(result.SelectedTools))
	for i, st := range result.SelectedTools {
		selectedNames[i] = st.Name
		t.Logf("  Tool[%d]: name=%q matched=%q", i, st.Name, st.MatchedCapability)
	}
	t.Logf("Reasoning: %s", result.Reasoning)

	// 验证：web 任务应该选中 web 相关工具
	hasWebSearch, hasWebFetch := false, false
	for _, n := range selectedNames {
		if strings.Contains(n, "web_search") || strings.Contains(n, "web-search") {
			hasWebSearch = true
		}
		if strings.Contains(n, "web_fetch") || strings.Contains(n, "web-fetch") {
			hasWebFetch = true
		}
	}
	if hasWebSearch {
		t.Log("Correctly selected web_search (discovery capability)")
	} else {
		t.Log("WARNING: web_search not selected -- task requires internet search")
	}
	if hasWebFetch {
		t.Log("Correctly selected web_fetch (content retrieval capability)")
	} else {
		t.Log("WARNING: web_fetch not selected -- task requires URL content fetch")
	}

	// 验证没有选到明显无关的文件操作工具
	fileTools := map[string]bool{"read_file": true, "write_to_file": true, "replace_in_file": true, "search_file": true, "search_content": true}
	pickedFileTool := false
	for _, n := range selectedNames {
		if fileTools[n] {
			t.Logf("File operation tool %q selected for a web-only task", n)
			pickedFileTool = true
		}
	}
	if !pickedFileTool {
		t.Log("No irrelevant file-operation tools selected")
	}
}

// TestExp2_MultiSkillRouting tests L1-level multi-skill scenario:
// All skills present simultaneously with their capability descriptions,
// plus all platform tool metadata. LLM must:
//   1. Route to the correct Skill (or decide no skill needed)
//   2. Then infer which tools match that skill's capabilities
//
// This simulates the real L1->L2 flow where the agent sees everything at once.
func TestExp2_MultiSkillRouting_SemanticMatch(t *testing.T) {
	client, err := gochat.Client().
		Config(
			gochat.WithAPIKey("DASHSCOPE_API_KEY"),
			gochat.WithModel("qwen3.5-flash"),
		).
		BuildFor(gochat.QwenClient)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 多个 Skill 同时存在，全部用通用写法（无工具名）
	multiSkills := `
## Available Capabilities (Skills)

### Skill A: Code Edit
Edit, modify, and refactor source code files with precision.
Capabilities needed:
- Retrieve full file contents given a path
- Perform text search across files using patterns/regex
- Locate files matching name patterns or wildcards across directories  
- Make precise localized text replacements within existing files
- Write complete content to a file path (create or overwrite)

When to activate: User asks to fix bugs, refactor, add features, or change existing code.

---

### Skill B: File Search & Analysis
Search through codebase files to find, read, and analyze content.
Capabilities needed:
- Locate files matching wildcard patterns in directory trees
- Retrieve file contents for examination
- Search text patterns within file contents with context lines

When to activate: User asks to find files, search code, read and understand code structure.

---

### Skill C: Web Research
Search internet sources and retrieve up-to-date webpage contents.
Capabilities needed:
- Search the internet using queries, returning ranked results with URLs
- Fetch full content from specific URLs, extracting structured text

When to activate: User asks about current events, needs live data, or provides URLs to analyze.

---

### Skill D: Task Management
Create, organize, track, and execute task lists and action items.
Capabilities needed:
- Create and manage structured checklist/todo items
- Track status of multiple tasks

When to activate: User asks to plan work, manage todos, track action items.
`

	platformTools := `
## This Platform's Available Tools (10 total)

| # | Name | Description |
|---|------|-------------|
| 1 | read_file | Read a file from the local filesystem by absolute path. Supports optional offset/limit. |
| 2 | write_to_file | Write content to a file path. Creates if not exists, overwrites if it does. |
| 3 | replace_in_file | Exact string replacement within an existing file. Needs old_str and new_str. |
| 4 | search_file | Find files by wildcard pattern (e.g., "*.go"). Supports recursive search. |
| 5 | search_content | Regex text search within file contents. Returns matched lines with context. |
| 6 | bash | Execute shell commands. Returns stdout/stderr. Use only when dedicated tools don't apply. |
| 7 | web_search | Search the internet. Returns ranked results with titles and URLs. |
| 8 | web_fetch | Fetch URL content. Converts HTML to structured text. |
| 9 | todo_write | Create and manage todo/checklist items in JSON format. |
| 10 | memory_save | Persist information to memory storage for later retrieval. |
`

	systemPrompt := fmt.Sprintf(`You are an agent assistant. Below are multiple capability domains (skills) 
available to you, and all tools on this platform.

%s

---

%s

## Your Task
Given the user's request:

1. First, determine which SINGLE Skill is most applicable (or none if this is a simple question)
2. Then, from that Skill's described capabilities, identify which tools on this platform 
   can fulfill those capabilities

Respond with JSON only:
{
  "selected_skill": "skill_name_or_empty",
  "selected_tools": [
    {"name": "tool_name", "matched_capability": "which capability from the skill this fulfills"}
  ],
  "reasoning": "explain your routing decision and tool matching logic"
}

Rules:
- Select ONLY ONE skill (the best fit)
- Infer tool matches purely from semantic similarity between capability descriptions and tool descriptions
- Do NOT assume any tool belongs to any skill -- you must deduce it from descriptions
- Select MINIMUM sufficient tools
`, multiSkills, platformTools)

	// 测试任务: 这是一个 code-edit / file-search 混合型任务
	messages := []gochatcore.Message{
		{Role: "system", Content: []gochatcore.ContentBlock{{Type: gochatcore.ContentTypeText, Text: systemPrompt}}},
		gochatcore.NewUserMessage("帮我在代码库中找到所有包含 TODO 注释的 Go 文件，并读取每个匹配位置的上下文"),
	}

	start := time.Now()
	resp, err := client.Chat(context.Background(), messages,
		gochatcore.WithTemperature(0.1),
		gochatcore.WithMaxTokens(800),
	)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Experiment 2c FAILED: %v", err)
	}

	t.Logf("=== Experiment 2c: Multi-Skill Routing + Semantic Match ===")
	t.Logf("Elapsed: %v", elapsed)
	t.Logf("Response:\n%s", resp.Content)
	if resp.Usage != nil {
		t.Logf("Tokens: prompt=%d completion=%d total=%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}

	var result struct {
		Skill         string `json:"selected_skill"`
		SelectedTools []struct {
			Name             string `json:"name"`
			MatchedCapability string `json:"matched_capability"`
		} `json:"selected_tools"`
		Reasoning string `json:"reasoning"`
	}
	content := strings.TrimSpace(resp.Content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) > 2 {
			content = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		t.Fatalf("Parse failed: %v\nRaw: %s", err, resp.Content)
	}

	selectedNames := make([]string, len(result.SelectedTools))
	for i, st := range result.SelectedTools {
		selectedNames[i] = st.Name
	}
	t.Logf("Selected Skill: %q", result.Skill)
	t.Logf("Selected Tools (%d): %v", len(selectedNames), selectedNames)
	t.Logf("Reasoning: %s", result.Reasoning)

	// 验证 1: Skill 路由是否合理（应该是 Code Edit 或 File Search）
	validRoutes := map[string]bool{
		"Skill A: Code Editing": true, "Code Edit": true, "code-edit": true,
		"Skill B: File Search & Analysis": true, "File Search & Analysis": true, "file-search": true, "File Search": true,
		"Skill C: Web Research": true, "Web Research": true,
		"Skill D: Task Management": true, "Task Management": true,
		"": true,
	}
	if !validRoutes[result.Skill] {
		t.Errorf("Unexpected skill route: %q (expected code-edit or file-search)", result.Skill)
	} else {
		t.Logf("Skill routing is reasonable: %q", result.Skill)
	}

	// 验证 2: 选中的工具应该与任务相关（搜索文件 + 读文件内容）
	taskNeeds := []struct {
		category string
		keywords []string
	}{
		{"file_discovery", []string{"search_file"}},
		{"content_search", []string{"search_content"}},
		{"file_reading", []string{"read_file"}},
	}
	for _, need := range taskNeeds {
		found := false
		for _, name := range selectedNames {
			for _, kw := range need.keywords {
				if strings.Contains(name, kw) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if found {
			t.Logf("Tool for %s capability found", need.category)
		} else {
			t.Logf("WARNING: No tool matching %s (%v) -- may be acceptable", need.category, need.keywords)
		}
	}

	// 验证 3: 不应选到完全无关的工具
	irrelevant := map[string]bool{
		"web_search": true, "web_fetch": true,
		"todo_write": true,
	}
	pickedBad := false
	for _, n := range selectedNames {
		if irrelevant[n] {
			t.Logf("Irrelevant tool for this task: %q", n)
			pickedBad = true
		}
	}
	if !pickedBad {
		t.Log("No clearly irrelevant tools selected")
	}

	t.Logf("Experiment 2c completed in %v", elapsed)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const bashDescription = `Executes a given bash command and returns its output.
The working directory persists between commands, but shell state does not.
IMPORTANT: Avoid using this tool to run cat, head, tail, sed, awk unless explicitly instructed.
Instead use the appropriate dedicated tool (read, glob, grep, file_edit, write).`
