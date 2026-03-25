package writer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valon-loshaj/clanchor/internal/model"
)

func TestWriteFiles_NewFile(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "svc/a"), 0o755)

	resolved := []ResolvedFile{
		{Dir: "svc/a", Content: []byte("# Context\n"), Hash: "abc123"},
	}
	result, err := WriteFiles(root, resolved, model.LockFile{})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}
	if len(result.Written) != 1 {
		t.Fatalf("written = %d, want 1", len(result.Written))
	}

	data, _ := os.ReadFile(filepath.Join(root, "svc/a/CLAUDE.md"))
	if !strings.HasPrefix(string(data), managedHeader) {
		t.Errorf("file missing managed header")
	}
	if !strings.Contains(string(data), "# Context") {
		t.Errorf("file missing content")
	}
}

func TestWriteFiles_OverwriteManaged(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "svc/a"), 0o755)
	os.WriteFile(filepath.Join(root, "svc/a/CLAUDE.md"), []byte(managedHeader+"old"), 0o644)

	lock := model.LockFile{
		Entries: []model.LockEntry{{Path: "svc/a", Version: "1.0.0"}},
	}
	resolved := []ResolvedFile{
		{Dir: "svc/a", Content: []byte("new content\n"), Hash: "def456"},
	}

	result, err := WriteFiles(root, resolved, lock)
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}
	if len(result.Written) != 1 {
		t.Errorf("written = %d, want 1", len(result.Written))
	}
	if len(result.Skipped) != 0 {
		t.Errorf("skipped = %d, want 0", len(result.Skipped))
	}

	data, _ := os.ReadFile(filepath.Join(root, "svc/a/CLAUDE.md"))
	if !strings.Contains(string(data), "new content") {
		t.Errorf("file not updated")
	}
}

func TestWriteFiles_SkipUnmanaged(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "svc/a"), 0o755)
	os.WriteFile(filepath.Join(root, "svc/a/CLAUDE.md"), []byte("hand-written context"), 0o644)

	resolved := []ResolvedFile{
		{Dir: "svc/a", Content: []byte("registry content\n"), Hash: "abc"},
	}

	// Empty lock — so the existing file is unmanaged.
	result, err := WriteFiles(root, resolved, model.LockFile{})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}
	if len(result.Written) != 0 {
		t.Errorf("written = %d, want 0", len(result.Written))
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("skipped = %d, want 1", len(result.Skipped))
	}

	// Verify original file is untouched.
	data, _ := os.ReadFile(filepath.Join(root, "svc/a/CLAUDE.md"))
	if string(data) != "hand-written context" {
		t.Errorf("unmanaged file was modified")
	}
}
