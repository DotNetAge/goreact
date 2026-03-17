package middlewares

import (
	"github.com/ray/goreact/pkg/core"
	"github.com/ray/goreact/pkg/core/thinker"
	"github.com/ray/goreact/pkg/types"
)

// ConfidenceEvaluator 置信度评估器接口
type ConfidenceEvaluator interface {
	// Evaluate 评估置信度
	Evaluate(thought *types.Thought, ctx *core.Context) float64
}

// DefaultConfidenceEvaluator 默认置信度评估器
type DefaultConfidenceEvaluator struct{}

// Evaluate 评估置信度（基于多个因素）
func (e *DefaultConfidenceEvaluator) Evaluate(thought *types.Thought, ctx *core.Context) float64 {
	score := 1.0

	// 因素 1：是否有明确的 Action 或 FinalAnswer
	if !thought.ShouldFinish && thought.Action == nil {
		score *= 0.5
	}

	// 因素 2：Reasoning 的长度（太短可能不够充分）
	if len(thought.Reasoning) < 10 {
		score *= 0.7
	}

	// 因素 3：如果有缓存命中，使用缓存的相似度
	if cacheHit, ok := ctx.Get("cache_hit"); ok && cacheHit.(bool) {
		if similarity, ok := ctx.Get("cache_similarity"); ok {
			score *= similarity.(float64)
		}
	}

	// 因素 4：如果有 RAG 增强，提升置信度
	if ragEnhanced, ok := ctx.Get("rag_enhanced"); ok && ragEnhanced.(bool) {
		score *= 1.1
		if score > 1.0 {
			score = 1.0
		}
	}

	return score
}

// ConfidenceMiddleware 置信度评估中间件
// 在 Think 之后评估结果的置信度
func ConfidenceMiddleware(minConfidence float64, evaluator ConfidenceEvaluator) thinker.ThinkMiddleware {
	if evaluator == nil {
		evaluator = &DefaultConfidenceEvaluator{}
	}

	return func(next thinker.ThinkHandler) thinker.ThinkHandler {
		return func(task string, ctx *core.Context) (*types.Thought, error) {
			// 执行 Think
			thought, err := next(task, ctx)
			if err != nil || thought == nil {
				return thought, err
			}

			// 评估置信度
			confidence := evaluator.Evaluate(thought, ctx)

			// 存入 metadata
			if thought.Metadata == nil {
				thought.Metadata = make(map[string]interface{})
			}
			thought.Metadata["confidence"] = confidence

			// 如果置信度过低，标记需要澄清
			if confidence < minConfidence {
				thought.Metadata["needs_clarification"] = true
				thought.Metadata["clarification_reason"] = "Low confidence in intent recognition"
			}

			return thought, nil
		}
	}
}
