package lockfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/valon-loshaj/clanchor/internal/model"
)

func TestReadWrite_RoundTrip(t *testing.T) {
	root := t.TempDir()
	lf := model.LockFile{
		Version: 1,
		Entries: []model.LockEntry{
			{Path: "services/b", Namespace: "acme/b", Version: "2.0.0", Registry: "org/repo", Hash: "abc"},
			{Path: "services/a", Namespace: "acme/a", Version: "1.0.0", Registry: "org/repo", Hash: "def"},
		},
	}

	if err := Write(root, lf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read(root)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	// Write sorts by path, so entries should be reordered.
	if got.Entries[0].Path != "services/a" {
		t.Errorf("first entry path = %q, want %q", got.Entries[0].Path, "services/a")
	}
	if got.Entries[1].Path != "services/b" {
		t.Errorf("second entry path = %q, want %q", got.Entries[1].Path, "services/b")
	}
}

func TestRead_FirstRun(t *testing.T) {
	root := t.TempDir()
	lf, err := Read(root)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if lf.Version != 1 {
		t.Errorf("version = %d, want 1", lf.Version)
	}
	if len(lf.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(lf.Entries))
	}
}

func TestRead_InvalidJSON(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, FileName), []byte(`{bad`), 0o644)
	_, err := Read(root)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDiff_NoChanges(t *testing.T) {
	markers := []model.DiscoveredMarker{
		{Dir: "svc/a", Marker: model.MarkerFile{Version: "1.0.0"}},
	}
	lf := model.LockFile{
		Entries: []model.LockEntry{
			{Path: "svc/a", Version: "1.0.0"},
		},
	}
	drifts := Diff(markers, lf)
	if len(drifts) != 0 {
		t.Errorf("got %d drifts, want 0", len(drifts))
	}
}

func TestDiff_Added(t *testing.T) {
	markers := []model.DiscoveredMarker{
		{Dir: "svc/new", Marker: model.MarkerFile{Version: "1.0.0"}},
	}
	lf := model.LockFile{}
	drifts := Diff(markers, lf)
	if len(drifts) != 1 || drifts[0].Type != DriftAdded {
		t.Errorf("expected 1 added drift, got %+v", drifts)
	}
}

func TestDiff_Removed(t *testing.T) {
	markers := []model.DiscoveredMarker{}
	lf := model.LockFile{
		Entries: []model.LockEntry{
			{Path: "svc/old", Version: "1.0.0"},
		},
	}
	drifts := Diff(markers, lf)
	if len(drifts) != 1 || drifts[0].Type != DriftRemoved {
		t.Errorf("expected 1 removed drift, got %+v", drifts)
	}
}

func TestDiff_VersionChanged(t *testing.T) {
	markers := []model.DiscoveredMarker{
		{Dir: "svc/a", Marker: model.MarkerFile{Version: "2.0.0"}},
	}
	lf := model.LockFile{
		Entries: []model.LockEntry{
			{Path: "svc/a", Version: "1.0.0"},
		},
	}
	drifts := Diff(markers, lf)
	if len(drifts) != 1 || drifts[0].Type != DriftVersionChanged {
		t.Errorf("expected 1 version_changed drift, got %+v", drifts)
	}
	if drifts[0].OldVersion != "1.0.0" || drifts[0].NewVersion != "2.0.0" {
		t.Errorf("drift versions = %q -> %q, want 1.0.0 -> 2.0.0", drifts[0].OldVersion, drifts[0].NewVersion)
	}
}
