package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	gochatcore "github.com/DotNetAge/gochat/pkg/core"
	"gopkg.in/yaml.v3"
)

// Manager 技能管理器接口
type Manager interface {
	// LoadSkill 从目录加载技能
	LoadSkill(path string) (*Skill, error)

	// RegisterSkill 注册技能
	RegisterSkill(skill *Skill) error

	// GetSkill 获取技能
	GetSkill(name string) (*Skill, error)

	// ListSkills 列出所有技能元数据（轻量级）
	ListSkills() []SkillMetadata

	// SelectSkill 根据任务选择最合适的技能
	SelectSkill(task string) (*Skill, error)

	// RecordExecution 记录技能执行结果
	RecordExecution(name string, success bool, executionTime time.Duration, tokenConsumed int, qualityScore float64) error

	// GetSkillStatistics 获取技能统计数据
	GetSkillStatistics(name string) (*SkillStatistics, error)

	// GetSkillRanking 获取技能排名
	GetSkillRanking() []SkillRanking

	// EvolveSkills 执行技能进化（优胜劣汰）
	EvolveSkills() error

	// StartEvolutionScheduler 启动定时演化调度器
	StartEvolutionScheduler(ctx context.Context, interval time.Duration)

	// ArchiveSkill 归档技能（软淘汰）
	ArchiveSkill(name string) error

	// RestoreSkill 恢复归档的技能
	RestoreSkill(name string) error
}

// SelectionMode 技能选择模式
type SelectionMode int

const (
	// KeywordOnly 仅使用关键词匹配（快速但不够精确）
	KeywordOnly SelectionMode = iota
	// SemanticOnly 仅使用语义匹配（精确但较慢）
	SemanticOnly
	// Hybrid 混合模式：先关键词筛选，再语义选择（推荐）
	Hybrid
)

// defaultMgr 默认技能管理器实现
type defaultMgr struct {
	skills         map[string]*Skill // 活跃技能库
	archivedSkills map[string]*Skill // 归档技能库
	totalTasks     int               // 总任务数（用于计算频率评分）
	llmClient      gochatcore.Client // LLM 客户端（用于语义匹配）
	selectionMode  SelectionMode     // 选择模式
	topN           int               // 混合模式下筛选的候选数量
	mu             sync.RWMutex      // 保护并发访问
}

// ManagerOption 管理器配置选项
type ManagerOption func(*defaultMgr)

// WithLLMClient 设置 LLM 客户端（用于语义匹配）
func WithLLMClient(client gochatcore.Client) ManagerOption {
	return func(m *defaultMgr) {
		m.llmClient = client
	}
}

// WithSelectionMode 设置选择模式
func WithSelectionMode(mode SelectionMode) ManagerOption {
	return func(m *defaultMgr) {
		m.selectionMode = mode
	}
}

// WithTopN 设置混合模式下的候选数量
func WithTopN(n int) ManagerOption {
	return func(m *defaultMgr) {
		m.topN = n
	}
}

// DefaultManager 创建新的默认技能管理器
func DefaultManager(options ...ManagerOption) *defaultMgr {
	m := &defaultMgr{
		skills:         make(map[string]*Skill),
		archivedSkills: make(map[string]*Skill),
		totalTasks:     0,
		selectionMode:  Hybrid, // 默认使用混合模式
		topN:           3,      // 默认筛选前3个候选
	}

	// 应用选项
	for _, opt := range options {
		opt(m)
	}

	return m
}

// SkillFrontmatter SKILL.md 的 YAML frontmatter 结构
type SkillFrontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty"`
}

// LoadSkill 从目录加载技能
func (m *defaultMgr) LoadSkill(path string) (*Skill, error) {
	// 读取 SKILL.md 文件
	skillMdPath := filepath.Join(path, "SKILL.md")
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	// 解析 frontmatter 和 body
	parts := strings.SplitN(string(content), "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid SKILL.md format: missing frontmatter")
	}

	// 解析 YAML frontmatter
	var frontmatter SkillFrontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &frontmatter); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// 验证必需字段
	if frontmatter.Name == "" {
		return nil, fmt.Errorf("skill name is required")
	}
	if frontmatter.Description == "" {
		return nil, fmt.Errorf("skill description is required")
	}

	// 创建技能对象
	skill := &Skill{
		Name:          frontmatter.Name,
		Description:   frontmatter.Description,
		License:       frontmatter.License,
		Compatibility: frontmatter.Compatibility,
		Metadata:      frontmatter.Metadata,
		Instructions:  strings.TrimSpace(parts[2]),
		Scripts:       make(map[string]string),
		References:    make(map[string]string),
		Assets:        make(map[string][]byte),
		Statistics: &SkillStatistics{
			CreatedAt: time.Now(),
		},
	}

	// 解析 allowed-tools
	if frontmatter.AllowedTools != "" {
		skill.AllowedTools = strings.Fields(frontmatter.AllowedTools)
	}

	// 加载可选目录
	m.loadOptionalDirectories(path, skill)

	return skill, nil
}

// loadOptionalDirectories 加载可选目录（scripts, references, assets）
func (m *defaultMgr) loadOptionalDirectories(basePath string, skill *Skill) {
	// 加载 scripts/
	scriptsPath := filepath.Join(basePath, "scripts")
	if entries, err := os.ReadDir(scriptsPath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				filePath := filepath.Join(scriptsPath, entry.Name())
				if content, err := os.ReadFile(filePath); err == nil {
					skill.Scripts[entry.Name()] = string(content)
				}
			}
		}
	}

	// 加载 references/
	referencesPath := filepath.Join(basePath, "references")
	if entries, err := os.ReadDir(referencesPath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				filePath := filepath.Join(referencesPath, entry.Name())
				if content, err := os.ReadFile(filePath); err == nil {
					skill.References[entry.Name()] = string(content)
				}
			}
		}
	}

	// 加载 assets/
	assetsPath := filepath.Join(basePath, "assets")
	if entries, err := os.ReadDir(assetsPath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				filePath := filepath.Join(assetsPath, entry.Name())
				if content, err := os.ReadFile(filePath); err == nil {
					skill.Assets[entry.Name()] = content
				}
			}
		}
	}
}

// RegisterSkill 注册技能
func (m *defaultMgr) RegisterSkill(skill *Skill) error {
	// 验证技能数据
	if err := validateSkill(skill); err != nil {
		return fmt.Errorf("invalid skill: %w", err)
	}
	m.mu.Lock()
	m.skills[skill.Name] = skill
	m.mu.Unlock()
	return nil
}

// validateSkill 验证技能数据
func validateSkill(skill *Skill) error {
	if skill == nil {
		return fmt.Errorf("skill cannot be nil")
	}

	// 验证名称（必需，1-64字符，小写字母、数字和连字符）
	if skill.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if len(skill.Name) > 64 {
		return fmt.Errorf("skill name too long (max 64 characters): %s", skill.Name)
	}
	for _, ch := range skill.Name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return fmt.Errorf("skill name must contain only lowercase letters, numbers, and hyphens: %s", skill.Name)
		}
	}

	// 验证描述（必需，1-1024字符）
	if skill.Description == "" {
		return fmt.Errorf("skill description is required")
	}
	if len(skill.Description) > 1024 {
		return fmt.Errorf("skill description too long (max 1024 characters)")
	}

	// 验证兼容性字段（可选，最多500字符）
	if len(skill.Compatibility) > 500 {
		return fmt.Errorf("skill compatibility field too long (max 500 characters)")
	}

	// 验证指令内容（必需）
	if strings.TrimSpace(skill.Instructions) == "" {
		return fmt.Errorf("skill instructions are required")
	}

	return nil
}

// GetSkill 获取技能
func (m *defaultMgr) GetSkill(name string) (*Skill, error) {
	m.mu.RLock()
	skill, ok := m.skills[name]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", name)
	}
	return skill, nil
}

// ListSkills 列出所有技能元数据（轻量级）
func (m *defaultMgr) ListSkills() []SkillMetadata {
	m.mu.RLock()
	metadata := make([]SkillMetadata, 0, len(m.skills))
	for _, skill := range m.skills {
		metadata = append(metadata, skill.GetMetadata())
	}
	m.mu.RUnlock()
	return metadata
}

// SelectSkill 根据任务选择最合适的技能（混合模式）
func (m *defaultMgr) SelectSkill(task string) (*Skill, error) {
	m.mu.RLock()
	skillCount := len(m.skills)
	m.mu.RUnlock()

	if skillCount == 0 {
		return nil, fmt.Errorf("no skills available")
	}

	switch m.selectionMode {
	case KeywordOnly:
		return m.selectByKeyword(task)
	case SemanticOnly:
		if m.llmClient == nil {
			return nil, fmt.Errorf("LLM client is required for semantic selection")
		}
		return m.selectBySemantic(task, m.getAllSkills())
	case Hybrid:
		return m.selectHybrid(task)
	default:
		return m.selectByKeyword(task)
	}
}

// selectHybrid 混合选择：先关键词筛选，再语义匹配
func (m *defaultMgr) selectHybrid(task string) (*Skill, error) {
	// 1. 使用关键词快速筛选候选（取前 topN 个）
	candidates, err := m.filterCandidatesByKeyword(task, m.topN)
	if err != nil {
		return nil, err
	}

	// 2. 如果只有一个候选，直接返回
	if len(candidates) == 1 {
		return candidates[0], nil
	}

	// 3. 如果有多个候选且有 LLM，使用语义匹配精确选择
	if len(candidates) > 1 && m.llmClient != nil {
		selected, err := m.selectBySemantic(task, candidates)
		if err == nil {
			return selected, nil
		}
		// 如果语义匹配失败，降级到返回关键词评分最高的
		return candidates[0], nil
	}

	// 4. 没有 LLM 或语义匹配失败，返回关键词评分最高的
	return candidates[0], nil
}

// filterCandidatesByKeyword 使用关键词筛选候选技能
func (m *defaultMgr) filterCandidatesByKeyword(task string, topN int) ([]*Skill, error) {
	type scoredSkill struct {
		skill *Skill
		score float64
	}

	m.mu.RLock()
	scored := make([]scoredSkill, 0, len(m.skills))

	// 计算每个技能的关键词匹配分数
	for _, skill := range m.skills {
		score := m.calculateKeywordScore(task, skill)
		scored = append(scored, scoredSkill{skill: skill, score: score})
	}
	m.mu.RUnlock()

	// 按分数排序（冒泡排序，简单实现）
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// 取前 topN 个候选
	n := topN
	if n > len(scored) {
		n = len(scored)
	}

	// 过滤掉分数太低的（< 1.0）
	candidates := make([]*Skill, 0, n)
	for i := 0; i < n; i++ {
		if scored[i].score >= 1.0 {
			candidates = append(candidates, scored[i].skill)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no suitable skill found for task: %s", task)
	}

	return candidates, nil
}

// selectByKeyword 仅使用关键词匹配选择技能
func (m *defaultMgr) selectByKeyword(task string) (*Skill, error) {
	candidates, err := m.filterCandidatesByKeyword(task, 1)
	if err != nil {
		return nil, err
	}
	return candidates[0], nil
}

// calculateKeywordScore 计算关键词匹配分数
func (m *defaultMgr) calculateKeywordScore(task string, skill *Skill) float64 {
	taskLower := strings.ToLower(task)
	descLower := strings.ToLower(skill.Description)
	nameLower := strings.ToLower(skill.Name)

	score := 0.0

	// 定义关键词映射
	keywordMap := map[string][]string{
		"calculate":  {"math", "calculation", "calculator", "mathematical", "compute"},
		"math":       {"math", "calculation", "calculator", "mathematical", "compute"},
		"add":        {"math", "calculation", "calculator", "mathematical"},
		"subtract":   {"math", "calculation", "calculator", "mathematical"},
		"multiply":   {"math", "calculation", "calculator", "mathematical"},
		"divide":     {"math", "calculation", "calculator", "mathematical"},
		"sum":        {"math", "calculation", "calculator", "mathematical"},
		"compute":    {"math", "calculation", "calculator", "mathematical", "compute"},
		"number":     {"math", "calculation", "calculator", "mathematical"},
		"equation":   {"math", "calculation", "calculator", "mathematical"},
		"expression": {"math", "calculation", "calculator", "mathematical"},
	}

	// 1. 检查任务中的关键词是否匹配技能描述
	taskWords := strings.Fields(taskLower)
	for _, taskWord := range taskWords {
		// 直接匹配
		if strings.Contains(descLower, taskWord) {
			score += 2.0
		}
		if strings.Contains(nameLower, taskWord) {
			score += 3.0
		}

		// 通过关键词映射匹配
		if relatedWords, ok := keywordMap[taskWord]; ok {
			for _, relatedWord := range relatedWords {
				if strings.Contains(descLower, relatedWord) {
					score += 1.5
				}
				if strings.Contains(nameLower, relatedWord) {
					score += 2.0
				}
			}
		}
	}

	// 2. 检查技能描述中的关键词是否出现在任务中
	descWords := strings.Fields(descLower)
	for _, descWord := range descWords {
		if len(descWord) > 3 && strings.Contains(taskLower, descWord) {
			score += 1.0
		}
	}

	// 3. 考虑技能的综合评分（历史表现）
	if skill.Statistics != nil {
		score += skill.Statistics.OverallScore * 10
	}

	return score
}

// selectBySemantic 使用 LLM 进行语义匹配选择
func (m *defaultMgr) selectBySemantic(task string, candidates []*Skill) (*Skill, error) {
	if m.llmClient == nil {
		return nil, fmt.Errorf("LLM client is not configured")
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates provided")
	}

	if len(candidates) == 1 {
		return candidates[0], nil
	}

	// 构建 prompt
	skillList := ""
	for i, skill := range candidates {
		skillList += fmt.Sprintf("%d. %s: %s\n", i+1, skill.Name, skill.Description)
	}

	prompt := fmt.Sprintf(`You are a skill selection assistant. Given a task and a list of available skills, select the most suitable skill.

Task: "%s"

Available skills:
%s

Instructions:
- Analyze the task requirements carefully
- Compare each skill's description with the task
- Select the skill that best matches the task's needs
- Return ONLY the skill name (e.g., "math-wizard"), nothing else

Your selection:`, task, skillList)

	// 调用 LLM
	messages := []gochatcore.Message{
		gochatcore.NewUserMessage(prompt),
	}
	response, err := m.llmClient.Chat(context.Background(), messages)
	if err != nil {
		return nil, fmt.Errorf("LLM selection failed: %w", err)
	}

	// 解析响应，提取技能名称
	selectedName := strings.TrimSpace(response.Content)
	selectedName = strings.Trim(selectedName, "\"'`")

	// 在候选中查找匹配的技能
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(selectedName), strings.ToLower(candidate.Name)) ||
			strings.Contains(strings.ToLower(candidate.Name), strings.ToLower(selectedName)) {
			return candidate, nil
		}
	}

	// 如果 LLM 返回的名称无法匹配，返回第一个候选（关键词评分最高的）
	return candidates[0], nil
}

// getAllSkills 获取所有技能列表
func (m *defaultMgr) getAllSkills() []*Skill {
	m.mu.RLock()
	skills := make([]*Skill, 0, len(m.skills))
	for _, skill := range m.skills {
		skills = append(skills, skill)
	}
	m.mu.RUnlock()
	return skills
}

// RecordExecution 记录技能执行结果
func (m *defaultMgr) RecordExecution(name string, success bool, executionTime time.Duration, tokenConsumed int, qualityScore float64) error {
	skill, err := m.GetSkill(name)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	stats := skill.Statistics
	stats.LastUsed = time.Now()
	stats.UsageCount++
	m.totalTasks++

	if success {
		stats.SuccessCount++
	} else {
		stats.FailureCount++
	}

	stats.TotalExecutionTime += executionTime
	stats.TotalTokenConsumed += tokenConsumed
	stats.QualityScoreSum += qualityScore

	// 更新各项评分
	stats.UpdateSuccessRate()
	stats.UpdateAverageExecutionTime()
	stats.UpdateQualityScore()

	// 计算效率评分（假设基准时间为 1 秒）
	baselineTime := 1 * time.Second
	if stats.AverageExecutionTime > 0 {
		stats.EfficiencyScore = float64(baselineTime) / float64(stats.AverageExecutionTime)
		if stats.EfficiencyScore > 1.0 {
			stats.EfficiencyScore = 1.0
		}
	}

	// 计算频率评分
	if m.totalTasks > 0 {
		stats.FrequencyScore = float64(stats.UsageCount) / float64(m.totalTasks)
	}

	// 计算综合评分
	stats.CalculateOverallScore()
	stats.LastEvaluation = time.Now()

	return nil
}

// GetSkillStatistics 获取技能统计数据
func (m *defaultMgr) GetSkillStatistics(name string) (*SkillStatistics, error) {
	skill, err := m.GetSkill(name)
	if err != nil {
		return nil, err
	}
	return skill.Statistics, nil
}

// GetSkillRanking 获取技能排名
func (m *defaultMgr) GetSkillRanking() []SkillRanking {
	m.mu.RLock()
	rankings := make([]SkillRanking, 0, len(m.skills))

	for _, skill := range m.skills {
		rankings = append(rankings, SkillRanking{
			SkillName:    skill.Name,
			OverallScore: skill.Statistics.OverallScore,
		})
	}
	m.mu.RUnlock()

	// 按评分排序
	for i := 0; i < len(rankings); i++ {
		for j := i + 1; j < len(rankings); j++ {
			if rankings[j].OverallScore > rankings[i].OverallScore {
				rankings[i], rankings[j] = rankings[j], rankings[i]
			}
		}
	}

	// 设置排名
	for i := range rankings {
		rankings[i].Rank = i + 1
	}

	return rankings
}

// EvolveSkills 执行技能进化（优胜劣汰）
func (m *defaultMgr) EvolveSkills() error {
	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	for name, skill := range m.skills {
		stats := skill.Statistics
		score := stats.OverallScore

		// 淘汰条件检查
		shouldArchive := false

		// 条件 1：连续 30 天无使用且评分低于 0.4
		if now.Sub(stats.LastUsed) > 30*24*time.Hour && score < 0.4 {
			shouldArchive = true
		}

		// 条件 2：评分低于 0.4（需要额外的连续评估逻辑，这里简化处理）
		if score < 0.4 {
			shouldArchive = true
		}

		if shouldArchive {
			// 直接归档，不调用 ArchiveSkill 以避免死锁
			m.archivedSkills[name] = skill
			delete(m.skills, name)
		}
	}

	return nil
}

// StartEvolutionScheduler 启动定时演化调度器
func (m *defaultMgr) StartEvolutionScheduler(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = m.EvolveSkills()
			}
		}
	}()
}

// ArchiveSkill 归档技能（软淘汰）
func (m *defaultMgr) ArchiveSkill(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	skill, ok := m.skills[name]
	if !ok {
		return fmt.Errorf("skill not found: %s", name)
	}

	// 移至归档库
	m.archivedSkills[name] = skill
	delete(m.skills, name)

	return nil
}

// RestoreSkill 恢复归档的技能
func (m *defaultMgr) RestoreSkill(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	skill, ok := m.archivedSkills[name]
	if !ok {
		return fmt.Errorf("archived skill not found: %s", name)
	}

	// 恢复到活跃库
	m.skills[name] = skill
	delete(m.archivedSkills, name)

	return nil
}
