package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/valon-loshaj/clanchor/internal/manifest"
	"github.com/valon-loshaj/clanchor/internal/model"
)

func TestInit_CreatesManifest(t *testing.T) {
	root := setupV2TestRepo(t)
	oldDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(oldDir)

	if err := runInit("myorg/claude-registry"); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	m, err := manifest.Read(root)
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}
	if m.Version != 2 {
		t.Errorf("version = %d, want 2", m.Version)
	}
	if m.Registry != "myorg/claude-registry" {
		t.Errorf("registry = %q, want %q", m.Registry, "myorg/claude-registry")
	}
	if len(m.Packages) != 0 {
		t.Errorf("packages = %d, want 0", len(m.Packages))
	}
	if len(m.ClaudeMD) != 0 {
		t.Errorf("claude_md = %d, want 0", len(m.ClaudeMD))
	}
}

func TestInit_RefusesOverwrite(t *testing.T) {
	root := setupV2TestRepo(t)
	oldDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(oldDir)

	writeManifest(t, root, model.Manifest{
		Version:  2,
		Registry: "org/repo",
	})

	err := runInit("other/registry")
	if err == nil {
		t.Fatal("expected error when manifest exists")
	}
}

func TestInit_InvalidRegistry(t *testing.T) {
	root := setupV2TestRepo(t)
	oldDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(oldDir)

	err := runInit("not-a-valid-registry")
	if err == nil {
		t.Fatal("expected error for invalid registry format")
	}

	// Manifest should not have been created.
	if _, err := os.Stat(filepath.Join(root, "clanchor.json")); !os.IsNotExist(err) {
		t.Error("manifest was created despite invalid registry")
	}
}

func TestInit_NotInGitRepo(t *testing.T) {
	dir := t.TempDir() // no .git
	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	err := runInit("myorg/registry")
	if err == nil {
		t.Fatal("expected error outside git repo")
	}
}
