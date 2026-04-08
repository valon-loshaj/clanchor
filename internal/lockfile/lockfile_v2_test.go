package lockfile

import (
	"testing"

	"github.com/valon-loshaj/clanchor/internal/model"
)

func TestReadWriteV2_RoundTrip(t *testing.T) {
	root := t.TempDir()
	lf := model.LockFileV2{
		Version: 2,
		Packages: []model.PackageLockEntry{
			{
				Name:     "acme/z-pkg",
				Version:  "1.0.0",
				Registry: "org/repo",
				Scope:    model.ScopeProject,
				Files: []model.LockedFile{
					{Path: ".claude/skills/b/SKILL.md", Hash: "sha256:abc"},
					{Path: ".claude/skills/a/SKILL.md", Hash: "sha256:def"},
				},
			},
			{
				Name:     "acme/a-pkg",
				Version:  "2.0.0",
				Registry: "org/repo",
				Scope:    model.ScopeGlobal,
				Files: []model.LockedFile{
					{Path: ".claude/agents/reviewer.md", Hash: "sha256:ghi"},
				},
			},
		},
		ClaudeMD: []model.ClaudeMDLock{
			{Path: "services/b/CLAUDE.md", Namespace: "acme/b", Version: "1.0.0", Registry: "org/repo", Hash: "sha256:jkl"},
			{Path: "CLAUDE.md", Namespace: "acme/root", Version: "1.0.0", Registry: "org/repo", Hash: "sha256:mno"},
		},
	}

	if err := WriteV2(root, lf); err != nil {
		t.Fatalf("WriteV2: %v", err)
	}

	got, err := ReadV2(root)
	if err != nil {
		t.Fatalf("ReadV2: %v", err)
	}

	if got.Version != 2 {
		t.Errorf("version = %d, want 2", got.Version)
	}
	// Packages should be sorted by name.
	if got.Packages[0].Name != "acme/a-pkg" {
		t.Errorf("first package = %q, want %q", got.Packages[0].Name, "acme/a-pkg")
	}
	// Files within packages should be sorted by path.
	if got.Packages[1].Files[0].Path != ".claude/skills/a/SKILL.md" {
		t.Errorf("first file = %q, want %q", got.Packages[1].Files[0].Path, ".claude/skills/a/SKILL.md")
	}
	// ClaudeMD should be sorted by path.
	if got.ClaudeMD[0].Path != "CLAUDE.md" {
		t.Errorf("first claude_md = %q, want %q", got.ClaudeMD[0].Path, "CLAUDE.md")
	}
}

func TestReadV2_FirstRun(t *testing.T) {
	root := t.TempDir()
	lf, err := ReadV2(root)
	if err != nil {
		t.Fatalf("ReadV2: %v", err)
	}
	if lf.Version != 2 {
		t.Errorf("version = %d, want 2", lf.Version)
	}
	if len(lf.Packages) != 0 {
		t.Errorf("packages = %d, want 0", len(lf.Packages))
	}
	if len(lf.ClaudeMD) != 0 {
		t.Errorf("claude_md = %d, want 0", len(lf.ClaudeMD))
	}
}

func TestDiffV2Packages_NoChanges(t *testing.T) {
	packages := []model.PackageEntry{{Name: "acme/svc", Version: "1.0.0"}}
	lf := model.LockFileV2{
		Packages: []model.PackageLockEntry{{Name: "acme/svc", Version: "1.0.0"}},
	}
	drifts := DiffV2Packages(packages, "", lf)
	if len(drifts) != 0 {
		t.Errorf("got %d drifts, want 0", len(drifts))
	}
}

func TestDiffV2Packages_Added(t *testing.T) {
	packages := []model.PackageEntry{{Name: "acme/new", Version: "1.0.0"}}
	lf := model.LockFileV2{}
	drifts := DiffV2Packages(packages, "", lf)
	if len(drifts) != 1 || drifts[0].Type != DriftAdded {
		t.Errorf("expected 1 added drift, got %+v", drifts)
	}
}

func TestDiffV2Packages_Removed(t *testing.T) {
	packages := []model.PackageEntry{}
	lf := model.LockFileV2{
		Packages: []model.PackageLockEntry{{Name: "acme/old", Version: "1.0.0"}},
	}
	drifts := DiffV2Packages(packages, "", lf)
	if len(drifts) != 1 || drifts[0].Type != DriftRemoved {
		t.Errorf("expected 1 removed drift, got %+v", drifts)
	}
}

func TestDiffV2Packages_VersionChanged(t *testing.T) {
	packages := []model.PackageEntry{{Name: "acme/svc", Version: "2.0.0"}}
	lf := model.LockFileV2{
		Packages: []model.PackageLockEntry{{Name: "acme/svc", Version: "1.0.0"}},
	}
	drifts := DiffV2Packages(packages, "", lf)
	if len(drifts) != 1 || drifts[0].Type != DriftVersionChanged {
		t.Errorf("expected 1 version_changed drift, got %+v", drifts)
	}
}

func TestDiffV2ClaudeMD_AllDriftTypes(t *testing.T) {
	entries := []model.ClaudeMDEntry{
		{Path: ".", Version: "2.0.0"},           // version changed
		{Path: "services/new", Version: "1.0.0"}, // added
	}
	lf := model.LockFileV2{
		ClaudeMD: []model.ClaudeMDLock{
			{Path: ".", Version: "1.0.0"},
			{Path: "services/old", Version: "1.0.0"}, // removed
		},
	}
	drifts := DiffV2ClaudeMD(entries, lf)
	if len(drifts) != 3 {
		t.Fatalf("got %d drifts, want 3", len(drifts))
	}

	types := make(map[DriftType]int)
	for _, d := range drifts {
		types[d.Type]++
	}
	if types[DriftAdded] != 1 || types[DriftRemoved] != 1 || types[DriftVersionChanged] != 1 {
		t.Errorf("drift types = %+v, want 1 of each", types)
	}
}

func TestManagedFiles(t *testing.T) {
	lf := model.LockFileV2{
		Packages: []model.PackageLockEntry{
			{
				Name: "acme/svc",
				Files: []model.LockedFile{
					{Path: ".claude/skills/a/SKILL.md"},
					{Path: ".claude/agents/b.md"},
				},
			},
		},
	}
	managed := ManagedFiles(lf)
	if !managed[".claude/skills/a/SKILL.md"] {
		t.Error("expected .claude/skills/a/SKILL.md to be managed")
	}
	if !managed[".claude/agents/b.md"] {
		t.Error("expected .claude/agents/b.md to be managed")
	}
	if managed[".claude/skills/unknown/SKILL.md"] {
		t.Error("unexpected file marked as managed")
	}
}
