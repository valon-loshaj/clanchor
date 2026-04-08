package conflict

import (
	"testing"

	"github.com/valon-loshaj/clanchor/internal/model"
)

func TestDetect_NoConflicts(t *testing.T) {
	packages := []model.ResolvedPackage{
		{
			Name: "acme/pkg-a",
			Files: []model.ResolvedFile{
				{RelativePath: ".claude/skills/a/SKILL.md"},
			},
		},
		{
			Name: "acme/pkg-b",
			Files: []model.ResolvedFile{
				{RelativePath: ".claude/skills/b/SKILL.md"},
			},
		},
	}
	conflicts := Detect(packages)
	if len(conflicts) != 0 {
		t.Errorf("got %d conflicts, want 0", len(conflicts))
	}
}

func TestDetect_SingleConflict(t *testing.T) {
	packages := []model.ResolvedPackage{
		{
			Name: "acme/pkg-a",
			Files: []model.ResolvedFile{
				{RelativePath: ".claude/skills/shared/SKILL.md"},
				{RelativePath: ".claude/agents/unique-a.md"},
			},
		},
		{
			Name: "acme/pkg-b",
			Files: []model.ResolvedFile{
				{RelativePath: ".claude/skills/shared/SKILL.md"},
				{RelativePath: ".claude/agents/unique-b.md"},
			},
		},
	}
	conflicts := Detect(packages)
	if len(conflicts) != 1 {
		t.Fatalf("got %d conflicts, want 1", len(conflicts))
	}
	if conflicts[0].Path != ".claude/skills/shared/SKILL.md" {
		t.Errorf("path = %q, want %q", conflicts[0].Path, ".claude/skills/shared/SKILL.md")
	}
	if len(conflicts[0].Packages) != 2 {
		t.Errorf("packages = %d, want 2", len(conflicts[0].Packages))
	}
}

func TestDetect_MultipleConflicts(t *testing.T) {
	packages := []model.ResolvedPackage{
		{
			Name: "a",
			Files: []model.ResolvedFile{
				{RelativePath: ".claude/skills/x/SKILL.md"},
				{RelativePath: ".claude/agents/y.md"},
			},
		},
		{
			Name: "b",
			Files: []model.ResolvedFile{
				{RelativePath: ".claude/skills/x/SKILL.md"},
				{RelativePath: ".claude/agents/y.md"},
			},
		},
	}
	conflicts := Detect(packages)
	if len(conflicts) != 2 {
		t.Fatalf("got %d conflicts, want 2", len(conflicts))
	}
}

func TestDetect_ThreeWayConflict(t *testing.T) {
	packages := []model.ResolvedPackage{
		{Name: "a", Files: []model.ResolvedFile{{RelativePath: ".claude/skills/x/SKILL.md"}}},
		{Name: "b", Files: []model.ResolvedFile{{RelativePath: ".claude/skills/x/SKILL.md"}}},
		{Name: "c", Files: []model.ResolvedFile{{RelativePath: ".claude/skills/x/SKILL.md"}}},
	}
	conflicts := Detect(packages)
	if len(conflicts) != 1 {
		t.Fatalf("got %d conflicts, want 1", len(conflicts))
	}
	if len(conflicts[0].Packages) != 3 {
		t.Errorf("packages = %d, want 3", len(conflicts[0].Packages))
	}
}

func TestDetect_EmptyInput(t *testing.T) {
	conflicts := Detect(nil)
	if len(conflicts) != 0 {
		t.Errorf("got %d conflicts, want 0", len(conflicts))
	}
}
