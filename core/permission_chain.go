package core

type PermissionDecision struct {
	Behavior    PermissionBehavior
	Message     string
	UpdatedInput map[string]any
}

type PermissionChain interface {
	Check(ctx *ToolUseContext) (*PermissionDecision, error)
}

type chainPermissionChain struct {
	checkers []ToolPermissionChecker
}

func NewPermissionChain(checkers ...ToolPermissionChecker) PermissionChain {
	return &chainPermissionChain{checkers: checkers}
}

func (c *chainPermissionChain) Check(ctx *ToolUseContext) (*PermissionDecision, error) {
	for _, checker := range c.checkers {
		result := checker.CheckPermissions(ctx)
		if result.Behavior != PermissionAllow {
			return &PermissionDecision{
				Behavior:    result.Behavior,
				Message:     result.Message,
				UpdatedInput: result.UpdatedInput,
			}, nil
		}
		if result.UpdatedInput != nil {
			ctx.Params = result.UpdatedInput
		}
	}
	return &PermissionDecision{Behavior: PermissionAllow}, nil
}
