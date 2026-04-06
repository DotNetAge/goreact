package prompt

import (
	"strings"
)

// QuestionAnalyzer analyzes question types
type QuestionAnalyzer struct {
	patterns map[QuestionType][]string
}

// NewQuestionAnalyzer creates a new QuestionAnalyzer
func NewQuestionAnalyzer() *QuestionAnalyzer {
	return &QuestionAnalyzer{
		patterns: map[QuestionType][]string{
			QuestionTypeFactual: {
				"是什么", "什么是", "多少", "哪个", "谁", "where", "what", "who", "how many",
			},
			QuestionTypeProcedural: {
				"如何", "怎么", "怎样", "步骤", "流程", "how to", "steps", "procedure",
			},
			QuestionTypeAnalytical: {
				"为什么", "原因", "分析", "比较", "区别", "why", "analyze", "compare", "difference",
			},
			QuestionTypeCreative: {
				"设计", "创造", "想象", "建议", "创意", "design", "create", "imagine", "suggest",
			},
		},
	}
}

// Analyze analyzes the question and returns its type
func (a *QuestionAnalyzer) Analyze(question string) QuestionType {
	questionLower := strings.ToLower(question)

	for qType, patterns := range a.patterns {
		for _, pattern := range patterns {
			if strings.Contains(questionLower, strings.ToLower(pattern)) {
				return qType
			}
		}
	}

	return QuestionTypeFactual
}

// ShouldInjectRAG determines if RAG context should be injected
func (a *QuestionAnalyzer) ShouldInjectRAG(question string) bool {
	qType := a.Analyze(question)
	return qType == QuestionTypeFactual || qType == QuestionTypeAnalytical
}

// GetRecommendedStrategy returns the recommended injection strategy
func (a *QuestionAnalyzer) GetRecommendedStrategy(question string) InjectionStrategy {
	qType := a.Analyze(question)

	switch qType {
	case QuestionTypeFactual:
		return InjectionSuffix // Just before question
	case QuestionTypeAnalytical:
		return InjectionInfix // Before examples for reference
	case QuestionTypeProcedural:
		return InjectionPrefix // As background knowledge
	default:
		return InjectionDynamic
	}
}

// GetDifficulty estimates question difficulty
func (a *QuestionAnalyzer) GetDifficulty(question string) int {
	qType := a.Analyze(question)

	switch qType {
	case QuestionTypeFactual:
		return 1
	case QuestionTypeProcedural:
		return 2
	case QuestionTypeAnalytical:
		return 3
	case QuestionTypeCreative:
		return 4
	default:
		return 2
	}
}

// GetRecommendedExampleCount returns recommended number of examples
func (a *QuestionAnalyzer) GetRecommendedExampleCount(question string) int {
	difficulty := a.GetDifficulty(question)
	return min(difficulty+1, 3)
}
