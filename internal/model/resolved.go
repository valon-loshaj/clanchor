package model

// ResolvedFile represents a single file fetched from the registry.
type ResolvedFile struct {
	RelativePath string // path relative to the package root (e.g., ".claude/skills/go-review/SKILL.md")
	Content      []byte
	Hash         string // "sha256:<hex>"
}

// ResolvedPackage groups all resolved files for a single package entry.
type ResolvedPackage struct {
	Name     string
	Version  string
	Registry string
	Scope    Scope
	Files    []ResolvedFile
}
