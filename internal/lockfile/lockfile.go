package lockfile

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/valon-loshaj/clanchor/internal/model"
)

const FileName = "clanchor-lock.json"
const currentVersion = 1

// Read loads a lock file from disk. Returns a zero-value LockFile if the file
// does not exist (first run).
func Read(repoRoot string) (model.LockFile, error) {
	path := filepath.Join(repoRoot, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return model.LockFile{Version: currentVersion}, nil
		}
		return model.LockFile{}, err
	}

	var lf model.LockFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return model.LockFile{}, err
	}
	return lf, nil
}

// Write persists a lock file to disk with entries sorted by path for stable diffs.
func Write(repoRoot string, lf model.LockFile) error {
	lf.Version = currentVersion
	sort.Slice(lf.Entries, func(i, j int) bool {
		return lf.Entries[i].Path < lf.Entries[j].Path
	})

	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(repoRoot, FileName), data, 0o644)
}

// DriftType describes how a marker differs from the lock file.
type DriftType string

const (
	DriftAdded          DriftType = "added"
	DriftRemoved        DriftType = "removed"
	DriftVersionChanged DriftType = "version_changed"
)

// Drift records a single discrepancy between a discovered marker and the lock file.
type Drift struct {
	Path       string
	Type       DriftType
	OldVersion string // empty for added
	NewVersion string // empty for removed
}

// Diff compares discovered markers against an existing lock file and returns
// all discrepancies.
func Diff(markers []model.DiscoveredMarker, lf model.LockFile) []Drift {
	locked := make(map[string]model.LockEntry, len(lf.Entries))
	for _, e := range lf.Entries {
		locked[e.Path] = e
	}

	seen := make(map[string]bool, len(markers))
	var drifts []Drift

	for _, m := range markers {
		seen[m.Dir] = true
		entry, exists := locked[m.Dir]
		if !exists {
			drifts = append(drifts, Drift{
				Path:       m.Dir,
				Type:       DriftAdded,
				NewVersion: m.Marker.Version,
			})
			continue
		}
		if entry.Version != m.Marker.Version {
			drifts = append(drifts, Drift{
				Path:       m.Dir,
				Type:       DriftVersionChanged,
				OldVersion: entry.Version,
				NewVersion: m.Marker.Version,
			})
		}
	}

	for _, e := range lf.Entries {
		if !seen[e.Path] {
			drifts = append(drifts, Drift{
				Path:       e.Path,
				Type:       DriftRemoved,
				OldVersion: e.Version,
			})
		}
	}

	return drifts
}
