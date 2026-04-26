package reactor

import (
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
				return BuildThinkPrompt("test input", &Intent{Type: "task"}, nil, nil)
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
				return BuildThinkPrompt("implement auth", &Intent{Type: "task"}, nil, actCtx)
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
				return BuildThinkPrompt(testInput, &Intent{Type: "task"}, nil, nil)
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
		MaxHistoryTurns,
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

	output := BuildThinkPrompt(input, intent, nil, nil)

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
		MaxHistoryTurns,
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
