package writer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/valon-loshaj/clanchor/internal/model"
)

func TestWritePackageFiles_NewFiles(t *testing.T) {
	root := t.TempDir()
	pkg := model.ResolvedPackage{
		Name: "acme/go-backend",
		Files: []model.ResolvedFile{
			{RelativePath: ".claude/skills/review/SKILL.md", Content: []byte("# Review skill\n"), Hash: "sha256:abc"},
			{RelativePath: ".claude/agents/reviewer.md", Content: []byte("# Reviewer agent\n"), Hash: "sha256:def"},
		},
	}

	result, err := WritePackageFiles(root, pkg, nil)
	if err != nil {
		t.Fatalf("WritePackageFiles: %v", err)
	}
	if len(result.Written) != 2 {
		t.Errorf("written = %d, want 2", len(result.Written))
	}
	if len(result.Skipped) != 0 {
		t.Errorf("skipped = %d, want 0", len(result.Skipped))
	}

	data, _ := os.ReadFile(filepath.Join(root, ".claude/skills/review/SKILL.md"))
	if string(data) != "# Review skill\n" {
		t.Errorf("content = %q, want %q", data, "# Review skill\n")
	}
}

func TestWritePackageFiles_OverwriteManaged(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".claude/skills/review/SKILL.md")
	os.MkdirAll(filepath.Dir(target), 0o755)
	os.WriteFile(target, []byte("old content"), 0o644)

	managed := map[string]bool{".claude/skills/review/SKILL.md": true}
	pkg := model.ResolvedPackage{
		Name: "acme/go-backend",
		Files: []model.ResolvedFile{
			{RelativePath: ".claude/skills/review/SKILL.md", Content: []byte("new content"), Hash: "sha256:abc"},
		},
	}

	result, err := WritePackageFiles(root, pkg, managed)
	if err != nil {
		t.Fatalf("WritePackageFiles: %v", err)
	}
	if len(result.Written) != 1 {
		t.Errorf("written = %d, want 1", len(result.Written))
	}

	data, _ := os.ReadFile(target)
	if string(data) != "new content" {
		t.Errorf("content = %q, want %q", data, "new content")
	}
}

func TestWritePackageFiles_SkipUnmanaged(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, ".claude/skills/review/SKILL.md")
	os.MkdirAll(filepath.Dir(target), 0o755)
	os.WriteFile(target, []byte("user content"), 0o644)

	pkg := model.ResolvedPackage{
		Name: "acme/go-backend",
		Files: []model.ResolvedFile{
			{RelativePath: ".claude/skills/review/SKILL.md", Content: []byte("registry content"), Hash: "sha256:abc"},
		},
	}

	result, err := WritePackageFiles(root, pkg, nil)
	if err != nil {
		t.Fatalf("WritePackageFiles: %v", err)
	}
	if len(result.Written) != 0 {
		t.Errorf("written = %d, want 0", len(result.Written))
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("skipped = %d, want 1", len(result.Skipped))
	}

	data, _ := os.ReadFile(target)
	if string(data) != "user content" {
		t.Errorf("unmanaged file was modified")
	}
}

func TestDeletePackageFiles(t *testing.T) {
	root := t.TempDir()

	// Create files to delete.
	skillPath := filepath.Join(root, ".claude/skills/review/SKILL.md")
	agentPath := filepath.Join(root, ".claude/agents/reviewer.md")
	os.MkdirAll(filepath.Dir(skillPath), 0o755)
	os.MkdirAll(filepath.Dir(agentPath), 0o755)
	os.WriteFile(skillPath, []byte("skill"), 0o644)
	os.WriteFile(agentPath, []byte("agent"), 0o644)

	files := []model.LockedFile{
		{Path: ".claude/skills/review/SKILL.md"},
		{Path: ".claude/agents/reviewer.md"},
	}

	errs := DeletePackageFiles(root, files)
	if len(errs) != 0 {
		t.Fatalf("got %d errors: %v", len(errs), errs)
	}

	if fileExists(skillPath) {
		t.Error("skill file not deleted")
	}
	if fileExists(agentPath) {
		t.Error("agent file not deleted")
	}

	// Empty parent dirs should be cleaned up.
	if fileExists(filepath.Join(root, ".claude/skills/review")) {
		t.Error("empty review dir not cleaned up")
	}
	if fileExists(filepath.Join(root, ".claude/skills")) {
		t.Error("empty skills dir not cleaned up")
	}
	if fileExists(filepath.Join(root, ".claude/agents")) {
		t.Error("empty agents dir not cleaned up")
	}
}

func TestDeletePackageFiles_NonexistentFile(t *testing.T) {
	root := t.TempDir()
	files := []model.LockedFile{
		{Path: ".claude/skills/missing/SKILL.md"},
	}
	errs := DeletePackageFiles(root, files)
	if len(errs) != 0 {
		t.Errorf("got %d errors for nonexistent file, want 0", len(errs))
	}
}

func TestDeletePackageFiles_PreservesNonEmptyDirs(t *testing.T) {
	root := t.TempDir()

	// Create two files in the same skill dir.
	dir := filepath.Join(root, ".claude/skills/review")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("skill"), 0o644)
	os.WriteFile(filepath.Join(dir, "reference.md"), []byte("ref"), 0o644)

	// Only delete one.
	files := []model.LockedFile{
		{Path: ".claude/skills/review/SKILL.md"},
	}
	errs := DeletePackageFiles(root, files)
	if len(errs) != 0 {
		t.Fatalf("got %d errors: %v", len(errs), errs)
	}

	// Dir should still exist because reference.md is still there.
	if !fileExists(filepath.Join(dir, "reference.md")) {
		t.Error("reference.md was deleted")
	}
	if !fileExists(dir) {
		t.Error("dir was cleaned up despite having remaining files")
	}
}
