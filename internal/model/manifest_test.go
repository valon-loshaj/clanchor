package model

import (
	"testing"
)

func TestParseManifest_Valid(t *testing.T) {
	input := `{
		"version": 2,
		"registry": "myorg/claude-registry",
		"packages": [
			{"name": "acme/go-backend", "version": "1.2.0", "scope": "project"},
			{"name": "acme/security-agent", "version": "0.3.0", "scope": "global"}
		],
		"claude_md": [
			{"path": ".", "namespace": "acme/project-context", "version": "1.0.0"},
			{"path": "services/payments", "namespace": "acme/payments", "version": "0.5.0"}
		]
	}`
	m, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Version != 2 {
		t.Errorf("version = %d, want 2", m.Version)
	}
	if len(m.Packages) != 2 {
		t.Errorf("packages count = %d, want 2", len(m.Packages))
	}
	if len(m.ClaudeMD) != 2 {
		t.Errorf("claude_md count = %d, want 2", len(m.ClaudeMD))
	}
}

func TestParseManifest_DefaultRegistryInheritance(t *testing.T) {
	input := `{
		"version": 2,
		"registry": "myorg/default-reg",
		"packages": [
			{"name": "acme/svc", "version": "1.0.0"},
			{"name": "other/svc", "version": "1.0.0", "registry": "other-org/reg"}
		]
	}`
	m, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := m.Packages[0].EffectiveRegistry(m.Registry); got != "myorg/default-reg" {
		t.Errorf("packages[0] registry = %q, want %q", got, "myorg/default-reg")
	}
	if got := m.Packages[1].EffectiveRegistry(m.Registry); got != "other-org/reg" {
		t.Errorf("packages[1] registry = %q, want %q", got, "other-org/reg")
	}
}

func TestParseManifest_DefaultScope(t *testing.T) {
	input := `{
		"version": 2,
		"registry": "org/repo",
		"packages": [
			{"name": "acme/svc", "version": "1.0.0"}
		]
	}`
	m, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := m.Packages[0].EffectiveScope(); got != ScopeProject {
		t.Errorf("default scope = %q, want %q", got, ScopeProject)
	}
}

func TestParseManifest_PackagesOnly(t *testing.T) {
	input := `{
		"version": 2,
		"registry": "org/repo",
		"packages": [{"name": "acme/svc", "version": "1.0.0"}]
	}`
	_, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseManifest_ClaudeMDOnly(t *testing.T) {
	input := `{
		"version": 2,
		"registry": "org/repo",
		"claude_md": [{"path": ".", "namespace": "acme/ctx", "version": "1.0.0"}]
	}`
	_, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseManifest_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"wrong version", `{"version": 1, "packages": [{"name": "a/b", "version": "1.0.0", "registry": "o/r"}]}`},
		{"bad default registry", `{"version": 2, "registry": "noslash", "packages": [{"name": "a/b", "version": "1.0.0"}]}`},
		{"package missing name", `{"version": 2, "registry": "o/r", "packages": [{"version": "1.0.0"}]}`},
		{"package missing version", `{"version": 2, "registry": "o/r", "packages": [{"name": "a/b"}]}`},
		{"package bad semver", `{"version": 2, "registry": "o/r", "packages": [{"name": "a/b", "version": "v1.0.0"}]}`},
		{"package no registry anywhere", `{"version": 2, "packages": [{"name": "a/b", "version": "1.0.0"}]}`},
		{"package bad scope", `{"version": 2, "registry": "o/r", "packages": [{"name": "a/b", "version": "1.0.0", "scope": "invalid"}]}`},
		{"claude_md missing path", `{"version": 2, "registry": "o/r", "claude_md": [{"namespace": "a/b", "version": "1.0.0"}]}`},
		{"claude_md missing namespace", `{"version": 2, "registry": "o/r", "claude_md": [{"path": ".", "version": "1.0.0"}]}`},
		{"claude_md missing version", `{"version": 2, "registry": "o/r", "claude_md": [{"path": ".", "namespace": "a/b"}]}`},
		{"claude_md bad semver", `{"version": 2, "registry": "o/r", "claude_md": [{"path": ".", "namespace": "a/b", "version": "abc"}]}`},
		{"claude_md no registry", `{"version": 2, "claude_md": [{"path": ".", "namespace": "a/b", "version": "1.0.0"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(tt.input))
			if err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestParseManifest_InvalidJSON(t *testing.T) {
	_, err := ParseManifest([]byte(`{not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
