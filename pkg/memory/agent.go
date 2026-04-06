package memory

import (
	"context"
	"fmt"

	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/gorag/pkg/pattern"
)

// AgentAccessor manages Agent nodes
type AgentAccessor struct {
	BaseAccessor
}

// NewAgentAccessor creates a new AgentAccessor
func NewAgentAccessor(graphRAG pattern.GraphRAGPattern) *AgentAccessor {
	return &AgentAccessor{
		BaseAccessor: BaseAccessor{
			graphRAG: graphRAG,
			nodeType: goreactcommon.NodeTypeAgent,
		},
	}
}

// Get retrieves an agent by name
func (a *AgentAccessor) Get(ctx context.Context, agentName string) (*goreactcore.AgentNode, error) {
	node, err := a.BaseAccessor.Get(ctx, agentName)
	if err != nil {
		return nil, err
	}
	return nodeToAgentNode(node), nil
}

// List lists all agents
func (a *AgentAccessor) List(ctx context.Context) ([]*goreactcore.AgentNode, error) {
	nodes, err := a.BaseAccessor.List(ctx)
	if err != nil {
		return nil, err
	}

	agents := make([]*goreactcore.AgentNode, 0, len(nodes))
	for _, node := range nodes {
		agents = append(agents, nodeToAgentNode(node))
	}

	return agents, nil
}

// GetSkills retrieves skills for an agent
func (a *AgentAccessor) GetSkills(ctx context.Context, agentName string) ([]string, error) {
	query := fmt.Sprintf(
		"MATCH (a:%s {id: $agentId})-[:HAS_SKILL]->(s:%s) RETURN s.name as name",
		goreactcommon.NodeTypeAgent, goreactcommon.NodeTypeSkill,
	)

	results, err := a.graphRAG.QueryGraph(ctx, query, map[string]any{"agentId": agentName})
	if err != nil {
		return nil, err
	}

	skills := make([]string, 0, len(results))
	for _, result := range results {
		if name, ok := result["name"].(string); ok {
			skills = append(skills, name)
		}
	}

	return skills, nil
}

// GetModel retrieves the model for an agent
func (a *AgentAccessor) GetModel(ctx context.Context, agentName string) (string, error) {
	query := fmt.Sprintf(
		"MATCH (a:%s {id: $agentId})-[:USES_MODEL]->(m:%s) RETURN m.name as name",
		goreactcommon.NodeTypeAgent, goreactcommon.NodeTypeModel,
	)

	results, err := a.graphRAG.QueryGraph(ctx, query, map[string]any{"agentId": agentName})
	if err != nil {
		return "", err
	}

	if len(results) > 0 {
		if name, ok := results[0]["name"].(string); ok {
			return name, nil
		}
	}

	return "", nil
}
