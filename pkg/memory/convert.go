package memory

import (
	"fmt"
	"time"

	"github.com/DotNetAge/gorag/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
	goreactskill "github.com/DotNetAge/goreact/pkg/skill"
	goreacttool "github.com/DotNetAge/goreact/pkg/tool"
)

// Helper functions for ID generation
func generateSessionID() string {
	return "session-" + generateID()
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}

func parseTime(v any) time.Time {
	if v == nil {
		return time.Time{}
	}
	if s, ok := v.(string); ok {
		t, _ := time.Parse(time.RFC3339, s)
		return t
	}
	return time.Time{}
}

// anyToAgentNode converts an any value to an agent Node
func anyToAgentNode(name string, v any) (*core.Node, error) {
	// Try different known types
	switch a := v.(type) {
	case goreactcore.AgentNode:
		return agentNodeToNode(a), nil
	case *goreactcore.AgentNode:
		return agentNodeToNode(*a), nil
	case map[string]any:
		return mapToAgentNode(name, a), nil
	default:
		return nil, fmt.Errorf("unsupported agent type: %T", v)
	}
}

// anyToSkillNode converts an any value to a skill Node
func anyToSkillNode(name string, v any) (*core.Node, error) {
	switch s := v.(type) {
	case goreactskill.Skill:
		return skillToNode(s), nil
	case *goreactskill.Skill:
		return skillToNode(*s), nil
	case goreactskill.SkillNode:
		return skillNodeToNode(s), nil
	case *goreactskill.SkillNode:
		return skillNodeToNode(*s), nil
	case map[string]any:
		return mapToSkillNode(name, s), nil
	default:
		return nil, fmt.Errorf("unsupported skill type: %T", v)
	}
}

// anyToToolNode converts an any value to a tool Node
func anyToToolNode(name string, v any) (*core.Node, error) {
	switch t := v.(type) {
	case goreacttool.ToolNode:
		return toolNodeToNode(t), nil
	case *goreacttool.ToolNode:
		return toolNodeToNode(*t), nil
	case map[string]any:
		return mapToToolNode(name, t), nil
	default:
		return nil, fmt.Errorf("unsupported tool type: %T", v)
	}
}

// anyToModelNode converts an any value to a model Node
func anyToModelNode(name string, v any) (*core.Node, error) {
	switch m := v.(type) {
	case goreactcore.ModelNode:
		return modelNodeToNode(m), nil
	case *goreactcore.ModelNode:
		return modelNodeToNode(*m), nil
	case map[string]any:
		return mapToModelNode(name, m), nil
	default:
		return nil, fmt.Errorf("unsupported model type: %T", v)
	}
}

// getAgentSkills extracts skills from an agent (returns empty if not available)
func getAgentSkills(v any) []string {
	switch a := v.(type) {
	case goreactcore.AgentNode:
		return a.Skills
	case *goreactcore.AgentNode:
		return a.Skills
	case map[string]any:
		if skills, ok := a["skills"].([]string); ok {
			return skills
		}
	}
	return nil
}

// Node conversion functions

func agentNodeToNode(agent goreactcore.AgentNode) *core.Node {
	return &core.Node{
		ID:   agent.Name,
		Type: goreactcommon.NodeTypeAgent,
		Properties: map[string]any{
			"name":            agent.Name,
			"node_type":       goreactcommon.NodeTypeAgent,
			"domain":          agent.Domain,
			"description":     agent.Description,
			"model":           agent.Model,
			"skills":          agent.Skills,
			"prompt_template": agent.PromptTemplate,
			"created_at":      time.Now().Format(time.RFC3339),
		},
	}
}

func mapToAgentNode(name string, m map[string]any) *core.Node {
	return &core.Node{
		ID:   name,
		Type: goreactcommon.NodeTypeAgent,
		Properties: map[string]any{
			"name":            name,
			"node_type":       goreactcommon.NodeTypeAgent,
			"domain":          m["domain"],
			"description":     m["description"],
			"model":           m["model"],
			"skills":          m["skills"],
			"prompt_template": m["prompt_template"],
			"created_at":      time.Now().Format(time.RFC3339),
		},
	}
}

func skillToNode(skill goreactskill.Skill) *core.Node {
	return &core.Node{
		ID:   skill.Name,
		Type: goreactcommon.NodeTypeSkill,
		Properties: map[string]any{
			"name":          skill.Name,
			"node_type":     goreactcommon.NodeTypeSkill,
			"description":   skill.Description,
			"agent":         skill.Agent,
			"intent":        skill.Intent,
			"template":      skill.Template,
			"parameters":    skill.Parameters,
			"allowed_tools": skill.AllowedTools,
			"created_at":    skill.CreatedAt.Format(time.RFC3339),
		},
	}
}

func skillNodeToNode(skill goreactskill.SkillNode) *core.Node {
	return &core.Node{
		ID:   skill.Name,
		Type: goreactcommon.NodeTypeSkill,
		Properties: map[string]any{
			"name":          skill.Name,
			"node_type":     goreactcommon.NodeTypeSkill,
			"description":   skill.Description,
			"agent":         skill.Agent,
			"intent":        skill.Intent,
			"template":      skill.Template,
			"parameters":    skill.Parameters,
			"allowed_tools": skill.AllowedTools,
		},
	}
}

func mapToSkillNode(name string, m map[string]any) *core.Node {
	return &core.Node{
		ID:   name,
		Type: goreactcommon.NodeTypeSkill,
		Properties: map[string]any{
			"name":          name,
			"node_type":     goreactcommon.NodeTypeSkill,
			"description":   m["description"],
			"agent":         m["agent"],
			"intent":        m["intent"],
			"template":      m["template"],
			"parameters":    m["parameters"],
			"allowed_tools": m["allowed_tools"],
			"created_at":    time.Now().Format(time.RFC3339),
		},
	}
}

func toolNodeToNode(tool goreacttool.ToolNode) *core.Node {
	return &core.Node{
		ID:   tool.Name,
		Type: goreactcommon.NodeTypeTool,
		Properties: map[string]any{
			"name":           tool.Name,
			"node_type":      goreactcommon.NodeTypeTool,
			"description":    tool.Description,
			"type":           string(tool.Type),
			"security_level": string(tool.SecurityLevel),
			"is_idempotent":  tool.IsIdempotent,
			"created_at":     time.Now().Format(time.RFC3339),
		},
	}
}

func mapToToolNode(name string, m map[string]any) *core.Node {
	return &core.Node{
		ID:   name,
		Type: goreactcommon.NodeTypeTool,
		Properties: map[string]any{
			"name":           name,
			"node_type":      goreactcommon.NodeTypeTool,
			"description":    m["description"],
			"type":           m["type"],
			"security_level": m["security_level"],
			"is_idempotent":  m["is_idempotent"],
			"created_at":     time.Now().Format(time.RFC3339),
		},
	}
}

func modelNodeToNode(model goreactcore.ModelNode) *core.Node {
	return &core.Node{
		ID:   model.Name,
		Type: goreactcommon.NodeTypeModel,
		Properties: map[string]any{
			"name":        model.Name,
			"node_type":   goreactcommon.NodeTypeModel,
			"description": model.Description,
			"provider":    model.Provider,
			"created_at":  time.Now().Format(time.RFC3339),
		},
	}
}

func mapToModelNode(name string, m map[string]any) *core.Node {
	return &core.Node{
		ID:   name,
		Type: goreactcommon.NodeTypeModel,
		Properties: map[string]any{
			"name":        name,
			"node_type":   goreactcommon.NodeTypeModel,
			"description": m["description"],
			"provider":    m["provider"],
			"created_at":  time.Now().Format(time.RFC3339),
		},
	}
}

// Node to struct conversion functions

func nodeToSessionNode(node *core.Node) *goreactcore.SessionNode {
	if node == nil {
		return nil
	}
	return &goreactcore.SessionNode{
		BaseNode: goreactcore.BaseNode{
			Name:        node.ID,
			NodeType:    goreactcommon.NodeTypeSession,
			Description: getString(node.Properties["description"]),
			CreatedAt:   parseTime(node.Properties["created_at"]),
			Metadata:    node.Properties,
		},
		UserName:  getString(node.Properties["user_name"]),
		StartTime: parseTime(node.Properties["start_time"]),
		Status:    goreactcommon.SessionStatus(getString(node.Properties["status"])),
	}
}

func nodeToMemoryItemNode(node *core.Node) *goreactcore.MemoryItemNode {
	if node == nil {
		return nil
	}
	return &goreactcore.MemoryItemNode{
		BaseNode: goreactcore.BaseNode{
			Name:        node.ID,
			NodeType:    goreactcommon.NodeTypeMemoryItem,
			Description: getString(node.Properties["content"]),
			CreatedAt:   parseTime(node.Properties["created_at"]),
			Metadata:    node.Properties,
		},
		SessionName:   getString(node.Properties["session_name"]),
		Content:       getString(node.Properties["content"]),
		Type:          goreactcommon.MemoryItemType(getString(node.Properties["type"])),
		Source:        goreactcommon.MemorySource(getString(node.Properties["source"])),
		Importance:    getFloat64(node.Properties["importance"]),
		EmphasisLevel: parseEmphasisLevel(node.Properties["emphasis_level"]),
	}
}

func nodeToAgentNode(node *core.Node) *goreactcore.AgentNode {
	if node == nil {
		return nil
	}

	return &goreactcore.AgentNode{
		BaseNode: goreactcore.BaseNode{
			Name:        node.ID,
			NodeType:    goreactcommon.NodeTypeAgent,
			Description: getString(node.Properties["description"]),
			CreatedAt:   parseTime(node.Properties["created_at"]),
			Metadata:    node.Properties,
		},
		Domain:         getString(node.Properties["domain"]),
		Model:          getString(node.Properties["model"]),
		Skills:         getStringSlice(node.Properties["skills"]),
		PromptTemplate: getString(node.Properties["prompt_template"]),
	}
}

func nodeToSkill(node *core.Node) goreactskill.Skill {
	if node == nil {
		return goreactskill.Skill{}
	}

	return goreactskill.Skill{
		Name:         node.ID,
		Description:  getString(node.Properties["description"]),
		Agent:        getString(node.Properties["agent"]),
		Intent:       getString(node.Properties["intent"]),
		Template:     getString(node.Properties["template"]),
		AllowedTools: getStringSlice(node.Properties["allowed_tools"]),
	}
}

func nodeToExecutionPlan(node *core.Node) *goreactskill.SkillExecutionPlan {
	if node == nil {
		return nil
	}

	return &goreactskill.SkillExecutionPlan{
		Name:           node.ID,
		SkillName:      getString(node.Properties["skill_name"]),
		CompiledAt:     parseTime(node.Properties["compiled_at"]),
		ExecutionCount: getInt(node.Properties["execution_count"]),
		SuccessRate:    getFloat64(node.Properties["success_rate"]),
	}
}

func nodeToGeneratedSkill(node *core.Node) goreactskill.GeneratedSkill {
	if node == nil {
		return goreactskill.GeneratedSkill{}
	}

	return goreactskill.GeneratedSkill{
		Name:      node.ID,
		Content:   getString(node.Properties["content"]),
		FilePath:  getString(node.Properties["file_path"]),
		Status:    goreactcommon.GeneratedStatus(getString(node.Properties["status"])),
	}
}

func mapToGeneratedSkill(m map[string]any) goreactskill.GeneratedSkill {
	return goreactskill.GeneratedSkill{
		Name:      getString(m["name"]),
		Content:   getString(m["content"]),
		FilePath:  getString(m["file_path"]),
		Status:    goreactcommon.GeneratedStatus(getString(m["status"])),
	}
}

func nodeToTool(node *core.Node) goreacttool.ToolNode {
	if node == nil {
		return goreacttool.ToolNode{}
	}

	return goreacttool.ToolNode{
		Name:          node.ID,
		NodeType:      goreactcommon.NodeTypeTool,
		Description:   getString(node.Properties["description"]),
		Type:          goreactcommon.ToolType(getString(node.Properties["type"])),
		SecurityLevel: parseSecurityLevel(node.Properties["security_level"]),
		IsIdempotent:  getBool(node.Properties["is_idempotent"]),
	}
}

func nodeToGeneratedTool(node *core.Node) goreacttool.GeneratedTool {
	if node == nil {
		return goreacttool.GeneratedTool{}
	}

	return goreacttool.GeneratedTool{
		Name:      node.ID,
		Code:      getString(node.Properties["code"]),
		Status:    goreactcommon.GeneratedStatus(getString(node.Properties["status"])),
	}
}

func mapToGeneratedTool(m map[string]any) goreacttool.GeneratedTool {
	return goreacttool.GeneratedTool{
		Name:      getString(m["name"]),
		Code:      getString(m["code"]),
		Status:    goreactcommon.GeneratedStatus(getString(m["status"])),
	}
}

func nodeToReflectionNode(node *core.Node) *goreactcore.ReflectionNode {
	if node == nil {
		return nil
	}

	return &goreactcore.ReflectionNode{
		BaseNode: goreactcore.BaseNode{
			Name:        node.ID,
			NodeType:    goreactcommon.NodeTypeReflection,
			Description: getString(node.Properties["failure_reason"]),
			CreatedAt:   parseTime(node.Properties["created_at"]),
			Metadata:    node.Properties,
		},
		SessionName:    getString(node.Properties["session_name"]),
		TrajectoryName: getString(node.Properties["trajectory_name"]),
		FailureReason:  getString(node.Properties["failure_reason"]),
		Analysis:       getString(node.Properties["analysis"]),
		Heuristic:      getString(node.Properties["heuristic"]),
		Suggestions:    getStringSlice(node.Properties["suggestions"]),
		Score:          getFloat64(node.Properties["score"]),
		TaskType:       getString(node.Properties["task_type"]),
	}
}

func nodeToPlanNode(node *core.Node) *goreactcore.PlanNode {
	if node == nil {
		return nil
	}

	return &goreactcore.PlanNode{
		BaseNode: goreactcore.BaseNode{
			Name:        node.ID,
			NodeType:    goreactcommon.NodeTypePlan,
			Description: getString(node.Properties["goal"]),
			CreatedAt:   parseTime(node.Properties["created_at"]),
			Metadata:    node.Properties,
		},
		SessionName: getString(node.Properties["session_name"]),
		Goal:        getString(node.Properties["goal"]),
		Status:      goreactcommon.PlanStatus(getString(node.Properties["status"])),
		Success:     getBool(node.Properties["success"]),
		TaskType:    getString(node.Properties["task_type"]),
	}
}

func nodeToTrajectoryNode(node *core.Node) *goreactcore.TrajectoryNode {
	if node == nil {
		return nil
	}

	duration, _ := time.ParseDuration(getString(node.Properties["duration"]))

	return &goreactcore.TrajectoryNode{
		BaseNode: goreactcore.BaseNode{
			Name:        node.ID,
			NodeType:    goreactcommon.NodeTypeTrajectory,
			Description: getString(node.Properties["summary"]),
			CreatedAt:   parseTime(node.Properties["created_at"]),
			Metadata:    node.Properties,
		},
		SessionName:  getString(node.Properties["session_name"]),
		Success:      getBool(node.Properties["success"]),
		FailurePoint: getInt(node.Properties["failure_point"]),
		FinalResult:  getString(node.Properties["final_result"]),
		Duration:     duration,
		Summary:      getString(node.Properties["summary"]),
	}
}

func nodeToFrozenSessionNode(node *core.Node) *goreactcore.FrozenSessionNode {
	if node == nil {
		return nil
	}

	return &goreactcore.FrozenSessionNode{
		BaseNode: goreactcore.BaseNode{
			Name:        node.ID,
			NodeType:    goreactcommon.NodeTypeFrozenSession,
			Description: getString(node.Properties["suspend_reason"]),
			CreatedAt:   parseTime(node.Properties["created_at"]),
			Metadata:    node.Properties,
		},
		SessionName:   getString(node.Properties["session_name"]),
		QuestionID:    getString(node.Properties["question_id"]),
		StateData:     []byte(getString(node.Properties["state_data"])),
		Status:        goreactcommon.FrozenStatus(getString(node.Properties["status"])),
		SuspendReason: getString(node.Properties["suspend_reason"]),
		UserName:      getString(node.Properties["user_name"]),
		AgentName:     getString(node.Properties["agent_name"]),
		Priority:      goreactcommon.TaskPriority(getString(node.Properties["priority"])),
	}
}

// Helper functions for type-safe property extraction

func getString(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case fmt.Stringer:
		return s.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func getStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		result := make([]string, 0, len(s))
		for _, item := range s {
			result = append(result, getString(item))
		}
		return result
	default:
		return nil
	}
}

func getInt(v any) int {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case int32:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}

func getFloat64(v any) float64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func getBool(v any) bool {
	if v == nil {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	default:
		return false
	}
}

func parseEmphasisLevel(v any) goreactcommon.EmphasisLevel {
	if v == nil {
		return goreactcommon.EmphasisLevelNormal
	}
	switch s := v.(type) {
	case string:
		switch s {
		case "EmphasisLevelImportant", "important":
			return goreactcommon.EmphasisLevelImportant
		case "EmphasisLevelCritical", "critical":
			return goreactcommon.EmphasisLevelCritical
		default:
			return goreactcommon.EmphasisLevelNormal
		}
	case goreactcommon.EmphasisLevel:
		return s
	case int:
		return goreactcommon.EmphasisLevel(s)
	default:
		return goreactcommon.EmphasisLevelNormal
	}
}

func parseSecurityLevel(v any) goreactcommon.SecurityLevel {
	if v == nil {
		return goreactcommon.LevelSafe
	}
	switch s := v.(type) {
	case string:
		switch s {
		case "LevelSensitive", "sensitive":
			return goreactcommon.LevelSensitive
		case "LevelHighRisk", "high_risk":
			return goreactcommon.LevelHighRisk
		default:
			return goreactcommon.LevelSafe
		}
	case goreactcommon.SecurityLevel:
		return s
	case int:
		return goreactcommon.SecurityLevel(s)
	default:
		return goreactcommon.LevelSafe
	}
}
