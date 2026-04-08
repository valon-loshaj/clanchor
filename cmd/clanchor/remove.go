package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/valon-loshaj/clanchor/internal/lockfile"
	"github.com/valon-loshaj/clanchor/internal/manifest"
	"github.com/valon-loshaj/clanchor/internal/model"
	"github.com/valon-loshaj/clanchor/internal/writer"
)

func runRemove(packageName string) error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}

	m, err := manifest.Read(repoRoot)
	if err != nil {
		return err
	}

	lf, err := lockfile.ReadV2(repoRoot)
	if err != nil {
		return fmt.Errorf("reading lock file: %w", err)
	}

	// Find and remove the package from the manifest.
	found := false
	var remaining []model.PackageEntry
	for _, p := range m.Packages {
		if p.Name == packageName {
			found = true
			continue
		}
		remaining = append(remaining, p)
	}

	if !found {
		return fmt.Errorf("package %q not found in manifest", packageName)
	}

	// Delete the package's files from disk.
	for _, pkg := range lf.Packages {
		if pkg.Name == packageName {
			root, err := targetRoot(repoRoot, pkg.Scope)
			if err != nil {
				return err
			}
			errs := writer.DeletePackageFiles(root, pkg.Files)
			for _, e := range errs {
				slog.Warn("delete failed", "error", e)
			}
			if len(errs) == 0 {
				slog.Info("deleted package files", "name", packageName, "files", len(pkg.Files))
			}
			break
		}
	}

	// Update manifest.
	m.Packages = remaining
	manifestData, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	manifestData = append(manifestData, '\n')
	if err := os.WriteFile(filepath.Join(repoRoot, manifest.FileName), manifestData, 0o644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	// Update lock file.
	var remainingLock []model.PackageLockEntry
	for _, pkg := range lf.Packages {
		if pkg.Name != packageName {
			remainingLock = append(remainingLock, pkg)
		}
	}
	lf.Packages = remainingLock
	if err := lockfile.WriteV2(repoRoot, lf); err != nil {
		return fmt.Errorf("writing lock file: %w", err)
	}

	slog.Info("removed package", "name", packageName)
	return nil
}
