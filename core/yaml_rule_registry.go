package core

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// YAMLRuleRegistry implements RuleRegistry backed by a YAML file.
// It loads rules from a YAML file on creation and persists changes
// back to the same file on every mutation.
//
// YAML format:
//
//	rules:
//	  - id: no-delete-prod
//	    intro: "Never delete production data files."
//	    scope: global
//	    priority: 100
//	    enabled: true
//	  - id: backup-first
//	    intro: "Any modification must be backed up first."
//	    scope: local
//	    priority: 50
//	    enabled: true
type YAMLRuleRegistry struct {
	mu    sync.RWMutex
	file  string
	rules map[string]*Rule
}

// NewYAMLRuleRegistry loads rules from a YAML file and returns a registry
// implementing the RuleRegistry interface. If the file does not exist, an
// empty registry is created and the file will be written on first mutation.
func NewYAMLRuleRegistry(path string) (*YAMLRuleRegistry, error) {
	if path == "" {
		return nil, fmt.Errorf("rule registry path must not be empty")
	}

	r := &YAMLRuleRegistry{
		file:  path,
		rules: make(map[string]*Rule),
	}

	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

type yamlRulesFile struct {
	Rules []*Rule `yaml:"rules"`
}

func (r *YAMLRuleRegistry) load() error {
	data, err := os.ReadFile(r.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read rules file: %w", err)
	}

	var file yamlRulesFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to unmarshal rules YAML: %w", err)
	}

	for _, rule := range file.Rules {
		if rule == nil || rule.ID == "" {
			continue
		}
		r.rules[rule.ID] = rule
	}
	return nil
}

func (r *YAMLRuleRegistry) save() error {
	list := make([]*Rule, 0, len(r.rules))
	for _, rule := range r.rules {
		list = append(list, rule)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Priority > list[j].Priority ||
			(list[i].Priority == list[j].Priority && list[i].ID < list[j].ID)
	})

	file := yamlRulesFile{Rules: list}
	data, err := yaml.Marshal(file)
	if err != nil {
		return fmt.Errorf("failed to marshal rules YAML: %w", err)
	}

	if err := os.WriteFile(r.file, data, 0644); err != nil {
		return fmt.Errorf("failed to write rules file: %w", err)
	}
	return nil
}

// Register adds a new rule. Returns ErrDuplicateRule if the rule ID already exists.
func (r *YAMLRuleRegistry) Register(rule Rule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rules[rule.ID]; exists {
		return fmt.Errorf("rule registry: duplicate rule ID")
	}

	ruleCopy := rule
	r.rules[rule.ID] = &ruleCopy
	return r.save()
}

// Unregister removes a rule by ID and persists changes.
func (r *YAMLRuleRegistry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.rules, id)
	_ = r.save()
}

// Update replaces an existing rule by ID. Returns error if the rule does not exist.
func (r *YAMLRuleRegistry) Update(rule Rule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rules[rule.ID]; !exists {
		return fmt.Errorf("rule registry: rule not found")
	}

	ruleCopy := rule
	r.rules[rule.ID] = &ruleCopy
	return r.save()
}

// Get retrieves a rule by ID.
func (r *YAMLRuleRegistry) Get(id string) (*Rule, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rule, ok := r.rules[id]
	if !ok {
		return nil, false
	}
	ruleCopy := *rule
	return &ruleCopy, true
}

// All returns all rules sorted by priority (descending), then by ID (ascending).
func (r *YAMLRuleRegistry) All() []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rules := make([]Rule, 0, len(r.rules))
	for _, rule := range r.rules {
		if rule.Enabled {
			rules = append(rules, *rule)
		}
	}

	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority > rules[j].Priority
		}
		return rules[i].ID < rules[j].ID
	})
	return rules
}

// GetByScope returns all enabled rules matching the given scope.
func (r *YAMLRuleRegistry) GetByScope(scope RuleScope) []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rules := make([]Rule, 0)
	for _, rule := range r.rules {
		if rule.Enabled && rule.Scope == scope {
			rules = append(rules, *rule)
		}
	}

	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Priority != rules[j].Priority {
			return rules[i].Priority > rules[j].Priority
		}
		return rules[i].ID < rules[j].ID
	})
	return rules
}

// FormatPromptSection renders all enabled rules into a markdown-formatted
// string suitable for injection into an LLM system prompt.
func (r *YAMLRuleRegistry) FormatPromptSection() string {
	rules := r.All()
	if len(rules) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("## Behavioral Rules\n\n")
	for i, rule := range rules {
		scope := string(rule.Scope)
		if scope == "" {
			scope = "global"
		}
		b.WriteString(fmt.Sprintf("%d. [%s] %s (priority: %d)\n", i+1, scope, rule.Intro, rule.Priority))
	}
	b.WriteString("\n")
	return b.String()
}
