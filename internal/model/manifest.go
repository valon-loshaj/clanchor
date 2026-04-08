package model

import (
	"encoding/json"
	"fmt"
)

// Scope determines where a package's .claude/ contents are installed.
type Scope string

const (
	ScopeProject Scope = "project"
	ScopeGlobal  Scope = "global"
)

// Manifest represents the v2 clanchor.json at repo root.
type Manifest struct {
	Version  int             `json:"version"`
	Registry string          `json:"registry,omitempty"`
	Packages []PackageEntry  `json:"packages,omitempty"`
	ClaudeMD []ClaudeMDEntry `json:"claude_md,omitempty"`
}

// PackageEntry declares a .claude directory package to install.
type PackageEntry struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Registry string `json:"registry,omitempty"`
	Scope    Scope  `json:"scope,omitempty"`
}

// ClaudeMDEntry declares a CLAUDE.md file to place at a repo path.
type ClaudeMDEntry struct {
	Path      string `json:"path"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
	Registry  string `json:"registry,omitempty"`
}

// EffectiveRegistry returns the entry's registry if set, otherwise the manifest default.
func (p PackageEntry) EffectiveRegistry(manifestDefault string) string {
	if p.Registry != "" {
		return p.Registry
	}
	return manifestDefault
}

// EffectiveScope returns the entry's scope, defaulting to project.
func (p PackageEntry) EffectiveScope() Scope {
	if p.Scope != "" {
		return p.Scope
	}
	return ScopeProject
}

// EffectiveRegistry returns the entry's registry if set, otherwise the manifest default.
func (c ClaudeMDEntry) EffectiveRegistry(manifestDefault string) string {
	if c.Registry != "" {
		return c.Registry
	}
	return manifestDefault
}

// ParseManifest unmarshals and validates a v2 manifest from JSON.
func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("invalid JSON: %w", err)
	}
	return m, m.Validate()
}

func (m Manifest) Validate() error {
	if m.Version != 2 {
		return fmt.Errorf("unsupported manifest version %d (expected 2)", m.Version)
	}
	// An empty manifest is valid — it means all packages have been removed.
	if m.Registry != "" {
		if !orgRepoRe.MatchString(m.Registry) {
			return fmt.Errorf("default registry %q is not valid org/repo format", m.Registry)
		}
	}
	for i, p := range m.Packages {
		if err := validatePackageEntry(p, m.Registry, i); err != nil {
			return err
		}
	}
	for i, c := range m.ClaudeMD {
		if err := validateClaudeMDEntry(c, m.Registry, i); err != nil {
			return err
		}
	}
	return nil
}

func validatePackageEntry(p PackageEntry, defaultRegistry string, idx int) error {
	prefix := fmt.Sprintf("packages[%d]", idx)
	if p.Name == "" {
		return fmt.Errorf("%s: name is required", prefix)
	}
	if p.Version == "" {
		return fmt.Errorf("%s: version is required", prefix)
	}
	if !semverRe.MatchString(p.Version) {
		return fmt.Errorf("%s: version %q is not valid semver (expected X.Y.Z)", prefix, p.Version)
	}
	registry := p.EffectiveRegistry(defaultRegistry)
	if registry == "" {
		return fmt.Errorf("%s: registry is required (set per-entry or as manifest default)", prefix)
	}
	if !orgRepoRe.MatchString(registry) {
		return fmt.Errorf("%s: registry %q is not valid org/repo format", prefix, registry)
	}
	if p.Scope != "" && p.Scope != ScopeProject && p.Scope != ScopeGlobal {
		return fmt.Errorf("%s: scope must be %q or %q", prefix, ScopeProject, ScopeGlobal)
	}
	return nil
}

func validateClaudeMDEntry(c ClaudeMDEntry, defaultRegistry string, idx int) error {
	prefix := fmt.Sprintf("claude_md[%d]", idx)
	if c.Path == "" {
		return fmt.Errorf("%s: path is required", prefix)
	}
	if c.Namespace == "" {
		return fmt.Errorf("%s: namespace is required", prefix)
	}
	if c.Version == "" {
		return fmt.Errorf("%s: version is required", prefix)
	}
	if !semverRe.MatchString(c.Version) {
		return fmt.Errorf("%s: version %q is not valid semver (expected X.Y.Z)", prefix, c.Version)
	}
	registry := c.EffectiveRegistry(defaultRegistry)
	if registry == "" {
		return fmt.Errorf("%s: registry is required (set per-entry or as manifest default)", prefix)
	}
	if !orgRepoRe.MatchString(registry) {
		return fmt.Errorf("%s: registry %q is not valid org/repo format", prefix, registry)
	}
	return nil
}
