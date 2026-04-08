package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/valon-loshaj/clanchor/internal/model"
)

func TestRead_Valid(t *testing.T) {
	root := t.TempDir()
	content := `{
		"version": 2,
		"registry": "org/repo",
		"packages": [
			{"name": "acme/svc", "version": "1.0.0", "scope": "project"}
		],
		"claude_md": [
			{"path": ".", "namespace": "acme/ctx", "version": "1.0.0"}
		]
	}`
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := Read(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Version != 2 {
		t.Errorf("version = %d, want 2", m.Version)
	}
	if len(m.Packages) != 1 {
		t.Errorf("packages = %d, want 1", len(m.Packages))
	}
	if len(m.ClaudeMD) != 1 {
		t.Errorf("claude_md = %d, want 1", len(m.ClaudeMD))
	}
	if m.Packages[0].EffectiveRegistry(m.Registry) != "org/repo" {
		t.Errorf("registry = %q, want %q", m.Packages[0].EffectiveRegistry(m.Registry), "org/repo")
	}
}

func TestRead_MissingFile(t *testing.T) {
	root := t.TempDir()
	_, err := Read(root)
	if err == nil {
		t.Fatal("expected error for missing manifest")
	}
}

func TestRead_InvalidJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(`{bad`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Read(root)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestRead_FailsValidation(t *testing.T) {
	root := t.TempDir()
	// version 1 is invalid for v2 manifest
	content := `{"version": 1, "packages": [{"name": "a/b", "version": "1.0.0", "registry": "o/r"}]}`
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Read(root)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRead_DefaultRegistryInheritance(t *testing.T) {
	root := t.TempDir()
	content := `{
		"version": 2,
		"registry": "default/reg",
		"packages": [
			{"name": "a/b", "version": "1.0.0"},
			{"name": "c/d", "version": "2.0.0", "registry": "custom/reg"}
		]
	}`
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := Read(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := m.Packages[0].EffectiveRegistry(m.Registry); got != "default/reg" {
		t.Errorf("packages[0] registry = %q, want %q", got, "default/reg")
	}
	if got := m.Packages[1].EffectiveRegistry(m.Registry); got != "custom/reg" {
		t.Errorf("packages[1] registry = %q, want %q", got, "custom/reg")
	}
}

func TestRead_DefaultScope(t *testing.T) {
	root := t.TempDir()
	content := `{
		"version": 2,
		"registry": "o/r",
		"packages": [{"name": "a/b", "version": "1.0.0"}]
	}`
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := Read(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := m.Packages[0].EffectiveScope(); got != model.ScopeProject {
		t.Errorf("scope = %q, want %q", got, model.ScopeProject)
	}
}
