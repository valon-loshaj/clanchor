package model

// LockFileV2 represents the v2 lock file with separate package and CLAUDE.md tracking.
type LockFileV2 struct {
	Version  int                `json:"version"`
	Packages []PackageLockEntry `json:"packages,omitempty"`
	ClaudeMD []ClaudeMDLock     `json:"claude_md,omitempty"`
}

// PackageLockEntry tracks a resolved .claude directory package and all its files.
type PackageLockEntry struct {
	Name     string       `json:"name"`
	Version  string       `json:"version"`
	Registry string       `json:"registry"`
	Scope    Scope        `json:"scope"`
	Files    []LockedFile `json:"files"`
}

// LockedFile tracks an individual file placed by a package.
type LockedFile struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

// ClaudeMDLock tracks a resolved CLAUDE.md file placement.
type ClaudeMDLock struct {
	Path      string `json:"path"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
	Registry  string `json:"registry"`
	Hash      string `json:"hash"`
}
