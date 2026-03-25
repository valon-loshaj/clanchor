package resolver

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// GitHubResolver fetches CLAUDE.md files from a GitHub registry repo using the gh CLI.
type GitHubResolver struct{}

// ghContentsResponse is the subset of the GitHub Contents API response we need.
type ghContentsResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

func (r *GitHubResolver) Resolve(namespace, version, registry string) ([]byte, string, error) {
	if err := checkGHAvailable(); err != nil {
		return nil, "", err
	}

	ref := namespace + "@" + version
	path := namespace + "/CLAUDE.md"

	endpoint := fmt.Sprintf("repos/%s/contents/%s?ref=%s", registry, path, ref)

	out, err := exec.Command("gh", "api", endpoint).Output()
	if err != nil {
		return nil, "", categorizeGHError(err, namespace, version, registry)
	}

	var resp ghContentsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, "", fmt.Errorf("failed to parse gh response for %s@%s: %w", namespace, version, err)
	}

	if resp.Encoding != "base64" {
		return nil, "", fmt.Errorf("unexpected encoding %q for %s@%s", resp.Encoding, namespace, version)
	}

	content, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(resp.Content, "\n", ""))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode content for %s@%s: %w", namespace, version, err)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(content))
	return content, hash, nil
}

func checkGHAvailable() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found: install it from https://cli.github.com and run 'gh auth login'")
	}
	return nil
}

func categorizeGHError(err error, namespace, version, registry string) error {
	msg := err.Error()
	if exitErr, ok := err.(*exec.ExitError); ok {
		msg = string(exitErr.Stderr)
	}
	lower := strings.ToLower(msg)

	if strings.Contains(lower, "not found") {
		return fmt.Errorf("tag %s@%s not found in registry %s (check that the tag exists and the namespace directory contains CLAUDE.md)", namespace, version, registry)
	}
	if strings.Contains(lower, "auth") || strings.Contains(lower, "401") || strings.Contains(lower, "403") {
		return fmt.Errorf("gh authentication failed for registry %s: run 'gh auth login' or check repo access", registry)
	}
	return fmt.Errorf("gh api error resolving %s@%s from %s: %s", namespace, version, registry, msg)
}
