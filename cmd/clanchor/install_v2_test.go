package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valon-loshaj/clanchor/internal/lockfile"
	"github.com/valon-loshaj/clanchor/internal/model"
)

// mockResolverV2 satisfies the Resolver interface for v2 tests.
type mockResolverV2 struct {
	files    map[string]string                // namespace@version -> content (for ResolveFile)
	packages map[string][]model.ResolvedFile  // name@version -> resolved files (for ResolvePackage)
}

func (m *mockResolverV2) ResolveFile(namespace, version, registry string) ([]byte, string, error) {
	key := namespace + "@" + version
	content, ok := m.files[key]
	if !ok {
		return nil, "", fmt.Errorf("not found: %s", key)
	}
	hash := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(content)))
	return []byte(content), hash, nil
}

func (m *mockResolverV2) ResolvePackage(name, version, registry string) ([]model.ResolvedFile, error) {
	key := name + "@" + version
	files, ok := m.packages[key]
	if !ok {
		return nil, fmt.Errorf("not found: %s", key)
	}
	return files, nil
}

func setupV2TestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".git"), 0o755)
	return root
}

func writeManifest(t *testing.T, root string, m model.Manifest) {
	t.Helper()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "clanchor.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func makeResolvedFiles(files map[string]string) []model.ResolvedFile {
	var result []model.ResolvedFile
	for path, content := range files {
		hash := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(content)))
		result = append(result, model.ResolvedFile{
			RelativePath: path,
			Content:      []byte(content),
			Hash:         hash,
		})
	}
	return result
}

func TestV2_FreshInstall_PackageAndClaudeMD(t *testing.T) {
	root := setupV2TestRepo(t)
	oldDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(oldDir)

	writeManifest(t, root, model.Manifest{
		Version:  2,
		Registry: "org/repo",
		Packages: []model.PackageEntry{
			{Name: "acme/go-tools", Version: "1.0.0", Scope: model.ScopeProject},
		},
		ClaudeMD: []model.ClaudeMDEntry{
			{Path: ".", Namespace: "acme/root-ctx", Version: "1.0.0"},
		},
	})

	mock := &mockResolverV2{
		files: map[string]string{
			"acme/root-ctx@1.0.0": "# Root context\n",
		},
		packages: map[string][]model.ResolvedFile{
			"acme/go-tools@1.0.0": makeResolvedFiles(map[string]string{
				".claude/skills/lint/SKILL.md": "# Lint skill\n",
				".claude/agents/reviewer.md":   "# Reviewer\n",
			}),
		},
	}

	err := runInstallV2(false, mock)
	if err != nil {
		t.Fatalf("runInstallV2: %v", err)
	}

	// Verify package files written.
	skillData, err := os.ReadFile(filepath.Join(root, ".claude/skills/lint/SKILL.md"))
	if err != nil {
		t.Fatalf("skill file not written: %v", err)
	}
	if string(skillData) != "# Lint skill\n" {
		t.Errorf("skill content = %q", skillData)
	}

	agentData, err := os.ReadFile(filepath.Join(root, ".claude/agents/reviewer.md"))
	if err != nil {
		t.Fatalf("agent file not written: %v", err)
	}
	if string(agentData) != "# Reviewer\n" {
		t.Errorf("agent content = %q", agentData)
	}

	// Verify CLAUDE.md written with managed header.
	mdData, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("CLAUDE.md not written: %v", err)
	}
	if !strings.Contains(string(mdData), "managed by clanchor") {
		t.Error("CLAUDE.md missing managed header")
	}
	if !strings.Contains(string(mdData), "# Root context") {
		t.Error("CLAUDE.md missing content")
	}

	// Verify lock file.
	lf, err := lockfile.ReadV2(root)
	if err != nil {
		t.Fatalf("ReadV2: %v", err)
	}
	if lf.Version != 2 {
		t.Errorf("lock version = %d, want 2", lf.Version)
	}
	if len(lf.Packages) != 1 {
		t.Errorf("lock packages = %d, want 1", len(lf.Packages))
	}
	if len(lf.Packages[0].Files) != 2 {
		t.Errorf("lock package files = %d, want 2", len(lf.Packages[0].Files))
	}
	if len(lf.ClaudeMD) != 1 {
		t.Errorf("lock claude_md = %d, want 1", len(lf.ClaudeMD))
	}
}

func TestV2_NoDrift_SecondRun(t *testing.T) {
	root := setupV2TestRepo(t)
	oldDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(oldDir)

	writeManifest(t, root, model.Manifest{
		Version:  2,
		Registry: "org/repo",
		Packages: []model.PackageEntry{
			{Name: "acme/tools", Version: "1.0.0"},
		},
	})

	mock := &mockResolverV2{
		packages: map[string][]model.ResolvedFile{
			"acme/tools@1.0.0": makeResolvedFiles(map[string]string{
				".claude/skills/x/SKILL.md": "# X\n",
			}),
		},
		files: map[string]string{},
	}

	// First run.
	if err := runInstallV2(false, mock); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Second run — should not re-resolve (already locked).
	failMock := &mockResolverV2{
		packages: map[string][]model.ResolvedFile{}, // empty — would fail if called
		files:    map[string]string{},
	}
	if err := runInstallV2(false, failMock); err != nil {
		t.Fatalf("second install: %v", err)
	}
}

func TestV2_DriftDetected_BlocksWithoutUpdate(t *testing.T) {
	root := setupV2TestRepo(t)
	oldDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(oldDir)

	writeManifest(t, root, model.Manifest{
		Version:  2,
		Registry: "org/repo",
		Packages: []model.PackageEntry{
			{Name: "acme/tools", Version: "1.0.0"},
		},
	})

	mock := &mockResolverV2{
		packages: map[string][]model.ResolvedFile{
			"acme/tools@1.0.0": makeResolvedFiles(map[string]string{
				".claude/skills/x/SKILL.md": "# X\n",
			}),
		},
		files: map[string]string{},
	}

	if err := runInstallV2(false, mock); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Bump version in manifest.
	writeManifest(t, root, model.Manifest{
		Version:  2,
		Registry: "org/repo",
		Packages: []model.PackageEntry{
			{Name: "acme/tools", Version: "2.0.0"},
		},
	})

	err := runInstallV2(false, mock)
	if err == nil {
		t.Fatal("expected drift error")
	}
	if !strings.Contains(err.Error(), "drift detected") {
		t.Errorf("error = %q, want drift message", err)
	}
}

func TestV2_UpdateReconcilesDrift(t *testing.T) {
	root := setupV2TestRepo(t)
	oldDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(oldDir)

	writeManifest(t, root, model.Manifest{
		Version:  2,
		Registry: "org/repo",
		Packages: []model.PackageEntry{
			{Name: "acme/tools", Version: "1.0.0"},
		},
	})

	mock := &mockResolverV2{
		packages: map[string][]model.ResolvedFile{
			"acme/tools@1.0.0": makeResolvedFiles(map[string]string{
				".claude/skills/x/SKILL.md": "# V1\n",
			}),
			"acme/tools@2.0.0": makeResolvedFiles(map[string]string{
				".claude/skills/x/SKILL.md": "# V2\n",
			}),
		},
		files: map[string]string{},
	}

	if err := runInstallV2(false, mock); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Bump version.
	writeManifest(t, root, model.Manifest{
		Version:  2,
		Registry: "org/repo",
		Packages: []model.PackageEntry{
			{Name: "acme/tools", Version: "2.0.0"},
		},
	})

	if err := runInstallV2(true, mock); err != nil {
		t.Fatalf("update install: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(root, ".claude/skills/x/SKILL.md"))
	if string(data) != "# V2\n" {
		t.Errorf("content = %q, want %q", data, "# V2\n")
	}

	lf, _ := lockfile.ReadV2(root)
	if lf.Packages[0].Version != "2.0.0" {
		t.Errorf("lock version = %q, want %q", lf.Packages[0].Version, "2.0.0")
	}
}

func TestV2_ConflictDetection(t *testing.T) {
	root := setupV2TestRepo(t)
	oldDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(oldDir)

	writeManifest(t, root, model.Manifest{
		Version:  2,
		Registry: "org/repo",
		Packages: []model.PackageEntry{
			{Name: "acme/pkg-a", Version: "1.0.0"},
			{Name: "acme/pkg-b", Version: "1.0.0"},
		},
	})

	// Both packages provide the same file.
	mock := &mockResolverV2{
		packages: map[string][]model.ResolvedFile{
			"acme/pkg-a@1.0.0": makeResolvedFiles(map[string]string{
				".claude/skills/shared/SKILL.md": "# From A\n",
			}),
			"acme/pkg-b@1.0.0": makeResolvedFiles(map[string]string{
				".claude/skills/shared/SKILL.md": "# From B\n",
			}),
		},
		files: map[string]string{},
	}

	err := runInstallV2(false, mock)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "conflicts detected") {
		t.Errorf("error = %q, want conflict message", err)
	}
}

func TestV2_ActiveDeletion_OnUpdate(t *testing.T) {
	root := setupV2TestRepo(t)
	oldDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(oldDir)

	writeManifest(t, root, model.Manifest{
		Version:  2,
		Registry: "org/repo",
		Packages: []model.PackageEntry{
			{Name: "acme/tools", Version: "1.0.0"},
		},
	})

	mock := &mockResolverV2{
		packages: map[string][]model.ResolvedFile{
			"acme/tools@1.0.0": makeResolvedFiles(map[string]string{
				".claude/skills/x/SKILL.md": "# X\n",
			}),
		},
		files: map[string]string{},
	}

	if err := runInstallV2(false, mock); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Verify file exists.
	skillPath := filepath.Join(root, ".claude/skills/x/SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Fatalf("skill file not created: %v", err)
	}

	// Remove package from manifest.
	writeManifest(t, root, model.Manifest{
		Version:  2,
		Registry: "org/repo",
	})

	if err := runInstallV2(true, mock); err != nil {
		t.Fatalf("update install: %v", err)
	}

	// File should be deleted.
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Error("skill file not deleted after package removal")
	}

	// Lock should be empty.
	lf, _ := lockfile.ReadV2(root)
	if len(lf.Packages) != 0 {
		t.Errorf("lock packages = %d, want 0", len(lf.Packages))
	}
}

func TestV2_NewPackageAddedAfterFirstRun(t *testing.T) {
	root := setupV2TestRepo(t)
	oldDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(oldDir)

	writeManifest(t, root, model.Manifest{
		Version:  2,
		Registry: "org/repo",
		Packages: []model.PackageEntry{
			{Name: "acme/first", Version: "1.0.0"},
		},
	})

	mock := &mockResolverV2{
		packages: map[string][]model.ResolvedFile{
			"acme/first@1.0.0": makeResolvedFiles(map[string]string{
				".claude/skills/a/SKILL.md": "# A\n",
			}),
			"acme/second@1.0.0": makeResolvedFiles(map[string]string{
				".claude/skills/b/SKILL.md": "# B\n",
			}),
		},
		files: map[string]string{},
	}

	if err := runInstallV2(false, mock); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Add a second package — should be non-blocking drift.
	writeManifest(t, root, model.Manifest{
		Version:  2,
		Registry: "org/repo",
		Packages: []model.PackageEntry{
			{Name: "acme/first", Version: "1.0.0"},
			{Name: "acme/second", Version: "1.0.0"},
		},
	})

	if err := runInstallV2(false, mock); err != nil {
		t.Fatalf("second install: %v", err)
	}

	// Both files should exist.
	if _, err := os.Stat(filepath.Join(root, ".claude/skills/a/SKILL.md")); err != nil {
		t.Error("first package file missing")
	}
	if _, err := os.Stat(filepath.Join(root, ".claude/skills/b/SKILL.md")); err != nil {
		t.Error("second package file missing")
	}

	lf, _ := lockfile.ReadV2(root)
	if len(lf.Packages) != 2 {
		t.Errorf("lock packages = %d, want 2", len(lf.Packages))
	}
}

func TestV2_GlobalScope_WritesToHome(t *testing.T) {
	root := setupV2TestRepo(t)
	fakeHome := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(oldDir)

	// Override HOME so targetRoot resolves global scope to our temp dir.
	t.Setenv("HOME", fakeHome)

	writeManifest(t, root, model.Manifest{
		Version:  2,
		Registry: "org/repo",
		Packages: []model.PackageEntry{
			{Name: "acme/global-skill", Version: "1.0.0", Scope: model.ScopeGlobal},
			{Name: "acme/project-skill", Version: "1.0.0", Scope: model.ScopeProject},
		},
	})

	mock := &mockResolverV2{
		packages: map[string][]model.ResolvedFile{
			"acme/global-skill@1.0.0": makeResolvedFiles(map[string]string{
				".claude/skills/global/SKILL.md": "# Global skill\n",
			}),
			"acme/project-skill@1.0.0": makeResolvedFiles(map[string]string{
				".claude/skills/project/SKILL.md": "# Project skill\n",
			}),
		},
		files: map[string]string{},
	}

	if err := runInstallV2(false, mock); err != nil {
		t.Fatalf("runInstallV2: %v", err)
	}

	// Global skill should be in fakeHome.
	globalPath := filepath.Join(fakeHome, ".claude/skills/global/SKILL.md")
	if _, err := os.Stat(globalPath); err != nil {
		t.Errorf("global skill not written to home: %v", err)
	}

	// Project skill should be in repo root.
	projectPath := filepath.Join(root, ".claude/skills/project/SKILL.md")
	if _, err := os.Stat(projectPath); err != nil {
		t.Errorf("project skill not written to repo: %v", err)
	}

	// Global skill should NOT be in repo root.
	wrongPath := filepath.Join(root, ".claude/skills/global/SKILL.md")
	if _, err := os.Stat(wrongPath); !os.IsNotExist(err) {
		t.Error("global skill was incorrectly written to repo root")
	}

	// Lock file should track both.
	lf, _ := lockfile.ReadV2(root)
	if len(lf.Packages) != 2 {
		t.Errorf("lock packages = %d, want 2", len(lf.Packages))
	}
}
