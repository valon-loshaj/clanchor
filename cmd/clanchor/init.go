package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/valon-loshaj/clanchor/internal/manifest"
	"github.com/valon-loshaj/clanchor/internal/model"
)

func runInit(registry string) error {
	if !model.ValidOrgRepo(registry) {
		return fmt.Errorf("registry %q is not valid org/repo format (e.g. myorg/claude-registry)", registry)
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(repoRoot, manifest.FileName)
	if _, err := os.Stat(manifestPath); err == nil {
		return fmt.Errorf("%s already exists in %s", manifest.FileName, repoRoot)
	}

	m := model.Manifest{
		Version:  2,
		Registry: registry,
	}

	if err := manifest.Write(repoRoot, m); err != nil {
		return err
	}

	slog.Info("initialized manifest", "path", manifestPath, "registry", registry)
	return nil
}
