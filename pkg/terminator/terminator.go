package terminator

import "github.com/ray/goreact/pkg/types"

// Terminator 循环终结器接口，负责决定 ReAct 循环是否继续
type Terminator interface {
	Control(state *types.LoopState) *types.LoopAction
}
