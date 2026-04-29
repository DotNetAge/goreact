package reactor

import (
	"embed"
	"io/fs"

	"github.com/DotNetAge/goreact/core"
)

// bundledSkills is the embedded filesystem containing built-in skills.
// Each skill is a subdirectory under skills/ with a SKILL.md file.
//
//go:embed skills/*/SKILL.md
var bundledSkills embed.FS

// RegisterBundledSkills loads and registers built-in skills from the embedded filesystem.
// If skills is empty, all bundled skills are loaded.
// If skills is not empty, only skills with matching names are loaded.
// Skills are stored under reactor/skills/<skill-name>/SKILL.md and follow the Agent Skills spec.
func RegisterBundledSkills(registry core.SkillRegistry, skills []string) error {
	// Use the embedded FS's subdirectory as root for the bundled loader
	subFS, err := fs.Sub(bundledSkills, "skills")
	if err != nil {
		return err
	}

	loader := core.NewBundledSkillLoader(subFS, ".")
	allSkills, err := loader.Load()
	if err != nil {
		return err
	}

	for _, skill := range allSkills {
		// If skills filter is not empty, only register matching skills
		if len(skills) > 0 {
			match := false
			for _, name := range skills {
				if skill.Name == name {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if err := registry.RegisterSkill(skill); err != nil {
			return err
		}
	}
	return nil
}
