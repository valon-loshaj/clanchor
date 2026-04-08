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

const v2Version = 2

// ReadV2 loads a v2 lock file from disk. Returns a zero-value LockFileV2 if the
// file does not exist (first run).
func ReadV2(repoRoot string) (model.LockFileV2, error) {
	path := filepath.Join(repoRoot, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return model.LockFileV2{Version: v2Version}, nil
		}
		return model.LockFileV2{}, err
	}

	var lf model.LockFileV2
	if err := json.Unmarshal(data, &lf); err != nil {
		return model.LockFileV2{}, err
	}
	return lf, nil
}

// WriteV2 persists a v2 lock file to disk with entries sorted for stable diffs.
func WriteV2(repoRoot string, lf model.LockFileV2) error {
	lf.Version = v2Version

	sort.Slice(lf.Packages, func(i, j int) bool {
		return lf.Packages[i].Name < lf.Packages[j].Name
	})
	for k := range lf.Packages {
		sort.Slice(lf.Packages[k].Files, func(i, j int) bool {
			return lf.Packages[k].Files[i].Path < lf.Packages[k].Files[j].Path
		})
	}
	sort.Slice(lf.ClaudeMD, func(i, j int) bool {
		return lf.ClaudeMD[i].Path < lf.ClaudeMD[j].Path
	})

	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(repoRoot, FileName), data, 0o644)
}

// PackageDrift describes how a package entry differs from the lock file.
type PackageDrift struct {
	Name       string
	Type       DriftType
	OldVersion string
	NewVersion string
}

// DiffV2Packages compares manifest package entries against the v2 lock file.
func DiffV2Packages(packages []model.PackageEntry, defaultRegistry string, lf model.LockFileV2) []PackageDrift {
	locked := make(map[string]model.PackageLockEntry, len(lf.Packages))
	for _, e := range lf.Packages {
		locked[e.Name] = e
	}

	seen := make(map[string]bool, len(packages))
	var drifts []PackageDrift

	for _, p := range packages {
		seen[p.Name] = true
		entry, exists := locked[p.Name]
		if !exists {
			drifts = append(drifts, PackageDrift{
				Name:       p.Name,
				Type:       DriftAdded,
				NewVersion: p.Version,
			})
			continue
		}
		if entry.Version != p.Version {
			drifts = append(drifts, PackageDrift{
				Name:       p.Name,
				Type:       DriftVersionChanged,
				OldVersion: entry.Version,
				NewVersion: p.Version,
			})
		}
	}

	for _, e := range lf.Packages {
		if !seen[e.Name] {
			drifts = append(drifts, PackageDrift{
				Name:       e.Name,
				Type:       DriftRemoved,
				OldVersion: e.Version,
			})
		}
	}

	return drifts
}

// ClaudeMDDrift describes how a CLAUDE.md entry differs from the lock file.
type ClaudeMDDrift struct {
	Path       string
	Type       DriftType
	OldVersion string
	NewVersion string
}

// DiffV2ClaudeMD compares manifest claude_md entries against the v2 lock file.
func DiffV2ClaudeMD(entries []model.ClaudeMDEntry, lf model.LockFileV2) []ClaudeMDDrift {
	locked := make(map[string]model.ClaudeMDLock, len(lf.ClaudeMD))
	for _, e := range lf.ClaudeMD {
		locked[e.Path] = e
	}

	seen := make(map[string]bool, len(entries))
	var drifts []ClaudeMDDrift

	for _, c := range entries {
		seen[c.Path] = true
		entry, exists := locked[c.Path]
		if !exists {
			drifts = append(drifts, ClaudeMDDrift{
				Path:       c.Path,
				Type:       DriftAdded,
				NewVersion: c.Version,
			})
			continue
		}
		if entry.Version != c.Version {
			drifts = append(drifts, ClaudeMDDrift{
				Path:       c.Path,
				Type:       DriftVersionChanged,
				OldVersion: entry.Version,
				NewVersion: c.Version,
			})
		}
	}

	for _, e := range lf.ClaudeMD {
		if !seen[e.Path] {
			drifts = append(drifts, ClaudeMDDrift{
				Path:       e.Path,
				Type:       DriftRemoved,
				OldVersion: e.Version,
			})
		}
	}

	return drifts
}

// ManagedFiles returns the set of file paths owned by clanchor from the lock file.
func ManagedFiles(lf model.LockFileV2) map[string]bool {
	managed := make(map[string]bool)
	for _, pkg := range lf.Packages {
		for _, f := range pkg.Files {
			managed[f.Path] = true
		}
	}
	return managed
}
