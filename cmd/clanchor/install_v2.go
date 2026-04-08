package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/valon-loshaj/clanchor/internal/conflict"
	"github.com/valon-loshaj/clanchor/internal/lockfile"
	"github.com/valon-loshaj/clanchor/internal/manifest"
	"github.com/valon-loshaj/clanchor/internal/model"
	"github.com/valon-loshaj/clanchor/internal/resolver"
	"github.com/valon-loshaj/clanchor/internal/writer"
)

// targetRoot returns the write destination for a given scope.
func targetRoot(repoRoot string, scope model.Scope) (string, error) {
	if scope == model.ScopeGlobal {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}
		return home, nil
	}
	return repoRoot, nil
}

func runInstallV2(update bool, res resolver.Resolver) error {
	// 1. Find repo root.
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	slog.Info("repo root", "path", repoRoot)

	// 2. Read manifest.
	m, err := manifest.Read(repoRoot)
	if err != nil {
		return err
	}
	slog.Info("manifest loaded",
		"packages", len(m.Packages),
		"claude_md", len(m.ClaudeMD),
	)

	// 3. Read lock file.
	lf, err := lockfile.ReadV2(repoRoot)
	if err != nil {
		return fmt.Errorf("reading lock file: %w", err)
	}

	// 4. Detect drift.
	pkgDrifts := lockfile.DiffV2Packages(m.Packages, m.Registry, lf)
	mdDrifts := lockfile.DiffV2ClaudeMD(m.ClaudeMD, lf)

	var blocking []string
	for _, d := range pkgDrifts {
		switch d.Type {
		case lockfile.DriftAdded:
			slog.Info("new package", "name", d.Name, "version", d.NewVersion)
		case lockfile.DriftRemoved:
			blocking = append(blocking, fmt.Sprintf("package removed: %s (was %s)", d.Name, d.OldVersion))
		case lockfile.DriftVersionChanged:
			blocking = append(blocking, fmt.Sprintf("package version changed: %s %s → %s", d.Name, d.OldVersion, d.NewVersion))
		}
	}
	for _, d := range mdDrifts {
		switch d.Type {
		case lockfile.DriftAdded:
			slog.Info("new claude_md", "path", d.Path, "version", d.NewVersion)
		case lockfile.DriftRemoved:
			blocking = append(blocking, fmt.Sprintf("claude_md removed: %s (was %s)", d.Path, d.OldVersion))
		case lockfile.DriftVersionChanged:
			blocking = append(blocking, fmt.Sprintf("claude_md version changed: %s %s → %s", d.Path, d.OldVersion, d.NewVersion))
		}
	}

	if len(blocking) > 0 && !update {
		slog.Warn("drift detected between manifest and lock file")
		for _, b := range blocking {
			slog.Warn("  " + b)
		}
		return fmt.Errorf("drift detected: run 'clanchor install --update' to reconcile")
	}

	// Build lookup of locked packages and claude_md entries.
	lockedPkgs := make(map[string]model.PackageLockEntry, len(lf.Packages))
	for _, e := range lf.Packages {
		lockedPkgs[e.Name] = e
	}
	lockedMD := make(map[string]model.ClaudeMDLock, len(lf.ClaudeMD))
	for _, e := range lf.ClaudeMD {
		lockedMD[e.Path] = e
	}

	// 5. Resolve from registry.
	var resolvedPkgs []model.ResolvedPackage
	var resolvedMDs []writer.ResolvedFile
	var resolveErrors []string

	// Resolve packages.
	for _, p := range m.Packages {
		if !update {
			if _, locked := lockedPkgs[p.Name]; locked {
				continue
			}
		}

		registry := p.EffectiveRegistry(m.Registry)
		files, err := res.ResolvePackage(p.Name, p.Version, registry)
		if err != nil {
			slog.Warn("resolve failed", "package", p.Name, "version", p.Version, "error", err)
			resolveErrors = append(resolveErrors, fmt.Sprintf("package %s@%s: %v", p.Name, p.Version, err))
			continue
		}

		resolvedPkgs = append(resolvedPkgs, model.ResolvedPackage{
			Name:     p.Name,
			Version:  p.Version,
			Registry: registry,
			Scope:    p.EffectiveScope(),
			Files:    files,
		})
	}

	// Resolve claude_md entries.
	for _, c := range m.ClaudeMD {
		if !update {
			if _, locked := lockedMD[c.Path]; locked {
				continue
			}
		}

		registry := c.EffectiveRegistry(m.Registry)
		content, hash, err := res.ResolveFile(c.Namespace, c.Version, registry)
		if err != nil {
			slog.Warn("resolve failed", "namespace", c.Namespace, "version", c.Version, "error", err)
			resolveErrors = append(resolveErrors, fmt.Sprintf("claude_md %s@%s: %v", c.Namespace, c.Version, err))
			continue
		}

		resolvedMDs = append(resolvedMDs, writer.ResolvedFile{
			Dir:     c.Path,
			Content: content,
			Hash:    hash,
		})
	}

	// 6. Detect conflicts across packages.
	if conflicts := conflict.Detect(resolvedPkgs); len(conflicts) != 0 {
		var msgs []string
		for _, c := range conflicts {
			msgs = append(msgs, c.String())
		}
		return fmt.Errorf("package conflicts detected:\n  %s", strings.Join(msgs, "\n  "))
	}

	// 7. Write files.
	managedFiles := lockfile.ManagedFiles(lf)

	var pkgWriteCount int
	var pkgSkipCount int
	for _, pkg := range resolvedPkgs {
		root, err := targetRoot(repoRoot, pkg.Scope)
		if err != nil {
			return err
		}
		result, err := writer.WritePackageFiles(root, pkg, managedFiles)
		if err != nil {
			return fmt.Errorf("writing package %s: %w", pkg.Name, err)
		}
		pkgWriteCount += len(result.Written)
		pkgSkipCount += len(result.Skipped)
		for _, s := range result.Skipped {
			slog.Warn("skipped", "file", s.Path, "reason", s.Reason)
		}
		if len(result.Written) > 0 {
			slog.Info("installed package", "name", pkg.Name, "version", pkg.Version, "files", len(result.Written))
		}
	}

	// Write CLAUDE.md files using the v1 writer (it handles managed header + safety checks).
	// Build a v1 lock for the writer's managed-file check.
	v1Lock := model.LockFile{Version: 1}
	for _, e := range lf.ClaudeMD {
		v1Lock.Entries = append(v1Lock.Entries, model.LockEntry{
			Path:      e.Path,
			Namespace: e.Namespace,
			Version:   e.Version,
			Registry:  e.Registry,
			Hash:      e.Hash,
		})
	}
	mdResult, err := writer.WriteFiles(repoRoot, resolvedMDs, v1Lock)
	if err != nil {
		return fmt.Errorf("writing CLAUDE.md files: %w", err)
	}
	for _, s := range mdResult.Skipped {
		slog.Warn("skipped", "path", s.Path, "reason", s.Reason)
	}

	// Handle active deletion for removed packages (only on --update).
	if update {
		for _, d := range pkgDrifts {
			if d.Type != lockfile.DriftRemoved {
				continue
			}
			if locked, ok := lockedPkgs[d.Name]; ok {
				root, err := targetRoot(repoRoot, locked.Scope)
				if err != nil {
					slog.Warn("could not resolve target for deletion", "package", d.Name, "error", err)
					continue
				}
				errs := writer.DeletePackageFiles(root, locked.Files)
				for _, e := range errs {
					slog.Warn("delete failed", "error", e)
				}
				if len(errs) == 0 {
					slog.Info("removed package files", "name", d.Name)
				}
			}
		}
	}

	// 8. Update lock file.
	newLock := buildV2Lock(m, lf, resolvedPkgs, resolvedMDs, mdResult)
	if err := lockfile.WriteV2(repoRoot, newLock); err != nil {
		return fmt.Errorf("writing lock file: %w", err)
	}

	// Summary.
	slog.Info("install complete",
		"packages_installed", len(resolvedPkgs),
		"package_files_written", pkgWriteCount,
		"package_files_skipped", pkgSkipCount,
		"claude_md_written", len(mdResult.Written),
		"claude_md_skipped", len(mdResult.Skipped),
		"failed", len(resolveErrors),
	)

	if len(resolveErrors) > 0 {
		slog.Warn("the following entries could not be resolved:")
		for _, e := range resolveErrors {
			slog.Warn("  " + e)
		}
	}

	return nil
}

// buildV2Lock constructs the new lock file from resolved results and carried-forward entries.
func buildV2Lock(
	m model.Manifest,
	existingLock model.LockFileV2,
	resolvedPkgs []model.ResolvedPackage,
	resolvedMDs []writer.ResolvedFile,
	mdResult writer.WriteResult,
) model.LockFileV2 {
	newLock := model.LockFileV2{Version: 2}

	// Package entries: newly resolved + carried forward.
	resolvedPkgNames := make(map[string]bool)
	for _, pkg := range resolvedPkgs {
		resolvedPkgNames[pkg.Name] = true
		var files []model.LockedFile
		for _, f := range pkg.Files {
			files = append(files, model.LockedFile{Path: f.RelativePath, Hash: f.Hash})
		}
		newLock.Packages = append(newLock.Packages, model.PackageLockEntry{
			Name:     pkg.Name,
			Version:  pkg.Version,
			Registry: pkg.Registry,
			Scope:    pkg.Scope,
			Files:    files,
		})
	}

	// Carry forward packages that weren't re-resolved and still exist in manifest.
	manifestPkgs := make(map[string]bool)
	for _, p := range m.Packages {
		manifestPkgs[p.Name] = true
	}
	for _, e := range existingLock.Packages {
		if resolvedPkgNames[e.Name] {
			continue
		}
		if manifestPkgs[e.Name] {
			newLock.Packages = append(newLock.Packages, e)
		}
	}

	// ClaudeMD entries: newly resolved + carried forward.
	resolvedMDPaths := make(map[string]bool)
	for _, rf := range resolvedMDs {
		// Only include if actually written (not skipped).
		written := false
		for _, w := range mdResult.Written {
			if w == rf.Dir {
				written = true
				break
			}
		}
		if !written {
			continue
		}
		resolvedMDPaths[rf.Dir] = true
		// Find the manifest entry to get namespace/registry.
		for _, c := range m.ClaudeMD {
			if c.Path == rf.Dir {
				newLock.ClaudeMD = append(newLock.ClaudeMD, model.ClaudeMDLock{
					Path:      c.Path,
					Namespace: c.Namespace,
					Version:   c.Version,
					Registry:  c.EffectiveRegistry(m.Registry),
					Hash:      rf.Hash,
				})
				break
			}
		}
	}

	// Carry forward claude_md entries that weren't re-resolved and still exist in manifest.
	manifestMD := make(map[string]bool)
	for _, c := range m.ClaudeMD {
		manifestMD[c.Path] = true
	}
	for _, e := range existingLock.ClaudeMD {
		if resolvedMDPaths[e.Path] {
			continue
		}
		if manifestMD[e.Path] {
			newLock.ClaudeMD = append(newLock.ClaudeMD, e)
		}
	}

	return newLock
}
