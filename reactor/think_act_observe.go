package reactor

import (
	"fmt"
	"time"

	"github.com/DotNetAge/goreact/core"
)

// Think asks the LLM to decide the next action based on the current context.
// Implements TWO-PHASE thinking:
//
//	Phase 1 (Skill Selection):  Lightweight — choose which Skill to use (if any)
//	Phase 2 (Tool Planning):    Weighted — under chosen Skill's L2 instructions,
//	                            decide which Tool to invoke with what parameters
//
// Uses streaming to emit ThinkingDelta events in real-time via EventBus.
// Token accounting covers: tool definitions, skill sections, prompts, and L2 instructions.
func (r *Reactor) Think(ctx *ReactContext) (int, error) {
	estimateFn := r.tokenEstimator.Estimate
	totalTokens := 0

	toolInfos := r.toolRegistry.ToToolInfos()
	llmTools := ToolInfosToLLMTools(toolInfos)

	skills, _ := r.skillRegistry.FindApplicableSkills(ctx.Intent)
	skillsSection := BuildSkillsSystemPrompt(skills)

	var memoryRecords []core.MemoryRecord
	if r.memory != nil {
		records, err := r.memory.Retrieve(
			ctx.Ctx(), ctx.Input,
			core.WithMemoryTypes(core.MemoryTypeLongTerm, core.MemoryTypeUser, core.MemoryTypeExperience),
			core.WithMemoryLimit(3),
		)
		if err != nil {
			ctx.EmitEvent(core.Error, fmt.Sprintf("memory retrieval failed (non-fatal): %v", err))
		} else {
			memoryRecords = records
		}
	}

	accountTokens := func(content string) {
		if r.contextWindow != nil && content != "" {
			r.contextWindow.AddTokens(int64(estimateFn(content)))
		}
	}

	accountTokens(skillsSection)
	toolTokens := EstimateTokensForTools(llmTools, estimateFn)
	if toolTokens > 0 && r.contextWindow != nil {
		r.contextWindow.AddTokens(toolTokens)
	}

	var actCtx *ActivatedSkillContext

	if len(skills) > 0 {
		selectInstructions := BuildSkillSelectPrompt(ctx.Input, ctx.Intent, skills)
		capabilitiesSection := BuildCapabilitiesList(skills)
		accountTokens(selectInstructions)
		accountTokens(capabilitiesSection)
		r.checkSlide(ctx.Ctx())

		selectContent, selectTokens, err := r.callLLMStream(
			ctx, selectInstructions, ctx.Input, ctx.ConversationHistory, MaxHistoryTurns, nil, capabilitiesSection,
		)
		if err != nil {
			return totalTokens + selectTokens, fmt.Errorf("think phase1 (skill select) failed: %w", err)
		}
		totalTokens += selectTokens

		selectThought, err := ParseThinkResponse(selectContent)
		if err != nil {
			return totalTokens, fmt.Errorf("think phase1 parse failed: %w", err)
		}

		if selectThought.SelectedSkill != "" {
			actCtx, err = r.ActivateSkill(selectThought.SelectedSkill, toolInfos)
			if err != nil {
				actCtx = nil
			} else if actCtx != nil {
				accountTokens(actCtx.Instructions)
				filteredToolTokens := EstimateTokensForTools(actCtx.FilteredTools, estimateFn)
				if filteredToolTokens > 0 && r.contextWindow != nil {
					r.contextWindow.AddTokens(filteredToolTokens)
				}
				llmTools = actCtx.FilteredTools
			}
		}
	}

	instructions := BuildThinkPrompt(ctx.Input, ctx.Intent, memoryRecords, actCtx)
	accountTokens(instructions)
	r.checkSlide(ctx.Ctx())

	content, tokens, err := r.callLLMStream(ctx, instructions, ctx.Input, ctx.ConversationHistory, MaxHistoryTurns, llmTools, skillsSection)
	if err != nil {
		return totalTokens + tokens, fmt.Errorf("think phase2 (tool plan) failed: %w", err)
	}
	totalTokens += tokens

	thought, err := ParseThinkResponse(content)
	if err != nil {
		return totalTokens, fmt.Errorf("think phase2 parse failed: %w", err)
	}

	if actCtx != nil && thought.SelectedSkill == "" {
		thought.SelectedSkill = actCtx.Skill.Name
	}

	ctx.LastThought = thought
	return totalTokens, nil
}

// Act executes the decision from the Think phase.
func (r *Reactor) Act(ctx *ReactContext) error {
	thought := ctx.LastThought
	if thought == nil {
		return fmt.Errorf("act called without a thought")
	}

	start := time.Now()
	action := Action{
		Timestamp: start,
	}

	switch thought.Decision {
	case DecisionAnswer:
		action.Type = ActionTypeAnswer
		action.Result = thought.FinalAnswer
		if action.Result == "" {
			action.Result = thought.Reasoning
		}

	case DecisionClarify:
		action.Type = ActionTypeClarify
		question := thought.ClarificationQuestion
		if question == "" {
			question = "Could you provide more details so I can better assist you?"
		}
		action.Result = question

	case DecisionAct:
		action.Type = ActionTypeToolCall
		action.Target = thought.ActionTarget
		action.Params = thought.ActionParams

		if action.Target == "" {
			action.Type = ActionTypeAnswer
			action.Result = thought.FinalAnswer
			if action.Result == "" {
				action.Result = "Sorry, I cannot determine which tool to use for your request."
			}
			break
		}

		result, duration, err := r.toolRegistry.ExecuteTool(ctx.Ctx(), action.Target, action.Params)
		if err != nil {
			action.Error = err
			action.ErrorMsg = err.Error()
		} else {
			action.Result = result
		}
		action.Duration = duration

	default:
		action.Type = ActionTypeAnswer
		action.Result = thought.FinalAnswer
		if action.Result == "" {
			action.Result = thought.Reasoning
		}
	}

	ctx.LastAction = &action
	return nil
}

// Observe evaluates the result of the Act phase.
func (r *Reactor) Observe(ctx *ReactContext) error {
	action := ctx.LastAction
	if action == nil {
		return fmt.Errorf("observe called without an action")
	}

	var obs *Observation

	switch action.Type {
	case ActionTypeToolCall:
		if action.Error != nil {
			obs = NewErrorObservation(action.Error, false)
			obs.Insights = []string{fmt.Sprintf("Tool %q execution failed", action.Target)}
		} else {
			insights := analyzeActionResult(action.Result)
			obs = NewSuccessObservation(action.Result, insights...)
		}

	case ActionTypeAnswer:
		obs = NewSuccessObservation(action.Result, "direct answer generated")

	case ActionTypeClarify:
		obs = NewSuccessObservation(action.Result, "clarification question generated")

	default:
		obs = NewSuccessObservation(action.Result)
	}

	ctx.LastObservation = obs
	return nil
}
