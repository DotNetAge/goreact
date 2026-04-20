package core

import (
	"fmt"
	"strings"
)

// frontmatterDelimiter is the YAML frontmatter delimiter.
const frontmatterDelimiter = "---"

// parseYamlFrontmatter extracts YAML frontmatter and body from a markdown file.
// The file format is:
//
//	---
//	key: value
//	---
//	Markdown body content
//
// Returns the parsed frontmatter fields, the body content, and any error.
func parseYamlFrontmatter(content string) (skillFrontmatter, string, error) {
	var fm skillFrontmatter

	content = strings.TrimLeft(content, "\n\r")
	if !strings.HasPrefix(content, frontmatterDelimiter) {
		return fm, content, fmt.Errorf("SKILL.md must start with YAML frontmatter (---)")
	}

	// Find closing delimiter
	rest := content[len(frontmatterDelimiter):]
	closeIdx := strings.Index(rest, "\n" + frontmatterDelimiter)
	if closeIdx < 0 {
		return fm, content, fmt.Errorf("SKILL.md has unclosed YAML frontmatter (missing closing ---)")
	}

	yamlBlock := rest[:closeIdx]
	body := rest[closeIdx+len(frontmatterDelimiter)+1:]

	// Parse YAML frontmatter into skillFrontmatter
	if err := parseSimpleYaml(yamlBlock, &fm); err != nil {
		return fm, body, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	return fm, body, nil
}

// parseSimpleYaml parses a minimal subset of YAML into a skillFrontmatter.
// This avoids pulling in a full YAML dependency for the small subset we need.
// Supports: scalar strings, space-separated strings, and simple maps.
func parseSimpleYaml(yaml string, fm *skillFrontmatter) error {
	lines := strings.Split(yaml, "\n")
	var currentMapKey string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Top-level key: value
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			// Remove surrounding quotes
			value = strings.Trim(value, `"'`)

			switch key {
			case "name":
				fm.Name = value
			case "description":
				// Description can be multiline (no value on the colon line)
				if value == "" {
					continue // description will be handled as multiline below
				}
				fm.Description = value
			case "license":
				fm.License = value
			case "compatibility":
				fm.Compatibility = value
			case "allowed-tools":
				fm.AllowedTools = value
			case "metadata":
				if fm.Metadata == nil {
					fm.Metadata = make(map[string]string)
				}
				currentMapKey = "metadata"
				if value != "" && value != "{" {
					// Inline map: key: value, key2: value2
					parseInlineMap(value, fm.Metadata)
				}
			default:
				// Unknown field, skip
			}
		} else if currentMapKey != "" && strings.HasPrefix(line, "- ") || strings.Contains(line, ":") {
			// Map sub-key: value
			if subIdx := strings.Index(line, ":"); subIdx > 0 {
				subKey := strings.TrimSpace(line[:subIdx])
				subVal := strings.TrimSpace(line[subIdx+1:])
				subVal = strings.Trim(subVal, `"'`)
				if currentMapKey == "metadata" && fm.Metadata != nil {
					fm.Metadata[subKey] = subVal
				}
			}
		}
	}

	return nil
}

// parseInlineMap parses "key: value, key2: value2" into a map.
func parseInlineMap(s string, m map[string]string) {
	for pair := range strings.SplitSeq(s, ",") {
		pair = strings.TrimSpace(pair)
		if idx := strings.Index(pair, ":"); idx > 0 {
			k := strings.TrimSpace(pair[:idx])
			v := strings.TrimSpace(pair[idx+1:])
			v = strings.Trim(v, `"'`)
			m[k] = v
		}
	}
}
