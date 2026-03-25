package model

import (
	"encoding/json"
	"testing"
)

func TestParseMarkerFile(t *testing.T) {
	valid := `{"namespace":"acme/payments-service","version":"1.4.0","registry":"acme-org/claude-context-registry"}`
	m, err := ParseMarkerFile([]byte(valid))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Namespace != "acme/payments-service" {
		t.Errorf("namespace = %q, want %q", m.Namespace, "acme/payments-service")
	}
	if m.Version != "1.4.0" {
		t.Errorf("version = %q, want %q", m.Version, "1.4.0")
	}
	if m.Registry != "acme-org/claude-context-registry" {
		t.Errorf("registry = %q, want %q", m.Registry, "acme-org/claude-context-registry")
	}
}

func TestParseMarkerFile_RoundTrip(t *testing.T) {
	original := MarkerFile{
		Namespace: "acme/auth-service",
		Version:   "2.0.0",
		Registry:  "acme-org/registry",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	parsed, err := ParseMarkerFile(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed != original {
		t.Errorf("round-trip mismatch: got %+v, want %+v", parsed, original)
	}
}

func TestParseMarkerFile_InvalidJSON(t *testing.T) {
	_, err := ParseMarkerFile([]byte(`{not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMarkerFile_Validate(t *testing.T) {
	tests := []struct {
		name    string
		marker  MarkerFile
		wantErr bool
	}{
		{"valid", MarkerFile{"acme/svc", "1.0.0", "org/repo"}, false},
		{"empty namespace", MarkerFile{"", "1.0.0", "org/repo"}, true},
		{"empty version", MarkerFile{"acme/svc", "", "org/repo"}, true},
		{"bad semver", MarkerFile{"acme/svc", "v1.0.0", "org/repo"}, true},
		{"bad semver letters", MarkerFile{"acme/svc", "abc", "org/repo"}, true},
		{"empty registry", MarkerFile{"acme/svc", "1.0.0", ""}, true},
		{"bad registry format", MarkerFile{"acme/svc", "1.0.0", "no-slash"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.marker.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
