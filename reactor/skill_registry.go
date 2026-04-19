package reactor

import (
	"fmt"
	"strings"
	"sync"
	"github.com/DotNetAge/goreact/core"
)

type defaultSkillRegistry struct {
	mu     sync.RWMutex
	skills map[string]*core.Skill
}

func NewSkillRegistry() core.SkillRegistry {
	return &defaultSkillRegistry{
		skills: make(map[string]*core.Skill),
	}
}

func (r *defaultSkillRegistry) RegisterSkill(skill *core.Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if skill.Name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}
	r.skills[skill.Name] = skill
	return nil
}

func (r *defaultSkillRegistry) GetSkill(name string) (*core.Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	skill, ok := r.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", name)
	}
	return skill, nil
}

// FindApplicableSkills finds skills whose trigger rules match the given intent context.
// The context parameter should be a *reactor.Intent; other types are silently ignored.
func (r *defaultSkillRegistry) FindApplicableSkills(context any) ([]*core.Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	intent, isIntent := context.(*Intent)
	if !isIntent || intent == nil {
		return nil, nil
	}

	intentType := strings.ToLower(strings.TrimSpace(string(intent.Type)))
	topic := strings.ToLower(strings.TrimSpace(intent.Topic))
	summary := strings.ToLower(strings.TrimSpace(intent.Summary))

	var entityParts []string
	for k, v := range intent.Entities {
		entityParts = append(entityParts, strings.ToLower(strings.TrimSpace(k)))
		entityParts = append(entityParts, strings.ToLower(strings.TrimSpace(fmt.Sprint(v))))
	}
	entityBlob := strings.Join(entityParts, " ")

	var applicable []*core.Skill
	for _, skill := range r.skills {
		matched := false
		for _, rule := range skill.TriggerRules {
			r := strings.ToLower(strings.TrimSpace(rule))
			if r == "" {
				continue
			}
			// 先做精确匹配，再做包含匹配，提升触发稳定性
			if r == intentType || r == topic ||
				strings.Contains(topic, r) ||
				strings.Contains(summary, r) ||
				strings.Contains(entityBlob, r) {
				matched = true
				break
			}
		}
		if matched {
			applicable = append(applicable, skill)
		}
	}
	return applicable, nil
}
