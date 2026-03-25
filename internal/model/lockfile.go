package model

// LockEntry represents a single resolved CLAUDE.md in the lock file.
type LockEntry struct {
	Path      string `json:"path"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
	Registry  string `json:"registry"`
	Hash      string `json:"hash"`
}

// LockFile represents the full manifest/lock file.
type LockFile struct {
	Version int         `json:"version"`
	Entries []LockEntry `json:"entries"`
}
