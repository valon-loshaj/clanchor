package writer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/valon-loshaj/clanchor/internal/model"
)

// PackageWriteResult records the outcome of writing package files.
type PackageWriteResult struct {
	Written []string     // file paths successfully written (relative to target root)
	Skipped []SkipReason // files skipped with reasons
}

// WritePackageFiles writes resolved package files into the target .claude directory.
// targetRoot is the directory containing .claude/ (e.g., repo root for project scope,
// home dir for global scope). managedFiles is the set of file paths currently owned
// by clanchor (from the lock file) — files not in this set that already exist are skipped.
func WritePackageFiles(targetRoot string, pkg model.ResolvedPackage, managedFiles map[string]bool) (PackageWriteResult, error) {
	var result PackageWriteResult

	for _, f := range pkg.Files {
		target := filepath.Join(targetRoot, f.RelativePath)

		if fileExists(target) && !managedFiles[f.RelativePath] {
			result.Skipped = append(result.Skipped, SkipReason{
				Path:   f.RelativePath,
				Reason: fmt.Sprintf("unmanaged file already exists at %s", f.RelativePath),
			})
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return result, fmt.Errorf("creating directory for %s: %w", f.RelativePath, err)
		}

		if err := os.WriteFile(target, f.Content, 0o644); err != nil {
			return result, fmt.Errorf("writing %s: %w", f.RelativePath, err)
		}

		result.Written = append(result.Written, f.RelativePath)
	}

	return result, nil
}

// DeletePackageFiles removes files that were previously installed by a package.
// Only deletes files tracked in the lock file. Cleans up empty parent directories
// up to but not including targetRoot.
func DeletePackageFiles(targetRoot string, files []model.LockedFile) []error {
	var errs []error

	for _, f := range files {
		target := filepath.Join(targetRoot, f.Path)
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("removing %s: %w", f.Path, err))
			continue
		}
		cleanEmptyParents(filepath.Dir(target), targetRoot)
	}

	return errs
}

// cleanEmptyParents removes empty directories walking up from dir to (but not including) stopAt.
func cleanEmptyParents(dir, stopAt string) {
	absStop, _ := filepath.Abs(stopAt)
	for {
		absDir, _ := filepath.Abs(dir)
		if absDir == absStop || absDir == "/" {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}
