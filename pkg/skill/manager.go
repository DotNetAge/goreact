package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

	// ArchiveSkill 归档技能（软淘汰）
	ArchiveSkill(name string) error

	// RestoreSkill 恢复归档的技能
	RestoreSkill(name string) error
}

// DefaultManager 默认技能管理器实现
type DefaultManager struct {
	skills         map[string]*Skill // 活跃技能库
	archivedSkills map[string]*Skill // 归档技能库
	totalTasks     int               // 总任务数（用于计算频率评分）
}

// NewDefaultManager 创建新的默认技能管理器
func NewDefaultManager() *DefaultManager {
	return &DefaultManager{
		skills:         make(map[string]*Skill),
		archivedSkills: make(map[string]*Skill),
		totalTasks:     0,
	}
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
func (m *DefaultManager) LoadSkill(path string) (*Skill, error) {
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
func (m *DefaultManager) loadOptionalDirectories(basePath string, skill *Skill) {
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
func (m *DefaultManager) RegisterSkill(skill *Skill) error {
	if skill.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	m.skills[skill.Name] = skill
	return nil
}

// GetSkill 获取技能
func (m *DefaultManager) GetSkill(name string) (*Skill, error) {
	skill, ok := m.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", name)
	}
	return skill, nil
}

// ListSkills 列出所有技能元数据（轻量级）
func (m *DefaultManager) ListSkills() []SkillMetadata {
	metadata := make([]SkillMetadata, 0, len(m.skills))
	for _, skill := range m.skills {
		metadata = append(metadata, skill.GetMetadata())
	}
	return metadata
}

// SelectSkill 根据任务选择最合适的技能
func (m *DefaultManager) SelectSkill(task string) (*Skill, error) {
	taskLower := strings.ToLower(task)
	var bestSkill *Skill
	bestScore := 0.0

	for _, skill := range m.skills {
		// 简单的关键词匹配评分
		score := 0.0
		descLower := strings.ToLower(skill.Description)

		// 检查描述中的关键词
		words := strings.Fields(descLower)
		for _, word := range words {
			if strings.Contains(taskLower, word) {
				score += 1.0
			}
		}

		// 考虑技能的综合评分
		if skill.Statistics != nil {
			score += skill.Statistics.OverallScore * 10
		}

		if score > bestScore {
			bestScore = score
			bestSkill = skill
		}
	}

	if bestSkill == nil {
		return nil, fmt.Errorf("no suitable skill found for task: %s", task)
	}

	return bestSkill, nil
}

// RecordExecution 记录技能执行结果
func (m *DefaultManager) RecordExecution(name string, success bool, executionTime time.Duration, tokenConsumed int, qualityScore float64) error {
	skill, err := m.GetSkill(name)
	if err != nil {
		return err
	}

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
func (m *DefaultManager) GetSkillStatistics(name string) (*SkillStatistics, error) {
	skill, err := m.GetSkill(name)
	if err != nil {
		return nil, err
	}
	return skill.Statistics, nil
}

// GetSkillRanking 获取技能排名
func (m *DefaultManager) GetSkillRanking() []SkillRanking {
	rankings := make([]SkillRanking, 0, len(m.skills))

	for _, skill := range m.skills {
		rankings = append(rankings, SkillRanking{
			SkillName:    skill.Name,
			OverallScore: skill.Statistics.OverallScore,
		})
	}

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
func (m *DefaultManager) EvolveSkills() error {
	now := time.Now()

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
			if err := m.ArchiveSkill(name); err != nil {
				return fmt.Errorf("failed to archive skill %s: %w", name, err)
			}
		}
	}

	return nil
}

// ArchiveSkill 归档技能（软淘汰）
func (m *DefaultManager) ArchiveSkill(name string) error {
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
func (m *DefaultManager) RestoreSkill(name string) error {
	skill, ok := m.archivedSkills[name]
	if !ok {
		return fmt.Errorf("archived skill not found: %s", name)
	}

	// 恢复到活跃库
	m.skills[name] = skill
	delete(m.archivedSkills, name)

	return nil
}
