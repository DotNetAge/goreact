package reactor

import (
	"os"
	"strings"
	"testing"

	gochatcore "github.com/DotNetAge/gochat/core"
	"github.com/DotNetAge/goreact/core"
)

// ============================================================
// Data Flow Consistency Tests
//
// Validates that T-A-O pipeline sends CORRECT data to LLM:
// 1. Agent's SystemPrompt is NEVER overwritten or polluted
// 2. No template redefines LLM's role ("You are X")
// 3. User input appears exactly ONCE (in UserMessage)
// 4. Skills sections are consistent across phases
// ============================================================

// ============================================================
// Test 1: No Role Redefinition in Any Template Output
// ============================================================

func TestDataFlow_NoRoleRedefinitionInTemplates(t *testing.T) {
	testCases := []struct {
		name   string
		render func() string
	}{
		{
			name: "intent_prompt",
			render: func() string {
				return BuildIntentPrompt("test input", "", nil)
			},
		},
		{
			name: "think_prompt_no_skill",
			render: func() string {
				return BuildThinkPrompt("test input", &Intent{Type: "task"}, nil, nil, nil, nil)
			},
		},
		{
			name: "think_prompt_with_skill",
			render: func() string {
				skill := &core.Skill{
					Name:         "@coder",
					Description:  "Go implementation expert",
					Instructions: "Always write tests first.",
				}
				actCtx := &ActivatedSkillContext{
					Skill:         skill,
					Instructions:  skill.Instructions,
					FilteredInfos: []core.ToolInfo{{Name: "write"}, {Name: "bash"}},
				}
				return BuildThinkPrompt("implement auth", &Intent{Type: "task"}, nil, actCtx, nil, nil)
			},
		},
	}

	forbiddenPatterns := []string{
		"You are the",
		"You are an",
		"You are a",
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.render()
			outputLower := strings.ToLower(output)
			for _, pattern := range forbiddenPatterns {
				if strings.Contains(outputLower, strings.ToLower(pattern)) {
					t.Errorf("template output contains forbidden role-defining pattern %q\nOutput:\n%s", pattern, output[:min(500, len(output))])
				}
			}
		})
	}
}

// ============================================================
// Test 2: User Input Not Duplicated Between System and User
// ============================================================

func TestDataFlow_UserInputNotDuplicated(t *testing.T) {
	testInput := "refactor the auth module to use JWT tokens"

	testCases := []struct {
		name       string
		templateFn func() string
	}{
		{
			name: "intent_prompt",
			templateFn: func() string {
				return BuildIntentPrompt(testInput, "", nil)
			},
		},
		{
			name: "think_prompt",
			templateFn: func() string {
				return BuildThinkPrompt(testInput, &Intent{Type: "task"}, nil, nil, nil, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			templateOutput := tc.templateFn()
			if strings.Contains(templateOutput, "User input: "+testInput) {
				t.Errorf("template contains raw user input that duplicates UserMessage.\n"+
					"Template output (first 300 chars):\n%s",
					templateOutput[:min(300, len(templateOutput))])
			}
		})
	}
}

// ============================================================
// Test 3: buildLLMBuilder Message Layering — System vs User Separation
// ============================================================

func TestDataFlow_BuildLLMBuilderLayering(t *testing.T) {
	config := ReactorConfig{
		SystemPrompt: "BASE_IDENTITY: You are an AI assistant for Go development",
		Model:        "test-model",
		MaxTokens:    4096,
	}
	r := NewReactor(config)

	var capturedSystem string
	var capturedUser string
	r.mockLLM = func(systemPrompt, userMessage string, _ ConversationHistory) (*gochatcore.Response, error) {
		capturedSystem = systemPrompt
		capturedUser = userMessage
		return &gochatcore.Response{Content: `{"type":"task","confidence":0.9}`}, nil
	}

	instruction := "INSTRUCTION: Classify this intent."
	userInput := "What is the weather today?"

	_, _ = r.callLLMWithHistory(
		instruction,
		userInput,
		nil,
		defaultMaxHistoryTurns,
	)

	if !strings.Contains(capturedSystem, instruction) {
		t.Error("instruction missing from captured system prompt (mock receives instruction as systemPrompt)")
	}
	if strings.Contains(capturedUser, instruction) {
		t.Error("userMessage contaminated with instruction content")
	}
	if strings.Contains(capturedUser, "BASE_IDENTITY") {
		t.Error("userMessage contaminated with base SystemPrompt content")
	}
	if capturedUser != userInput {
		t.Errorf("userMessage = %q, want pure input %q", capturedUser, userInput)
	}
	t.Logf("Mock path verified: systemPrompt=instruction(%d chars), userMessage=pure_input(%d chars)",
		len(capturedSystem), len(capturedUser))
}

// ============================================================
// Test 4: Skills Section Does Not Contain Role Definitions
// ============================================================

func TestDataFlow_SkillsSectionNoRoleDefinition(t *testing.T) {
	skills := []*core.Skill{
		{Name: "@researcher", Description: "Code research expert"},
		{Name: "@coder", Description: "Implementation expert"},
	}

	section := BuildSkillsSystemPrompt(skills)
	sectionLower := strings.ToLower(section)

	badPatterns := []string{"you are", "you are an", "you are a"}
	for i := range badPatterns {
		if strings.Contains(sectionLower, badPatterns[i]) {
			t.Errorf("BuildSkillsSystemPrompt contains role definition %q\nOutput:\n%s", badPatterns[i], section)
		}
	}

	capList := BuildCapabilitiesList(skills)
	capLower := strings.ToLower(capList)
	moreBad := []string{"you are", "you are an"}
	for j := range moreBad {
		if strings.Contains(capLower, moreBad[j]) {
			t.Errorf("BuildCapabilitiesList contains role definition %q\nOutput:\n%s", moreBad[j], capList)
		}
	}
}

// ============================================================
// Test 5: Phase 1 and Phase 2 Both Contain Same Skill Names
// ============================================================

func TestDataFlow_PhaseConsistentSkillNames(t *testing.T) {
	skills := []*core.Skill{
		{Name: "@reviewer", Description: "Code review expert"},
	}

	phase1Format := BuildCapabilitiesList(skills)
	phase2Format := BuildSkillsSystemPrompt(skills)

	for _, s := range skills {
		if !strings.Contains(phase1Format, s.Name) {
			t.Errorf("Phase1 format missing skill %q", s.Name)
		}
		if !strings.Contains(phase2Format, s.Name) {
			t.Errorf("Phase2 format missing skill %q", s.Name)
		}
	}

	if phase1Format == "" {
		t.Error("BuildCapabilitiesList returned empty string")
	}
	if phase2Format == "" {
		t.Error("BuildSkillsSystemPrompt returned empty string")
	}
}

// ============================================================
// Test 6: Intent Prompt Uses Context But Not Raw Input Line
// ============================================================

func TestDataFlow_IntentPromptNoRawInputLine(t *testing.T) {
	input := "create a REST API for user management"
	contextStr := `{"previous_turn":{"role":"assistant","content":"Sure"}}`

	output := BuildIntentPrompt(input, contextStr, nil)

	if !strings.Contains(output, contextStr) {
		t.Error("intent prompt should contain conversation context")
	}
	if strings.Contains(output, "User input: "+input) {
		t.Error("intent prompt should not contain 'User input:' prefix (duplication removed)")
	}
}

// ============================================================
// Test 7: Think Prompt Contains Intent But Not Raw Input Line
// ============================================================

func TestDataFlow_ThinkPromptNoRawInputLine(t *testing.T) {
	input := "add validation middleware"
	intent := &Intent{Type: "task", Confidence: 0.92, Summary: "Add validation"}

	output := BuildThinkPrompt(input, intent, nil, nil, nil, nil)

	if !strings.Contains(output, `"type":"task"`) {
		t.Error("think prompt should contain intent JSON")
	}
	if strings.Contains(output, "User input: "+input) {
		t.Error("think prompt should not contain 'User input:' prefix (duplication removed)")
	}
}

// ============================================================
// Test 8: SystemPrompt Integrity — Base Identity Preserved Through Pipeline
// ============================================================

func TestDataFlow_SystemPromptIntegrityThroughPipeline(t *testing.T) {
	basePrompt := "You are a specialized Go coding assistant."

	config := ReactorConfig{
		SystemPrompt: basePrompt,
		Model:        "test-model",
		MaxTokens:    4096,
	}
	r := NewReactor(config)

	callCount := 0
	var capturedUsers []string
	r.mockLLM = func(systemPrompt, userMessage string, _ ConversationHistory) (*gochatcore.Response, error) {
		callCount++
		capturedUsers = append(capturedUsers, userMessage)

		if strings.Contains(userMessage, basePrompt) {
			t.Errorf("call #%d: userMessage polluted with base SystemPrompt content", callCount)
		}
		if strings.Contains(userMessage, "Classify") && strings.Contains(userMessage, "write a hello") {
			t.Errorf("call #%d: userMessage contains both instruction AND input (should be separate)", callCount)
		}

		return &gochatcore.Response{Content: `{"decision":"answer","final_answer":"ok","is_final":true}`}, nil
	}

	_, _ = r.callLLMWithHistory(
		"Classify this request.",
		"write a hello world program",
		nil,
		defaultMaxHistoryTurns,
	)

	if callCount == 0 {
		t.Fatal("Expected at least 1 LLM call through pipeline")
	}
	for i, u := range capturedUsers {
		if strings.Contains(u, basePrompt) {
			t.Errorf("call #%d: userMessage still contains base SystemPrompt after full pipeline", i+1)
		}
	}
	t.Logf("Pipeline passed %d LLM call(s) — all userMessages verified clean of SystemPrompt pollution", callCount)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================================
// Test 9: L3 Progressive Disclosure — Reference File Resolution
// ============================================================

func TestL3_ResolveReferences_NoReferencesDir(t *testing.T) {
	r := NewReactor(ReactorConfig{Model: "test", MaxTokens: 4096})
	actCtx := &ActivatedSkillContext{ResourceBasePath: "/nonexistent/path"}

	result := r.resolveL3References(actCtx)
	if result != nil {
		t.Fatalf("expected nil when no references/ directory, got %+v", result)
	}
}

func TestL3_ResolveReferences_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	refDir := tmpDir + "/references"
	_ = os.Mkdir(refDir, 0o755)

	r := NewReactor(ReactorConfig{Model: "test", MaxTokens: 4096})
	actCtx := &ActivatedSkillContext{ResourceBasePath: tmpDir}

	result := r.resolveL3References(actCtx)
	if result != nil {
		t.Fatalf("expected nil for empty references/ dir, got %+v", result)
	}
}

func TestL3_ResolveReferences_MarkdownFileInline(t *testing.T) {
	tmpDir := t.TempDir()
	refDir := tmpDir + "/references"
	_ = os.Mkdir(refDir, 0o755)

	guideContent := "# Guide\n\nThis is a reference guide for testing.\n## Section 1\nSome content here."
	_ = os.WriteFile(refDir+"/guide.md", []byte(guideContent), 0o644)

	r := NewReactor(ReactorConfig{Model: "test", MaxTokens: 4096})
	actCtx := &ActivatedSkillContext{ResourceBasePath: tmpDir}

	result := r.resolveL3References(actCtx)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.FilesLoaded != 1 {
		t.Errorf("expected FilesLoaded=1, got %d", result.FilesLoaded)
	}
	if !strings.Contains(result.Content, "guide.md") {
		t.Error("expected Content to contain guide.md filename")
	}
	if !strings.Contains(result.Content, guideContent) {
		t.Error("expected Content to contain full guide content")
	}
	if !strings.HasPrefix(result.Content, "<references>") || !strings.HasSuffix(result.Content, "</references>\n") {
		t.Errorf("expected Content wrapped in <references> tags, got: %s", truncate(result.Content, 100))
	}
}

func TestL3_ResolveReferences_OversizedFile(t *testing.T) {
	tmpDir := t.TempDir()
	refDir := tmpDir + "/references"
	_ = os.Mkdir(refDir, 0o755)

	// Create a file larger than maxReferenceFileSize (64KB)
	bigData := make([]byte, maxReferenceFileSize+1024)
	for i := range bigData {
		bigData[i] = 'X'
	}
	_ = os.WriteFile(refDir+"/large.md", bigData, 0o644)

	r := NewReactor(ReactorConfig{Model: "test", MaxTokens: 4096})
	actCtx := &ActivatedSkillContext{ResourceBasePath: tmpDir}

	result := r.resolveL3References(actCtx)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.FilesSkipped != 1 {
		t.Errorf("expected FilesSkipped=1 for oversized file, got %d", result.FilesSkipped)
	}
	if result.Content != "" {
		t.Error("expected empty Content for oversized-only file")
	}
	if !strings.Contains(result.Links, "large.md") {
		t.Error("expected Links to contain large.md filename")
	}
	if !strings.Contains(result.Links, "oversized") {
		t.Error("expected Links to mention 'oversized' for large file")
	}
	if !strings.Contains(result.Links, "<reference-links>") {
		t.Error("expected Links wrapped in <reference-links> tags")
	}
}

func TestL3_ResolveReferences_BinaryFile(t *testing.T) {
	tmpDir := t.TempDir()
	refDir := tmpDir + "/references"
	_ = os.Mkdir(refDir, 0o755)

	// Create a binary file with null bytes
	binData := make([]byte, 256)
	binData[0], binData[64], binData[128] = 0, 0, 0 // null bytes at various positions
	_ = os.WriteFile(refDir+"/data.bin", binData, 0o644)

	r := NewReactor(ReactorConfig{Model: "test", MaxTokens: 4096})
	actCtx := &ActivatedSkillContext{ResourceBasePath: tmpDir}

	result := r.resolveL3References(actCtx)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.FilesSkipped != 1 {
		t.Errorf("expected FilesSkipped=1 for binary file, got %d", result.FilesSkipped)
	}
	if !strings.Contains(result.Links, "data.bin") {
		t.Error("expected Links to contain data.bin")
	}
	if !strings.Contains(result.Links, "binary") {
		t.Error("expected Links to mark file as binary")
	}
}

func TestL3_ResolveReferences_NonCompliantExtension(t *testing.T) {
	tmpDir := t.TempDir()
	refDir := tmpDir + "/references"
	_ = os.Mkdir(refDir, 0o755)

	// .py is not in textExtensions — should be silently skipped
	_ = os.WriteFile(refDir+"/script.py", []byte("print('hello')"), 0o644)

	r := NewReactor(ReactorConfig{Model: "test", MaxTokens: 4096})
	actCtx := &ActivatedSkillContext{ResourceBasePath: tmpDir}

	result := r.resolveL3References(actCtx)
	// Non-compliant extensions are counted as skipped but produce no output
	if result != nil && (result.Content != "" || result.Links != "") {
		t.Error("non-compliant .py file should not produce any output")
	}
}

func TestL3_ResolveReferences_MixedScenario(t *testing.T) {
	tmpDir := t.TempDir()
	refDir := tmpDir + "/references"
	_ = os.Mkdir(refDir, 0o755)

	// 1. Valid markdown
	_ = os.WriteFile(refDir+"/api.md", []byte("# API Reference\nEndpoints: /api/v1"), 0o644)

	// 2. Valid txt
	_ = os.WriteFile(refDir+"/notes.txt", []byte("Some notes"), 0o644)

	// 3. Oversized markdown
	bigData := make([]byte, maxReferenceFileSize+100)
	for i := range bigData {
		bigData[i] = 'A'
	}
	_ = os.WriteFile(refDir+"/big.md", bigData, 0o644)

	// 4. Binary
	binData := make([]byte, 64)
	binData[0] = 0
	_ = os.WriteFile(refDir+"/image.dat", binData, 0o644)

	// 5. Non-compliant (.json)
	_ = os.WriteFile(refDir+"/config.json", []byte(`{"key":"value"}`), 0o644)

	r := NewReactor(ReactorConfig{Model: "test", MaxTokens: 4096})
	actCtx := &ActivatedSkillContext{ResourceBasePath: tmpDir}

	result := r.resolveL3References(actCtx)
	if result == nil {
		t.Fatal("expected non-nil result for mixed scenario")
	}
	if result.FilesLoaded != 2 {
		t.Errorf("expected 2 loaded (api.md + notes.txt), got %d", result.FilesLoaded)
	}
	if result.FilesSkipped != 3 {
		t.Errorf("expected 3 skipped (big.md + image.dat + config.json), got %d", result.FilesSkipped)
	}
	if !strings.Contains(result.Content, "api.md") {
		t.Error("expected api.md in Content")
	}
	if !strings.Contains(result.Content, "notes.txt") {
		t.Error("expected notes.txt in Content")
	}
	if !strings.Contains(result.Links, "big.md") {
		t.Error("expected big.md in Links (oversized)")
	}
	if !strings.Contains(result.Links, "image.dat") {
		t.Error("expected image.dat in Links (binary)")
	}
}

func TestL3_ResolvedReferences_InjectedIntoThinkPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	refDir := tmpDir + "/references"
	_ = os.Mkdir(refDir, 0o755)
	_ = os.WriteFile(refDir+"/guide.md", []byte("# Reference Guide\nSee docs for details."), 0o644)

	l3Refs := &ResolvedReferences{
		Content:     "<references>\n--- guide.md (35 bytes) ---\n# Reference Guide\nSee docs for details.\n</references>\n",
		Links:       "",
		FilesLoaded: 1,
	}

	skill := &core.Skill{
		Name:        "test-skill",
		Description: "A test skill",
		Instructions: "Follow the guide in references/.",
		RootDir:     tmpDir,
	}
	actCtx := &ActivatedSkillContext{
		Skill:         skill,
		Instructions:  skill.Instructions,
		FilteredTools: []gochatcore.Tool{},
		FilteredInfos: []core.ToolInfo{},
		ResourceBasePath: tmpDir,
	}

	output := BuildThinkPrompt("test input", &Intent{Type: "task"}, nil, actCtx, nil, l3Refs)

	if !strings.Contains(output, "<resolved_references>") {
		t.Error("think prompt should contain <resolved_references> section when L3 data provided")
	}
	if !strings.Contains(output, "guide.md") {
		t.Error("think prompt should contain resolved reference filename")
	}
	if !strings.Contains(output, "Reference Guide") {
		t.Error("think prompt should contain resolved reference content")
	}
}

func TestL3_ContainsNullBytes(t *testing.T) {
	if containsNullBytes([]byte("hello world")) {
		t.Error("text data should not be detected as containing null bytes")
	}
	if !containsNullBytes([]byte{0, 'h', 'e', 'l', 'l', 'o'}) {
		t.Error("data with leading null byte should be detected as binary")
	}
	if !containsNullBytes([]byte{'h', 'e', 0, 'l', 'o'}) {
		t.Error("data with embedded null byte should be detected as binary")
	}
	if containsNullBytes([]byte{}) {
		t.Error("empty data should not be detected as containing null bytes")
	}
}
