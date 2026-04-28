package reactor

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/DotNetAge/goreact/core"
)

// DefaultRuleRegistry manages behavior rules with thread-safe operations.
// Rules are sorted by Priority (descending) before rendering into prompts,
// so higher-priority rules appear first and have more LLM attention.
type DefaultRuleRegistry struct {
	mu    sync.RWMutex
	rules map[string]*core.Rule
}

// NewDefaultRuleRegistry creates an empty rule registry.
func NewDefaultRuleRegistry() *DefaultRuleRegistry {
	return &DefaultRuleRegistry{
		rules: make(map[string]*core.Rule),
	}
}

var _ core.RuleRegistry = (*DefaultRuleRegistry)(nil)

func (r *DefaultRuleRegistry) Register(rule core.Rule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.rules[rule.ID]; exists {
		return fmt.Errorf("rule %q already registered", rule.ID)
	}
	cp := rule
	r.rules[rule.ID] = &cp
	return nil
}

func (r *DefaultRuleRegistry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rules, id)
}

func (r *DefaultRuleRegistry) Get(id string) (*core.Rule, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if rule, ok := r.rules[id]; ok && rule.Enabled {
		return rule, true
	}
	return nil, false
}

func (r *DefaultRuleRegistry) All() []core.Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]core.Rule, 0, len(r.rules))
	for _, rule := range r.rules {
		out = append(out, *rule)
	}
	return out
}

func (r *DefaultRuleRegistry) GetByScope(scope core.RuleScope) []core.Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var filtered []core.Rule
	for _, rule := range r.rules {
		if rule.Scope == scope && rule.Enabled {
			filtered = append(filtered, *rule)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Priority > filtered[j].Priority
	})
	return filtered
}

// FormatPromptSection renders all enabled rules into a text block suitable for
// injection into the System Prompt's <behavioral_rules> section.
//
// Output format:
//
//	[RULE] #<priority> <name> (<scope>)
//	  <content>
//
// Rules are sorted by Priority descending so high-importance rules appear first.
func (r *DefaultRuleRegistry) FormatPromptSection() string {
	rules := r.All()

	var enabled []core.Rule
	for _, rule := range rules {
		if rule.Enabled {
			enabled = append(enabled, rule)
		}
	}

	if len(enabled) == 0 {
		return ""
	}

	sort.Slice(enabled, func(i, j int) bool {
		return enabled[i].Priority > enabled[j].Priority
	})

	var sb strings.Builder
	for i, rule := range enabled {
		fmt.Fprintf(&sb, "%d. [RULE:%s] %s (%s)\n", i+1, rule.ID, rule.Name, rule.Scope)
		sb.WriteString("   ")
		sb.WriteString(rule.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}
