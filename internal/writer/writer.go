package writer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/valon-loshaj/clanchor/internal/model"
)

const managedHeader = "<!-- managed by clanchor — do not edit manually -->\n"

// WriteResult records the outcome of writing CLAUDE.md files.
type WriteResult struct {
	Written  []string // paths where files were written
	Skipped  []SkipReason
}

// SkipReason records why a directory was skipped.
type SkipReason struct {
	Path   string
	Reason string
}

// WriteFiles writes resolved CLAUDE.md content into each target directory.
// It skips directories that already contain an unmanaged CLAUDE.md (one not
// present in the existing lock file).
func WriteFiles(repoRoot string, resolved []ResolvedFile, existingLock model.LockFile) (WriteResult, error) {
	managed := make(map[string]bool, len(existingLock.Entries))
	for _, e := range existingLock.Entries {
		managed[e.Path] = true
	}

	var result WriteResult

	for _, rf := range resolved {
		target := filepath.Join(repoRoot, rf.Dir, "CLAUDE.md")

		if fileExists(target) && !managed[rf.Dir] {
			result.Skipped = append(result.Skipped, SkipReason{
				Path:   rf.Dir,
				Reason: fmt.Sprintf("unmanaged CLAUDE.md already exists at %s", rf.Dir),
			})
			continue
		}

		content := []byte(managedHeader + string(rf.Content))
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return result, fmt.Errorf("writing %s: %w", target, err)
		}
		result.Written = append(result.Written, rf.Dir)
	}

	return result, nil
}

// ResolvedFile pairs a directory with its resolved CLAUDE.md content and hash.
type ResolvedFile struct {
	Dir     string
	Content []byte
	Hash    string
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
