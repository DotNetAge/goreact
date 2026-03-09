package skill

import "time"

const (
	// 综合评分权重
	SuccessRateWeight = 0.4
	EfficiencyWeight  = 0.25
	QualityWeight     = 0.25
	FrequencyWeight   = 0.1
)

// Skill 技能数据结构
// 基于 Agent Skills 规范 (https://agentskills.io)
// Skill 是一套完整的工作方案/指令集，包含 SKILL.md 文件和可选的支持文件
type Skill struct {
	// Frontmatter 字段（来自 SKILL.md 的 YAML frontmatter）
	Name          string            // 技能名称（必需，1-64字符，小写字母、数字和连字符）
	Description   string            // 技能描述（必需，1-1024字符，描述功能和使用场景）
	License       string            // 许可证（可选）
	Compatibility string            // 兼容性要求（可选，1-500字符）
	Metadata      map[string]string // 元数据（可选，自定义键值对）
	AllowedTools  []string          // 预批准的工具列表（可选，实验性）

	// Body 内容（来自 SKILL.md 的 Markdown 正文）
	Instructions string // 技能指令（Markdown 格式的分步指令）

	// 可选目录内容
	Scripts    map[string]string // scripts/ 目录中的脚本文件（文件名 -> 内容）
	References map[string]string // references/ 目录中的参考文档（文件名 -> 内容）
	Assets     map[string][]byte // assets/ 目录中的静态资源（文件名 -> 二进制内容）

	// 运行时统计数据（不属于 Skill 定义，由 SkillManager 维护）
	Statistics *SkillStatistics
}

// SkillMetadata 技能元数据（用于列表展示，轻量级）
type SkillMetadata struct {
	Name        string // 技能名称
	Description string // 技能描述
}

// SkillStatistics 技能统计数据
type SkillStatistics struct {
	CreatedAt            time.Time     // 创建时间
	LastUsed             time.Time     // 最后使用时间
	UsageCount           int           // 使用次数
	SuccessCount         int           // 成功次数
	FailureCount         int           // 失败次数
	TotalExecutionTime   time.Duration // 总执行时间
	AverageExecutionTime time.Duration // 平均执行时间
	TotalTokenConsumed   int           // 总 Token 消耗
	QualityScoreSum      float64       // 质量评分总和
	LastEvaluation       time.Time     // 最后评估时间

	// 综合评分指标
	SuccessRate     float64 // 成功率 = 成功次数 / 总执行次数
	EfficiencyScore float64 // 效率评分 = 基准时间 / 实际平均执行时间
	QualityScore    float64 // 质量评分 = 质量评分总和 / 评估次数
	FrequencyScore  float64 // 频率评分 = 使用次数 / 总任务数
	OverallScore    float64 // 综合评分 = 成功率×0.4 + 效率×0.25 + 质量×0.25 + 频率×0.1
}

// SkillRanking 技能排名
type SkillRanking struct {
	SkillName    string  // 技能名称
	OverallScore float64 // 综合评分
	Rank         int     // 排名
	Trend        string  // 趋势（上升/下降/稳定）
}

// NewSkill 创建新的技能
func NewSkill(name, description string) *Skill {
	return &Skill{
		Name:        name,
		Description: description,
		Metadata:    make(map[string]string),
		Scripts:     make(map[string]string),
		References:  make(map[string]string),
		Assets:      make(map[string][]byte),
		Statistics: &SkillStatistics{
			CreatedAt: time.Now(),
		},
	}
}

// GetMetadata 获取技能元数据（轻量级）
func (s *Skill) GetMetadata() SkillMetadata {
	return SkillMetadata{
		Name:        s.Name,
		Description: s.Description,
	}
}

// CalculateOverallScore 计算综合评分
func (s *SkillStatistics) CalculateOverallScore() float64 {
	// 综合评分公式：成功率×0.4 + 效率×0.25 + 质量×0.25 + 频率×0.1
	s.OverallScore = s.SuccessRate*SuccessRateWeight + s.EfficiencyScore*EfficiencyWeight + s.QualityScore*QualityWeight + s.FrequencyScore*FrequencyWeight
	return s.OverallScore
}

// UpdateSuccessRate 更新成功率
func (s *SkillStatistics) UpdateSuccessRate() {
	totalExecutions := s.SuccessCount + s.FailureCount
	if totalExecutions > 0 {
		s.SuccessRate = float64(s.SuccessCount) / float64(totalExecutions)
	}
}

// UpdateAverageExecutionTime 更新平均执行时间
func (s *SkillStatistics) UpdateAverageExecutionTime() {
	totalExecutions := s.SuccessCount + s.FailureCount
	if totalExecutions > 0 {
		s.AverageExecutionTime = s.TotalExecutionTime / time.Duration(totalExecutions)
	}
}

// UpdateQualityScore 更新质量评分
func (s *SkillStatistics) UpdateQualityScore() {
	evaluationCount := s.SuccessCount + s.FailureCount
	if evaluationCount > 0 {
		s.QualityScore = s.QualityScoreSum / float64(evaluationCount)
	}
}
