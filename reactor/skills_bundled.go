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

// RegisterBundledSkills loads and registers all built-in skills from the embedded filesystem.
// Skills are stored under reactor/skills/<skill-name>/SKILL.md and follow the Agent Skills spec.
func RegisterBundledSkills(registry core.SkillRegistry) error {
	// Use the embedded FS's subdirectory as root for the bundled loader
	subFS, err := fs.Sub(bundledSkills, "skills")
	if err != nil {
		return err
	}

	loader := core.NewBundledSkillLoader(subFS, ".")
	skills, err := loader.Load()
	if err != nil {
		return err
	}

	for _, skill := range skills {
		if err := registry.RegisterSkill(skill); err != nil {
			return err
		}
	}
	return nil
}
