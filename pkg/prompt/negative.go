package prompt

import (
	"strings"
	"sync"
)

// NegativePromptManager manages negative prompt groups
type NegativePromptManager struct {
	mu     sync.RWMutex
	groups map[string]*NegativePromptGroup
}

// NewNegativePromptManager creates a new NegativePromptManager
func NewNegativePromptManager() *NegativePromptManager {
	m := &NegativePromptManager{
		groups: make(map[string]*NegativePromptGroup),
	}

	// Initialize with default groups
	for _, group := range DefaultNegativePromptGroups() {
		m.groups[group.ID] = group
	}

	return m
}

// AddGroup adds a negative prompt group
func (m *NegativePromptManager) AddGroup(group *NegativePromptGroup) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.groups[group.ID] = group
}

// RemoveGroup removes a negative prompt group
func (m *NegativePromptManager) RemoveGroup(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.groups, id)
}

// GetGroup retrieves a negative prompt group by ID
func (m *NegativePromptManager) GetGroup(id string) *NegativePromptGroup {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.groups[id]
}

// GetEnabledGroups returns all enabled negative prompt groups
func (m *NegativePromptManager) GetEnabledGroups() []*NegativePromptGroup {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*NegativePromptGroup, 0)
	for _, group := range m.groups {
		if group.Enabled {
			result = append(result, group)
		}
	}
	return result
}

// EnableGroup enables a negative prompt group
func (m *NegativePromptManager) EnableGroup(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if group, ok := m.groups[id]; ok {
		group.Enabled = true
	}
}

// DisableGroup disables a negative prompt group
func (m *NegativePromptManager) DisableGroup(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if group, ok := m.groups[id]; ok {
		group.Enabled = false
	}
}

// GetAllPrompts returns all prompts from enabled groups
func (m *NegativePromptManager) GetAllPrompts() []*NegativePrompt {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*NegativePrompt, 0)
	for _, group := range m.groups {
		if group.Enabled {
			result = append(result, group.Prompts...)
		}
	}
	return result
}

// GetPromptsByGroup returns prompts from a specific group
func (m *NegativePromptManager) GetPromptsByGroup(id string) []*NegativePrompt {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if group, ok := m.groups[id]; ok {
		return group.Prompts
	}
	return nil
}

// AddPromptToGroup adds a prompt to a group
func (m *NegativePromptManager) AddPromptToGroup(groupID string, prompt *NegativePrompt) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if group, ok := m.groups[groupID]; ok {
		group.Prompts = append(group.Prompts, prompt)
	}
}

// FilterByPermission filters prompts based on user permission
func (m *NegativePromptManager) FilterByPermission(permission *Permission) []*NegativePrompt {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*NegativePrompt, 0)
	for _, group := range m.groups {
		if !group.Enabled {
			continue
		}

		// Skip permission group for admins
		if group.ID == "permission" && permission != nil && permission.IsAdmin {
			continue
		}

		result = append(result, group.Prompts...)
	}
	return result
}

// FormatPrompts formats prompts for injection
func (m *NegativePromptManager) FormatPrompts(prompts []*NegativePrompt) string {
	if len(prompts) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Negative Prompts\n\n")
	sb.WriteString("请严格遵守以下约束条件：\n\n")

	for _, prompt := range prompts {
		sb.WriteString("- ")
		sb.WriteString(prompt.Pattern)
		if prompt.Alternative != "" {
			sb.WriteString("。替代方案：")
			sb.WriteString(prompt.Alternative)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ShouldInjectGroup determines if a group should be injected
func (m *NegativePromptManager) ShouldInjectGroup(group *NegativePromptGroup, permission *Permission) bool {
	if !group.Enabled {
		return false
	}

	// Skip permission constraints for admins
	if group.ID == "permission" && permission != nil && permission.IsAdmin {
		return false
	}

	return true
}
