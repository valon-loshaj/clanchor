package main

import (
	"fmt"
	"log/slog"

	"github.com/valon-loshaj/clanchor/internal/lockfile"
	"github.com/valon-loshaj/clanchor/internal/manifest"
)

func runStatus() error {
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

	// Show installed packages.
	if len(lf.Packages) > 0 {
		fmt.Println("Installed packages:")
		for _, pkg := range lf.Packages {
			fmt.Printf("  %s@%s (%s, %d files)\n", pkg.Name, pkg.Version, pkg.Scope, len(pkg.Files))
		}
	}

	if len(lf.ClaudeMD) > 0 {
		fmt.Println("Installed CLAUDE.md files:")
		for _, md := range lf.ClaudeMD {
			fmt.Printf("  %s → %s@%s\n", md.Path, md.Namespace, md.Version)
		}
	}

	if len(lf.Packages) == 0 && len(lf.ClaudeMD) == 0 {
		fmt.Println("No packages installed. Run 'clanchor install' to install from manifest.")
		return nil
	}

	// Check for drift.
	pkgDrifts := lockfile.DiffV2Packages(m.Packages, m.Registry, lf)
	mdDrifts := lockfile.DiffV2ClaudeMD(m.ClaudeMD, lf)

	if len(pkgDrifts) == 0 && len(mdDrifts) == 0 {
		fmt.Println("\nNo drift detected. Lock file is up to date.")
		return nil
	}

	fmt.Println("\nDrift detected:")
	for _, d := range pkgDrifts {
		switch d.Type {
		case lockfile.DriftAdded:
			fmt.Printf("  + package %s@%s (new)\n", d.Name, d.NewVersion)
		case lockfile.DriftRemoved:
			fmt.Printf("  - package %s (removed from manifest)\n", d.Name)
		case lockfile.DriftVersionChanged:
			fmt.Printf("  ~ package %s %s → %s\n", d.Name, d.OldVersion, d.NewVersion)
		}
	}
	for _, d := range mdDrifts {
		switch d.Type {
		case lockfile.DriftAdded:
			fmt.Printf("  + claude_md %s@%s (new)\n", d.Path, d.NewVersion)
		case lockfile.DriftRemoved:
			fmt.Printf("  - claude_md %s (removed from manifest)\n", d.Path)
		case lockfile.DriftVersionChanged:
			fmt.Printf("  ~ claude_md %s %s → %s\n", d.Path, d.OldVersion, d.NewVersion)
		}
	}

	slog.Info("run 'clanchor install --update' to reconcile drift")
	return nil
}
