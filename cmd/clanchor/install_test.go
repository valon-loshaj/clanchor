package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valon-loshaj/clanchor/internal/lockfile"
	"github.com/valon-loshaj/clanchor/internal/model"
)

// mockResolver returns predefined content for known namespaces.
type mockResolver struct {
	files map[string]string // namespace@version -> content
}

func (m *mockResolver) Resolve(namespace, version, registry string) ([]byte, string, error) {
	key := namespace + "@" + version
	content, ok := m.files[key]
	if !ok {
		return nil, "", fmt.Errorf("not found: %s", key)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
	return []byte(content), hash, nil
}

func writeMarker(t *testing.T, dir string, m model.MarkerFile) {
	t.Helper()
	os.MkdirAll(dir, 0o755)
	data := fmt.Sprintf(`{"namespace":%q,"version":%q,"registry":%q}`, m.Namespace, m.Version, m.Registry)
	if err := os.WriteFile(filepath.Join(dir, "clanchor.json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setupTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	// Initialize a .git directory so findRepoRoot works.
	os.MkdirAll(filepath.Join(root, ".git"), 0o755)
	return root
}

func TestInstall_FirstRun(t *testing.T) {
	root := setupTestRepo(t)
	writeMarker(t, filepath.Join(root, "services/payments"), model.MarkerFile{
		Namespace: "acme/payments",
		Version:   "1.0.0",
		Registry:  "org/registry",
	})
	writeMarker(t, filepath.Join(root, "services/auth"), model.MarkerFile{
		Namespace: "acme/auth",
		Version:   "2.0.0",
		Registry:  "org/registry",
	})

	mock := &mockResolver{files: map[string]string{
		"acme/payments@1.0.0": "# Payments context\n",
		"acme/auth@2.0.0":     "# Auth context\n",
	}}

	// Override working directory.
	origDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(origDir)

	if err := runInstall(false, mock); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Verify CLAUDE.md files were written.
	for _, dir := range []string{"services/payments", "services/auth"} {
		data, err := os.ReadFile(filepath.Join(root, dir, "CLAUDE.md"))
		if err != nil {
			t.Fatalf("reading %s/CLAUDE.md: %v", dir, err)
		}
		if !strings.Contains(string(data), "managed by clanchor") {
			t.Errorf("%s/CLAUDE.md missing managed header", dir)
		}
	}

	// Verify lock file was written.
	lf, err := lockfile.Read(root)
	if err != nil {
		t.Fatalf("reading lock: %v", err)
	}
	if len(lf.Entries) != 2 {
		t.Fatalf("lock entries = %d, want 2", len(lf.Entries))
	}
}

func TestInstall_NoDrift(t *testing.T) {
	root := setupTestRepo(t)
	writeMarker(t, filepath.Join(root, "services/payments"), model.MarkerFile{
		Namespace: "acme/payments",
		Version:   "1.0.0",
		Registry:  "org/registry",
	})

	mock := &mockResolver{files: map[string]string{
		"acme/payments@1.0.0": "# Payments context\n",
	}}

	origDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(origDir)

	// First install.
	if err := runInstall(false, mock); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Second install — should detect no drift.
	if err := runInstall(false, mock); err != nil {
		t.Fatalf("second install: %v", err)
	}
}

func TestInstall_DriftDetected(t *testing.T) {
	root := setupTestRepo(t)
	writeMarker(t, filepath.Join(root, "services/payments"), model.MarkerFile{
		Namespace: "acme/payments",
		Version:   "1.0.0",
		Registry:  "org/registry",
	})

	mock := &mockResolver{files: map[string]string{
		"acme/payments@1.0.0": "# Payments context\n",
		"acme/payments@2.0.0": "# Payments v2\n",
	}}

	origDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(origDir)

	// First install.
	if err := runInstall(false, mock); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Bump the marker version.
	writeMarker(t, filepath.Join(root, "services/payments"), model.MarkerFile{
		Namespace: "acme/payments",
		Version:   "2.0.0",
		Registry:  "org/registry",
	})

	// Second install without --update — should fail with drift.
	err := runInstall(false, mock)
	if err == nil {
		t.Fatal("expected drift error")
	}
	if !strings.Contains(err.Error(), "drift detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstall_PartialFailureThenRetry(t *testing.T) {
	root := setupTestRepo(t)
	writeMarker(t, filepath.Join(root, "services/payments"), model.MarkerFile{
		Namespace: "acme/payments",
		Version:   "1.0.0",
		Registry:  "org/registry",
	})
	writeMarker(t, filepath.Join(root, "services/auth"), model.MarkerFile{
		Namespace: "acme/auth",
		Version:   "1.0.0",
		Registry:  "org/registry",
	})

	// Only payments exists in the registry — auth will fail.
	mock := &mockResolver{files: map[string]string{
		"acme/payments@1.0.0": "# Payments context\n",
	}}

	origDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(origDir)

	// First install — payments succeeds, auth fails.
	if err := runInstall(false, mock); err != nil {
		t.Fatalf("first install: %v", err)
	}

	lf, _ := lockfile.Read(root)
	if len(lf.Entries) != 1 {
		t.Fatalf("lock entries = %d, want 1 (only payments)", len(lf.Entries))
	}

	// "Fix" the registry — auth is now available.
	mock.files["acme/auth@1.0.0"] = "# Auth context\n"

	// Second install (no --update) — should pick up auth without requiring --update.
	if err := runInstall(false, mock); err != nil {
		t.Fatalf("retry install: %v", err)
	}

	lf, _ = lockfile.Read(root)
	if len(lf.Entries) != 2 {
		t.Fatalf("lock entries = %d, want 2", len(lf.Entries))
	}

	// Verify both files exist.
	for _, dir := range []string{"services/payments", "services/auth"} {
		if _, err := os.Stat(filepath.Join(root, dir, "CLAUDE.md")); err != nil {
			t.Errorf("%s/CLAUDE.md missing after retry", dir)
		}
	}
}

func TestInstall_NewMarkerAddedAfterFirstRun(t *testing.T) {
	root := setupTestRepo(t)
	writeMarker(t, filepath.Join(root, "services/payments"), model.MarkerFile{
		Namespace: "acme/payments",
		Version:   "1.0.0",
		Registry:  "org/registry",
	})

	mock := &mockResolver{files: map[string]string{
		"acme/payments@1.0.0": "# Payments context\n",
		"acme/auth@1.0.0":     "# Auth context\n",
	}}

	origDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(origDir)

	// First install — only payments.
	if err := runInstall(false, mock); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Add a new marker — should not require --update.
	writeMarker(t, filepath.Join(root, "services/auth"), model.MarkerFile{
		Namespace: "acme/auth",
		Version:   "1.0.0",
		Registry:  "org/registry",
	})

	if err := runInstall(false, mock); err != nil {
		t.Fatalf("second install with new marker: %v", err)
	}

	lf, _ := lockfile.Read(root)
	if len(lf.Entries) != 2 {
		t.Fatalf("lock entries = %d, want 2", len(lf.Entries))
	}

	// Verify payments was NOT re-resolved (content unchanged).
	data, _ := os.ReadFile(filepath.Join(root, "services/payments/CLAUDE.md"))
	if !strings.Contains(string(data), "Payments context") {
		t.Errorf("payments file was unexpectedly modified")
	}

	// Verify auth was resolved.
	data, _ = os.ReadFile(filepath.Join(root, "services/auth/CLAUDE.md"))
	if !strings.Contains(string(data), "Auth context") {
		t.Errorf("auth file not written")
	}
}

func TestInstall_UpdateReconcilesDrift(t *testing.T) {
	root := setupTestRepo(t)
	writeMarker(t, filepath.Join(root, "services/payments"), model.MarkerFile{
		Namespace: "acme/payments",
		Version:   "1.0.0",
		Registry:  "org/registry",
	})

	mock := &mockResolver{files: map[string]string{
		"acme/payments@1.0.0": "# Payments context\n",
		"acme/payments@2.0.0": "# Payments v2\n",
	}}

	origDir, _ := os.Getwd()
	os.Chdir(root)
	defer os.Chdir(origDir)

	// First install.
	if err := runInstall(false, mock); err != nil {
		t.Fatalf("first install: %v", err)
	}

	// Bump version.
	writeMarker(t, filepath.Join(root, "services/payments"), model.MarkerFile{
		Namespace: "acme/payments",
		Version:   "2.0.0",
		Registry:  "org/registry",
	})

	// Install with --update.
	if err := runInstall(true, mock); err != nil {
		t.Fatalf("update install: %v", err)
	}

	// Verify the lock file reflects the new version.
	lf, err := lockfile.Read(root)
	if err != nil {
		t.Fatalf("reading lock: %v", err)
	}
	if lf.Entries[0].Version != "2.0.0" {
		t.Errorf("lock version = %q, want %q", lf.Entries[0].Version, "2.0.0")
	}

	// Verify file content was updated.
	data, _ := os.ReadFile(filepath.Join(root, "services/payments/CLAUDE.md"))
	if !strings.Contains(string(data), "Payments v2") {
		t.Errorf("CLAUDE.md not updated to v2 content")
	}
}
