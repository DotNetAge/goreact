package prompt

import (
	"testing"
)

func TestDefaultOutputFormat(t *testing.T) {
	format := DefaultOutputFormat()

	if format == nil {
		t.Fatal("DefaultOutputFormat() returned nil")
	}
	if format.ThoughtPrefix != "Thought:" {
		t.Errorf("ThoughtPrefix = %q, want 'Thought:'", format.ThoughtPrefix)
	}
	if format.ActionPrefix != "Action:" {
		t.Errorf("ActionPrefix = %q, want 'Action:'", format.ActionPrefix)
	}
	if format.ObservationPrefix != "Observation:" {
		t.Errorf("ObservationPrefix = %q, want 'Observation:'", format.ObservationPrefix)
	}
	if format.FinishAction != "Finish" {
		t.Errorf("FinishAction = %q, want 'Finish'", format.FinishAction)
	}
}

func TestDefaultNegativePromptGroups(t *testing.T) {
	groups := DefaultNegativePromptGroups()

	if len(groups) != 3 {
		t.Errorf("len(DefaultNegativePromptGroups) = %d, want 3", len(groups))
	}

	// Check safety group
	safetyGroup := groups[0]
	if safetyGroup.ID != "safety" {
		t.Errorf("First group ID = %q, want 'safety'", safetyGroup.ID)
	}
	if !safetyGroup.Enabled {
		t.Error("Safety group should be enabled")
	}

	// Check format group
	formatGroup := groups[1]
	if formatGroup.ID != "format" {
		t.Errorf("Second group ID = %q, want 'format'", formatGroup.ID)
	}

	// Check behavior group
	behaviorGroup := groups[2]
	if behaviorGroup.ID != "behavior" {
		t.Errorf("Third group ID = %q, want 'behavior'", behaviorGroup.ID)
	}
}

func TestDefaultExamples(t *testing.T) {
	examples := DefaultExamples()

	if len(examples) < 1 {
		t.Error("DefaultExamples() should return at least one example")
	}

	// Check first example
	ex := examples[0]
	if ex.ID != "success-001" {
		t.Errorf("First example ID = %q, want 'success-001'", ex.ID)
	}
	if ex.Question == "" {
		t.Error("Question should not be empty")
	}
	if len(ex.Thoughts) == 0 {
		t.Error("Thoughts should not be empty")
	}
	if ex.FinalAnswer == "" {
		t.Error("FinalAnswer should not be empty")
	}
}

func TestDefaultEvolutionConfig(t *testing.T) {
	config := DefaultEvolutionConfig()

	if config == nil {
		t.Fatal("DefaultEvolutionConfig() returned nil")
	}
	if config.MaxReflections != 5 {
		t.Errorf("MaxReflections = %d, want 5", config.MaxReflections)
	}
	if config.MaxSimilarPlans != 3 {
		t.Errorf("MaxSimilarPlans = %d, want 3", config.MaxSimilarPlans)
	}
	if !config.EnableReflectionInjection {
		t.Error("EnableReflectionInjection should be true")
	}
	if !config.EnablePlanInjection {
		t.Error("EnablePlanInjection should be true")
	}
}

func TestDefaultPromptTemplateConfig(t *testing.T) {
	config := DefaultPromptTemplateConfig()

	if config == nil {
		t.Fatal("DefaultPromptTemplateConfig() returned nil")
	}
	if config.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want 4096", config.MaxTokens)
	}
	if !config.EnableRAG {
		t.Error("EnableRAG should be true")
	}
	if !config.EnableFewShot {
		t.Error("EnableFewShot should be true")
	}
}

func TestPrompt(t *testing.T) {
	prompt := &Prompt{
		UserQuery: "What is the weather?",
		Metadata:  map[string]any{"source": "test"},
	}

	if prompt.UserQuery != "What is the weather?" {
		t.Errorf("UserQuery = %q, want 'What is the weather?'", prompt.UserQuery)
	}
	if prompt.Metadata["source"] != "test" {
		t.Errorf("Metadata[source] = %v, want 'test'", prompt.Metadata["source"])
	}
}

func TestSystemPrompt(t *testing.T) {
	sysPrompt := &SystemPrompt{
		Role:         "Assistant",
		Behavior:     "Helpful and accurate",
		Constraints:  "Do not make up information",
		OutputFormat: "Thought/Action/Observation",
	}

	if sysPrompt.Role != "Assistant" {
		t.Errorf("Role = %q, want 'Assistant'", sysPrompt.Role)
	}
}

func TestNegativePrompt(t *testing.T) {
	np := &NegativePrompt{
		ID:          "np-001",
		Pattern:     "Do not guess",
		Reason:      "Avoid hallucinations",
		Alternative: "Use tools to get accurate info",
		Severity:    "high",
	}

	if np.ID != "np-001" {
		t.Errorf("ID = %q, want 'np-001'", np.ID)
	}
	if np.Severity != "high" {
		t.Errorf("Severity = %q, want 'high'", np.Severity)
	}
}

func TestRAGContext(t *testing.T) {
	rag := &RAGContext{
		Query: "test query",
		Mode:  RAGModeHybrid,
		Documents: []*Document{
			{ID: "doc1", Content: "content1", Score: 0.9},
		},
	}

	if rag.Mode != RAGModeHybrid {
		t.Errorf("Mode = %d, want RAGModeHybrid", rag.Mode)
	}
	if len(rag.Documents) != 1 {
		t.Errorf("len(Documents) = %d, want 1", len(rag.Documents))
	}
}

func TestRAGMode(t *testing.T) {
	modes := []RAGMode{
		RAGModeNative,
		RAGModeGraph,
		RAGModeHybrid,
	}

	if modes[0] != RAGModeNative {
		t.Errorf("RAGModeNative = %d, want 0", modes[0])
	}
	if modes[1] != RAGModeGraph {
		t.Errorf("RAGModeGraph = %d, want 1", modes[1])
	}
	if modes[2] != RAGModeHybrid {
		t.Errorf("RAGModeHybrid = %d, want 2", modes[2])
	}
}

func TestDocument(t *testing.T) {
	doc := &Document{
		ID:       "doc-001",
		Content:  "This is the document content",
		Source:   "https://example.com/doc",
		Score:    0.95,
		Metadata: map[string]any{"author": "test"},
	}

	if doc.ID != "doc-001" {
		t.Errorf("ID = %q, want 'doc-001'", doc.ID)
	}
	if doc.Score != 0.95 {
		t.Errorf("Score = %f, want 0.95", doc.Score)
	}
}

func TestToolDefinition(t *testing.T) {
	toolDef := &ToolDefinition{
		Name:        "read_file",
		Description: "Read file contents",
		Parameters: map[string]any{
			"path": map[string]any{"type": "string"},
		},
		Examples: []string{"read_file[path='/tmp/test.txt']"},
	}

	if toolDef.Name != "read_file" {
		t.Errorf("Name = %q, want 'read_file'", toolDef.Name)
	}
	if len(toolDef.Examples) != 1 {
		t.Errorf("len(Examples) = %d, want 1", len(toolDef.Examples))
	}
}

func TestExample(t *testing.T) {
	ex := &Example{
		ID:           "ex-001",
		Question:     "What is 2+2?",
		Thoughts:     []string{"Need to add 2 and 2"},
		Actions:      []string{"calculate[2+2]"},
		Observations: []string{"4"},
		FinalAnswer:  "4",
		Tags:         []string{"math", "simple"},
		Difficulty:   1,
	}

	if ex.Difficulty != 1 {
		t.Errorf("Difficulty = %d, want 1", ex.Difficulty)
	}
	if len(ex.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(ex.Tags))
	}
}

func TestPromptSection(t *testing.T) {
	section := &PromptSection{
		Type:     "system",
		Content:  "You are a helpful assistant",
		Priority: 100,
	}

	if section.Type != "system" {
		t.Errorf("Type = %q, want 'system'", section.Type)
	}
	if section.Priority != 100 {
		t.Errorf("Priority = %d, want 100", section.Priority)
	}
}

func TestBuildRequest(t *testing.T) {
	req := &BuildRequest{
		Input:       "Test question",
		TemplateID:  "default",
		Permission:  &Permission{IsAdmin: false, Permissions: []string{"read"}},
	}

	if req.Input != "Test question" {
		t.Errorf("Input = %q, want 'Test question'", req.Input)
	}
	if req.Permission == nil {
		t.Error("Permission should not be nil")
	}
}

func TestPermission(t *testing.T) {
	perm := &Permission{
		IsAdmin:     true,
		Permissions: []string{"read", "write", "delete"},
	}

	if !perm.IsAdmin {
		t.Error("IsAdmin should be true")
	}
	if len(perm.Permissions) != 3 {
		t.Errorf("len(Permissions) = %d, want 3", len(perm.Permissions))
	}
}

func TestInjectionStrategy(t *testing.T) {
	strategies := []InjectionStrategy{
		InjectionPrefix,
		InjectionInfix,
		InjectionSuffix,
		InjectionDynamic,
	}

	if strategies[0] != InjectionPrefix {
		t.Errorf("InjectionPrefix = %d, want 0", strategies[0])
	}
}

func TestGraphContext(t *testing.T) {
	gc := &GraphContext{
		Nodes: []*GraphNode{{ID: "node1"}},
		Edges: []*Edge{{Source: "A", Target: "B"}},
		Paths: []*Path{{Nodes: []string{"A", "B"}}},
	}

	if len(gc.Nodes) != 1 {
		t.Errorf("len(Nodes) = %d, want 1", len(gc.Nodes))
	}
	if len(gc.Edges) != 1 {
		t.Errorf("len(Edges) = %d, want 1", len(gc.Edges))
	}
}

func TestGraphNode(t *testing.T) {
	node := &GraphNode{
		ID:         "node-001",
		Type:       "Entity",
		Label:      "User",
		Properties: map[string]any{"name": "Alice"},
	}

	if node.ID != "node-001" {
		t.Errorf("ID = %q, want 'node-001'", node.ID)
	}
}

func TestEdge(t *testing.T) {
	edge := &Edge{
		Source:   "A",
		Target:   "B",
		Relation: "KNOWS",
		Weight:   0.8,
	}

	if edge.Relation != "KNOWS" {
		t.Errorf("Relation = %q, want 'KNOWS'", edge.Relation)
	}
}

func TestTrajectory(t *testing.T) {
	traj := &Trajectory{
		Steps: []*TrajectoryStep{
			{Thought: "I need to check", Action: "read", Observation: "done"},
		},
		Success:      true,
		FailurePoint: -1,
	}

	if len(traj.Steps) != 1 {
		t.Errorf("len(Steps) = %d, want 1", len(traj.Steps))
	}
	if !traj.Success {
		t.Error("Success should be true")
	}
}

func TestReflection(t *testing.T) {
	ref := &Reflection{
		FailureReason: "Tool not found",
		Heuristic:     "Check tool availability before use",
		Suggestions:   []string{"Use list_tools first"},
	}

	if ref.FailureReason != "Tool not found" {
		t.Errorf("FailureReason = %q, want 'Tool not found'", ref.FailureReason)
	}
	if len(ref.Suggestions) != 1 {
		t.Errorf("len(Suggestions) = %d, want 1", len(ref.Suggestions))
	}
}

func TestConflict(t *testing.T) {
	conflict := &Conflict{
		Higher:      SourceSystemRole,
		Lower:       SourceSkillPrompt,
		Description: "Conflicting instructions",
		Resolution:  "System role takes precedence",
	}

	if conflict.Higher != SourceSystemRole {
		t.Errorf("Higher = %d, want SourceSystemRole", conflict.Higher)
	}
}

func TestPromptSource(t *testing.T) {
	sources := []PromptSource{
		SourceNegativePrompt,
		SourceSystemRole,
		SourceSkillPrompt,
		SourceUserRequest,
	}

	if sources[0] != SourceNegativePrompt {
		t.Errorf("SourceNegativePrompt = %d, want 0", sources[0])
	}
}
