package resolver

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
)

func TestGHContentsResponseParsing(t *testing.T) {
	original := []byte("# My CLAUDE.md\nSome context here.\n")
	encoded := base64.StdEncoding.EncodeToString(original)

	resp := ghContentsResponse{
		Content:  encoded,
		Encoding: "base64",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed ghContentsResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(parsed.Content)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if string(decoded) != string(original) {
		t.Errorf("decoded = %q, want %q", decoded, original)
	}

	expectedHash := fmt.Sprintf("%x", sha256.Sum256(original))
	actualHash := fmt.Sprintf("%x", sha256.Sum256(decoded))
	if actualHash != expectedHash {
		t.Errorf("hash = %q, want %q", actualHash, expectedHash)
	}
}

func TestCategorizeGHError_NotFound(t *testing.T) {
	err := categorizeGHError(fmt.Errorf("Not Found"), "acme/svc", "1.0.0", "org/repo")
	if err == nil {
		t.Fatal("expected error")
	}
	want := "tag acme/svc@1.0.0 not found"
	if got := err.Error(); !contains(got, want) {
		t.Errorf("error = %q, want it to contain %q", got, want)
	}
}

func TestCategorizeGHError_Auth(t *testing.T) {
	err := categorizeGHError(fmt.Errorf("401 Unauthorized"), "acme/svc", "1.0.0", "org/repo")
	if err == nil {
		t.Fatal("expected error")
	}
	want := "gh authentication failed"
	if got := err.Error(); !contains(got, want) {
		t.Errorf("error = %q, want it to contain %q", got, want)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
