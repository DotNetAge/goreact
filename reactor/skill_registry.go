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

// NewSkillRegistry creates a new empty skill registry.
func NewSkillRegistry() core.SkillRegistry {
	return &defaultSkillRegistry{
		skills: make(map[string]*core.Skill),
	}
}

func (r *defaultSkillRegistry) RegisterSkill(skill *core.Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if skill == nil || skill.Name == "" {
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
		return nil, core.ErrSkillNotFound
	}
	return skill, nil
}

// ListSkills returns all registered skills. Returns a copy to avoid data races.
func (r *defaultSkillRegistry) ListSkills() []*core.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*core.Skill, 0, len(r.skills))
	for _, s := range r.skills {
		result = append(result, s)
	}
	return result
}

// FindApplicableSkills finds skills whose description matches the given intent context.
// The matching is done by checking if any keyword from the skill's description or name
// appears in the intent's type, topic, summary, or entity blob.
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

	// Combine all intent text for matching
	intentText := strings.Join([]string{intentType, topic, summary, entityBlob}, " ")

	var applicable []*core.Skill
	for _, skill := range r.skills {
		if matchSkill(skill, intentText) {
			applicable = append(applicable, skill)
		}
	}
	return applicable, nil
}

// matchSkill checks if a skill is relevant to the given intent text.
// It extracts keywords from the intent text and checks if they appear in the
// skill's description or name. This ensures partial matches work correctly
// (e.g., intent keyword "debug" matches skill description containing "debugging").
func matchSkill(skill *core.Skill, intentText string) bool {
	skillText := strings.ToLower(skill.Name + " " + skill.Description)

	// Extract keywords from intent text
	intentKeywords := extractKeywords(intentText)
	if len(intentKeywords) == 0 {
		return false
	}

	// Match if any significant intent keyword (>= 3 chars) appears in skill text
	matched := 0
	for _, word := range intentKeywords {
		if len(word) >= 3 && strings.Contains(skillText, word) {
			matched++
		}
	}

	// Require at least one keyword match
	return matched >= 1
}

// extractKeywords splits text into lowercase words, filtering common stop words.
func extractKeywords(text string) []string {
	words := strings.Fields(text)
	var keywords []string
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "shall": true, "can": true,
		"to": true, "of": true, "in": true, "for": true, "on": true,
		"with": true, "at": true, "by": true, "from": true, "as": true,
		"into": true, "through": true, "during": true, "before": true,
		"after": true, "above": true, "below": true, "between": true,
		"out": true, "off": true, "over": true, "under": true, "again": true,
		"further": true, "then": true, "once": true, "here": true,
		"there": true, "when": true, "where": true, "why": true, "how": true,
		"all": true, "each": true, "every": true, "both": true, "few": true,
		"more": true, "most": true, "other": true, "some": true, "such": true,
		"no": true, "nor": true, "not": true, "only": true, "own": true,
		"same": true, "so": true, "than": true, "too": true, "very": true,
		"just": true, "because": true, "but": true, "and": true, "or": true,
		"if": true, "while": true, "that": true, "this": true, "it": true,
		"its": true, "use": true, "user": true, "you": true, "your": true,
	}
	for _, w := range words {
		w = strings.ToLower(strings.TrimSpace(w))
		if len(w) > 1 && !stopWords[w] {
			keywords = append(keywords, w)
		}
	}
	return keywords
}
