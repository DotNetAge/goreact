package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/ray/goreact/pkg/llm"
)

// SelectionMethod 表示 Agent 选择的方式
type SelectionMethod string

const (
	SelectionSingle   SelectionMethod = "single"   // 仅有一个 Agent
	SelectionKeyword  SelectionMethod = "keyword"  // 关键词匹配
	SelectionSemantic SelectionMethod = "semantic" // 语义匹配
	SelectionFallback SelectionMethod = "fallback" // 回退到默认

	minKeywordLength      = 3   // 关键词最小长度
	descriptionMatchScore = 2.0 // 描述匹配分数
	nameMatchScore        = 3.0 // 名称匹配分数
	reverseMatchScore     = 1.0 // 反向匹配分数
	maxCandidates         = 3   // 最大候选数量
)

// SelectionResult Agent 选择结果
type SelectionResult struct {
	Agent  *Agent
	Method SelectionMethod
	Score  float64 // 匹配分数（关键词匹配时有效）
}

// Manager 智能体管理器
type Manager struct {
	agents    map[string]*Agent
	mutex     sync.RWMutex
	llmClient llm.Client // 用于语义匹配（可选）
}

// NewManager 创建智能体管理器
func NewManager() *Manager {
	return &Manager{
		agents: make(map[string]*Agent),
	}
}

// WithLLMClient 设置 LLM 客户端（用于语义匹配）
func (m *Manager) WithLLMClient(client llm.Client) *Manager {
	m.llmClient = client
	return m
}

// Register 注册智能体
func (m *Manager) Register(agent *Agent) error {
	if agent.Name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.agents[agent.Name] = agent
	return nil
}

// Get 获取智能体
func (m *Manager) Get(name string) (*Agent, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	agent, exists := m.agents[name]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", name)
	}
	return agent, nil
}

// List 列出所有智能体
func (m *Manager) List() []*Agent {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	agents := make([]*Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	return agents
}

// SelectAgent 根据任务选择最合适的 Agent
// 使用关键词匹配 + 语义匹配（如果有 LLM）
// 返回选择结果，包含选择方式和匹配分数
func (m *Manager) SelectAgent(task string) (*Agent, error) {
	result, err := m.SelectAgentWithResult(task)
	if err != nil {
		return nil, err
	}
	return result.Agent, nil
}

// SelectAgentWithResult 根据任务选择最合适的 Agent，返回详细的选择结果
func (m *Manager) SelectAgentWithResult(task string) (*SelectionResult, error) {
	m.mutex.RLock()

	if len(m.agents) == 0 {
		m.mutex.RUnlock()
		return nil, fmt.Errorf("no agents available")
	}

	// 如果只有一个 Agent，直接返回
	if len(m.agents) == 1 {
		for _, agent := range m.agents {
			m.mutex.RUnlock()
			return &SelectionResult{Agent: agent, Method: SelectionSingle}, nil
		}
	}

	// 1. 关键词匹配筛选候选
	candidates := m.filterByKeywords(task)
	m.mutex.RUnlock()

	// 2. 如果没有候选，返回第一个 Agent（明确标记为 fallback）
	if len(candidates) == 0 {
		m.mutex.RLock()
		for _, agent := range m.agents {
			m.mutex.RUnlock()
			return &SelectionResult{Agent: agent, Method: SelectionFallback}, nil
		}
		m.mutex.RUnlock()
	}

	// 3. 如果只有一个候选，直接返回
	if len(candidates) == 1 {
		return &SelectionResult{
			Agent:  candidates[0].agent,
			Method: SelectionKeyword,
			Score:  candidates[0].score,
		}, nil
	}

	// 4. 如果有多个候选且有 LLM，使用语义匹配（不持有锁）
	if len(candidates) > 1 && m.llmClient != nil {
		agents := make([]*Agent, len(candidates))
		for i, c := range candidates {
			agents[i] = c.agent
		}
		selected, err := m.selectBySemantic(task, agents)
		if err == nil {
			return &SelectionResult{Agent: selected, Method: SelectionSemantic}, nil
		}
	}

	// 5. 否则返回关键词评分最高的（第一个候选）
	return &SelectionResult{
		Agent:  candidates[0].agent,
		Method: SelectionKeyword,
		Score:  candidates[0].score,
	}, nil
}

type scoredAgent struct {
	agent *Agent
	score float64
}

// filterByKeywords 使用关键词匹配筛选候选 Agent
func (m *Manager) filterByKeywords(task string) []scoredAgent {
	scored := make([]scoredAgent, 0, len(m.agents))
	taskLower := strings.ToLower(task)

	for _, agent := range m.agents {
		score := 0.0
		descLower := strings.ToLower(agent.Description)
		nameLower := strings.ToLower(agent.Name)

		// 检查任务中的关键词是否匹配 Agent 描述
		taskWords := strings.Fields(taskLower)
		for _, taskWord := range taskWords {
			if len(taskWord) < minKeywordLength {
				continue
			}
			if strings.Contains(descLower, taskWord) {
				score += descriptionMatchScore
			}
			if strings.Contains(nameLower, taskWord) {
				score += nameMatchScore
			}
		}

		// 检查 Agent 描述中的关键词是否出现在任务中
		descWords := strings.Fields(descLower)
		for _, descWord := range descWords {
			if len(descWord) > minKeywordLength && strings.Contains(taskLower, descWord) {
				score += reverseMatchScore
			}
		}

		if score > 0 {
			scored = append(scored, scoredAgent{agent: agent, score: score})
		}
	}

	// 按分数排序（使用 sort.Slice 替代冒泡排序）
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 返回前 maxCandidates 个候选
	if len(scored) > maxCandidates {
		scored = scored[:maxCandidates]
	}

	return scored
}

// selectBySemantic 使用 LLM 进行语义匹配选择
func (m *Manager) selectBySemantic(task string, candidates []*Agent) (*Agent, error) {
	if m.llmClient == nil || len(candidates) == 0 {
		return nil, fmt.Errorf("cannot perform semantic selection")
	}

	if len(candidates) == 1 {
		return candidates[0], nil
	}

	// 构建 prompt
	var sb strings.Builder
	for i, agent := range candidates {
		fmt.Fprintf(&sb, "%d. %s: %s\n", i+1, agent.Name, agent.Description)
	}

	prompt := fmt.Sprintf(`You are an agent selection assistant. Given a task and a list of available agents, select the most suitable agent.

Task: "%s"

Available agents:
%s

Instructions:
- Analyze the task requirements carefully
- Compare each agent's description with the task
- Select the agent that best matches the task's needs
- Return ONLY the agent name (e.g., "math-expert"), nothing else

Your selection:`, task, sb.String())

	// 调用 LLM
	response, err := m.llmClient.Generate(context.Background(), prompt)
	if err != nil {
		// 如果 LLM 调用失败，返回第一个候选
		return candidates[0], nil
	}

	// 解析响应，提取 Agent 名称
	selectedName := strings.TrimSpace(response)
	selectedName = strings.Trim(selectedName, "\"'`")

	// 在候选中查找匹配的 Agent
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(selectedName), strings.ToLower(candidate.Name)) ||
			strings.Contains(strings.ToLower(candidate.Name), strings.ToLower(selectedName)) {
			return candidate, nil
		}
	}

	// 如果 LLM 返回的名称无法匹配，返回第一个候选
	return candidates[0], nil
}
