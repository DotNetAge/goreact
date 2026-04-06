package memory

import (
	"context"
	"time"

	"github.com/DotNetAge/gorag/pkg/core"
	goreactcommon "github.com/DotNetAge/goreact/pkg/common"
	goreactcore "github.com/DotNetAge/goreact/pkg/core"
	"github.com/DotNetAge/gorag/pkg/pattern"
)

// TrajectoryAccessor manages Trajectory nodes
type TrajectoryAccessor struct {
	BaseAccessor
}

// NewTrajectoryAccessor creates a new TrajectoryAccessor
func NewTrajectoryAccessor(graphRAG pattern.GraphRAGPattern) *TrajectoryAccessor {
	return &TrajectoryAccessor{
		BaseAccessor: BaseAccessor{
			graphRAG: graphRAG,
			nodeType: goreactcommon.NodeTypeTrajectory,
		},
	}
}

// Get retrieves a trajectory by name
func (a *TrajectoryAccessor) Get(ctx context.Context, trajectoryName string) (*goreactcore.TrajectoryNode, error) {
	node, err := a.BaseAccessor.Get(ctx, trajectoryName)
	if err != nil {
		return nil, err
	}
	return nodeToTrajectoryNode(node), nil
}

// List lists trajectories
func (a *TrajectoryAccessor) List(ctx context.Context, opts ...ListOption) ([]*goreactcore.TrajectoryNode, error) {
	nodes, err := a.BaseAccessor.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	trajectories := make([]*goreactcore.TrajectoryNode, 0, len(nodes))
	for _, node := range nodes {
		trajectories = append(trajectories, nodeToTrajectoryNode(node))
	}

	return trajectories, nil
}

// Add adds a trajectory
func (a *TrajectoryAccessor) Add(ctx context.Context, trajectory *goreactcore.TrajectoryNode) error {
	node := &core.Node{
		ID:   trajectory.Name,
		Type: goreactcommon.NodeTypeTrajectory,
		Properties: map[string]any{
			"name":          trajectory.Name,
			"node_type":     goreactcommon.NodeTypeTrajectory,
			"session_name":  trajectory.SessionName,
			"steps":         trajectory.Steps,
			"success":       trajectory.Success,
			"failure_point": trajectory.FailurePoint,
			"final_result":  trajectory.FinalResult,
			"duration":      trajectory.Duration.String(),
			"summary":       trajectory.Summary,
			"created_at":    trajectory.CreatedAt.Format(time.RFC3339),
		},
	}

	return a.graphRAG.AddNode(ctx, node)
}

// AddStep adds a step to a trajectory
func (a *TrajectoryAccessor) AddStep(ctx context.Context, trajectoryName string, step *goreactcore.TrajectoryStep) error {
	trajectory, err := a.Get(ctx, trajectoryName)
	if err != nil {
		return err
	}

	trajectory.Steps = append(trajectory.Steps, step)
	return a.Add(ctx, trajectory)
}

// MarkComplete marks a trajectory as complete
func (a *TrajectoryAccessor) MarkComplete(ctx context.Context, trajectoryName string, success bool, result string) error {
	trajectory, err := a.Get(ctx, trajectoryName)
	if err != nil {
		return err
	}

	trajectory.Success = success
	trajectory.FinalResult = result
	if !success {
		trajectory.FailurePoint = len(trajectory.Steps) - 1
	}

	return a.Add(ctx, trajectory)
}
