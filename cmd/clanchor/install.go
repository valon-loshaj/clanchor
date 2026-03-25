package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/valon-loshaj/clanchor/internal/crawler"
	"github.com/valon-loshaj/clanchor/internal/lockfile"
	"github.com/valon-loshaj/clanchor/internal/model"
	"github.com/valon-loshaj/clanchor/internal/resolver"
	"github.com/valon-loshaj/clanchor/internal/writer"
)

func runInstall(update bool, res resolver.Resolver) error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	slog.Info("repo root", "path", repoRoot)

	// 1. Crawl for marker files.
	result, err := crawler.Crawl(repoRoot)
	if err != nil {
		return fmt.Errorf("crawl failed: %w", err)
	}

	for _, ce := range result.Errors {
		slog.Warn("marker parse error", "path", ce.Path, "error", ce.Err)
	}

	if len(result.Markers) == 0 {
		slog.Info("no marker files found")
		return nil
	}
	slog.Info("discovered markers", "count", len(result.Markers))

	// 2. Read existing lock file.
	lf, err := lockfile.Read(repoRoot)
	if err != nil {
		return fmt.Errorf("reading lock file: %w", err)
	}

	// 3. Check for drift.
	// Additions (new markers without lock entries) are non-blocking — they're
	// resolved on every install so partial failures and new markers are picked
	// up without requiring --update. Only removals and version changes are
	// blocking drift that requires explicit --update.
	drifts := lockfile.Diff(result.Markers, lf)

	var blockingDrifts []lockfile.Drift
	for _, d := range drifts {
		switch d.Type {
		case lockfile.DriftAdded:
			slog.Info("new marker found", "path", d.Path, "version", d.NewVersion)
		case lockfile.DriftRemoved, lockfile.DriftVersionChanged:
			blockingDrifts = append(blockingDrifts, d)
		}
	}

	if len(blockingDrifts) > 0 && !update {
		slog.Warn("drift detected between marker files and lock file")
		for _, d := range blockingDrifts {
			switch d.Type {
			case lockfile.DriftRemoved:
				slog.Warn("removed marker", "path", d.Path, "was", d.OldVersion)
			case lockfile.DriftVersionChanged:
				slog.Warn("version changed", "path", d.Path, "from", d.OldVersion, "to", d.NewVersion)
			}
		}
		return fmt.Errorf("drift detected: run 'clanchor install --update' to reconcile")
	}

	// 4. Resolve markers from the registry.
	// When not updating, only resolve markers that don't already have a lock
	// entry (new or previously failed). When updating, resolve everything.
	locked := make(map[string]model.LockEntry, len(lf.Entries))
	for _, e := range lf.Entries {
		locked[e.Path] = e
	}

	var resolved []writer.ResolvedFile
	var resolveErrors []string

	for _, m := range result.Markers {
		if !update {
			if _, alreadyLocked := locked[m.Dir]; alreadyLocked {
				continue
			}
		}

		content, hash, err := res.Resolve(m.Marker.Namespace, m.Marker.Version, m.Marker.Registry)
		if err != nil {
			slog.Warn("resolve failed", "namespace", m.Marker.Namespace, "version", m.Marker.Version, "error", err)
			resolveErrors = append(resolveErrors, fmt.Sprintf("%s@%s: %v", m.Marker.Namespace, m.Marker.Version, err))
			continue
		}
		resolved = append(resolved, writer.ResolvedFile{
			Dir:     m.Dir,
			Content: content,
			Hash:    hash,
		})
	}

	// 5. Write CLAUDE.md files.
	writeResult, err := writer.WriteFiles(repoRoot, resolved, lf)
	if err != nil {
		return fmt.Errorf("writing files: %w", err)
	}

	for _, s := range writeResult.Skipped {
		slog.Warn("skipped", "path", s.Path, "reason", s.Reason)
	}

	// 6. Build and write the updated lock file.
	// Three sources of entries:
	//   a) Newly resolved markers that were successfully written
	//   b) Previously locked markers that didn't need re-resolution
	//   c) Previously managed files that were skipped (unmanaged file conflict)
	newlyWritten := make(map[string]bool, len(writeResult.Written))
	var entries []model.LockEntry

	// (a) Entries from this resolution cycle.
	for _, rf := range resolved {
		if !slices.Contains(writeResult.Written, rf.Dir) {
			continue
		}
		newlyWritten[rf.Dir] = true
		m := result.Markers[findMarkerIndex(result.Markers, rf.Dir)]
		entries = append(entries, model.LockEntry{
			Path:      rf.Dir,
			Namespace: m.Marker.Namespace,
			Version:   m.Marker.Version,
			Registry:  m.Marker.Registry,
			Hash:      rf.Hash,
		})
	}

	// (b) Carry forward existing lock entries that were not re-resolved.
	for _, e := range lf.Entries {
		if newlyWritten[e.Path] {
			continue // replaced by freshly resolved entry
		}
		// Only carry forward if a marker still exists for this path.
		if findMarkerIndex(result.Markers, e.Path) != -1 {
			entries = append(entries, e)
		}
	}

	// (c) Preserve lock entries for skipped files (unmanaged CLAUDE.md conflict).
	for _, s := range writeResult.Skipped {
		for _, e := range lf.Entries {
			if e.Path == s.Path {
				entries = append(entries, e)
				break
			}
		}
	}

	newLock := model.LockFile{Entries: entries}
	if err := lockfile.Write(repoRoot, newLock); err != nil {
		return fmt.Errorf("writing lock file: %w", err)
	}

	// 7. Summary.
	if len(writeResult.Written) > 0 {
		slog.Info("newly resolved", "count", len(writeResult.Written))
		for _, w := range writeResult.Written {
			m := result.Markers[findMarkerIndex(result.Markers, w)]
			slog.Info("  resolved", "path", w, "namespace", m.Marker.Namespace, "version", m.Marker.Version)
		}
	}

	unchanged := len(result.Markers) - len(writeResult.Written) - len(writeResult.Skipped) - len(resolveErrors)
	slog.Info("install complete",
		"resolved", len(writeResult.Written),
		"unchanged", unchanged,
		"skipped", len(writeResult.Skipped),
		"failed", len(resolveErrors),
	)

	if len(resolveErrors) > 0 {
		slog.Warn("the following namespaces could not be resolved:")
		for _, e := range resolveErrors {
			slog.Warn("  " + e)
		}
	}

	return nil
}

func findMarkerIndex(markers []model.DiscoveredMarker, dir string) int {
	for i, m := range markers {
		if m.Dir == dir {
			return i
		}
	}
	return -1
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not inside a git repository")
		}
		dir = parent
	}
}
