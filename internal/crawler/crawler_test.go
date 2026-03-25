package crawler

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

var validMarker = `{"namespace":"acme/svc","version":"1.0.0","registry":"org/repo"}`

func TestCrawl_DiscoversMarkers(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "services/payments", markerFileName), validMarker)
	writeFile(t, filepath.Join(root, "services/auth", markerFileName), validMarker)

	result, err := Crawl(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Markers) != 2 {
		t.Fatalf("got %d markers, want 2", len(result.Markers))
	}
	if len(result.Errors) != 0 {
		t.Fatalf("got %d errors, want 0", len(result.Errors))
	}
}

func TestCrawl_SkipsHiddenAndNodeModules(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".git", markerFileName), validMarker)
	writeFile(t, filepath.Join(root, ".github", markerFileName), validMarker)
	writeFile(t, filepath.Join(root, "node_modules/pkg", markerFileName), validMarker)
	writeFile(t, filepath.Join(root, "valid", markerFileName), validMarker)

	result, err := Crawl(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Markers) != 1 {
		t.Errorf("got %d markers, want 1", len(result.Markers))
	}
	if result.Markers[0].Dir != "valid" {
		t.Errorf("dir = %q, want %q", result.Markers[0].Dir, "valid")
	}
}

func TestCrawl_CollectsParseErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "good", markerFileName), validMarker)
	writeFile(t, filepath.Join(root, "bad-json", markerFileName), `{not json}`)
	writeFile(t, filepath.Join(root, "missing-fields", markerFileName), `{"namespace":"x"}`)

	result, err := Crawl(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Markers) != 1 {
		t.Errorf("got %d markers, want 1", len(result.Markers))
	}
	if len(result.Errors) != 2 {
		t.Errorf("got %d errors, want 2", len(result.Errors))
	}
}

func TestCrawl_RootMarker(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, markerFileName), validMarker)

	result, err := Crawl(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Markers) != 1 {
		t.Fatalf("got %d markers, want 1", len(result.Markers))
	}
	if result.Markers[0].Dir != "." {
		t.Errorf("dir = %q, want %q", result.Markers[0].Dir, ".")
	}
}
