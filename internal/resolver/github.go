package resolver

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/valon-loshaj/clanchor/internal/model"
)

// GitHubResolver fetches CLAUDE.md files from a GitHub registry repo using the gh CLI.
type GitHubResolver struct{}

// ghContentsResponse is the subset of the GitHub Contents API response we need.
type ghContentsResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

func (r *GitHubResolver) ResolveFile(namespace, version, registry string) ([]byte, string, error) {
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

// ghTreeResponse is the subset of the GitHub Git Trees API response we need.
type ghTreeResponse struct {
	Tree []ghTreeEntry `json:"tree"`
}

type ghTreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"` // "blob" or "tree"
	SHA  string `json:"sha"`
}

func (r *GitHubResolver) ResolvePackage(name, version, registry string) ([]model.ResolvedFile, error) {
	if err := checkGHAvailable(); err != nil {
		return nil, err
	}

	ref := name + "@" + version
	claudeDir := name + "/.claude"

	// First, get the commit SHA for the tag so we can use the Trees API.
	refEndpoint := fmt.Sprintf("repos/%s/git/ref/tags/%s", registry, ref)
	refOut, err := exec.Command("gh", "api", refEndpoint).Output()
	if err != nil {
		return nil, categorizeGHError(err, name, version, registry)
	}

	var refResp struct {
		Object struct {
			SHA  string `json:"sha"`
			Type string `json:"type"`
		} `json:"object"`
	}
	if err := json.Unmarshal(refOut, &refResp); err != nil {
		return nil, fmt.Errorf("failed to parse ref response for %s@%s: %w", name, version, err)
	}

	// If the tag points to a tag object (annotated tag), dereference to the commit.
	commitSHA := refResp.Object.SHA
	if refResp.Object.Type == "tag" {
		tagEndpoint := fmt.Sprintf("repos/%s/git/tags/%s", registry, commitSHA)
		tagOut, err := exec.Command("gh", "api", tagEndpoint).Output()
		if err != nil {
			return nil, fmt.Errorf("failed to dereference annotated tag %s@%s: %w", name, version, err)
		}
		var tagResp struct {
			Object struct {
				SHA string `json:"sha"`
			} `json:"object"`
		}
		if err := json.Unmarshal(tagOut, &tagResp); err != nil {
			return nil, fmt.Errorf("failed to parse tag response for %s@%s: %w", name, version, err)
		}
		commitSHA = tagResp.Object.SHA
	}

	// Get the full tree recursively.
	treeEndpoint := fmt.Sprintf("repos/%s/git/trees/%s?recursive=1", registry, commitSHA)
	treeOut, err := exec.Command("gh", "api", treeEndpoint).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tree for %s@%s: %w", name, version, err)
	}

	var treeResp ghTreeResponse
	if err := json.Unmarshal(treeOut, &treeResp); err != nil {
		return nil, fmt.Errorf("failed to parse tree response for %s@%s: %w", name, version, err)
	}

	// Filter to blobs under {name}/.claude/
	prefix := claudeDir + "/"
	var blobs []ghTreeEntry
	for _, entry := range treeResp.Tree {
		if entry.Type == "blob" && strings.HasPrefix(entry.Path, prefix) {
			blobs = append(blobs, entry)
		}
	}

	if len(blobs) == 0 {
		return nil, fmt.Errorf("no .claude/ directory found in package %s@%s", name, version)
	}

	// Fetch each blob's content.
	var resolved []model.ResolvedFile
	for _, blob := range blobs {
		blobEndpoint := fmt.Sprintf("repos/%s/git/blobs/%s", registry, blob.SHA)
		blobOut, err := exec.Command("gh", "api", blobEndpoint).Output()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch blob %s in %s@%s: %w", blob.Path, name, version, err)
		}

		var blobResp ghContentsResponse
		if err := json.Unmarshal(blobOut, &blobResp); err != nil {
			return nil, fmt.Errorf("failed to parse blob response for %s: %w", blob.Path, err)
		}

		content, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(blobResp.Content, "\n", ""))
		if err != nil {
			return nil, fmt.Errorf("failed to decode blob %s: %w", blob.Path, err)
		}

		// Strip the package namespace prefix to get the relative path (e.g., ".claude/skills/foo/SKILL.md")
		relativePath := strings.TrimPrefix(blob.Path, name+"/")

		hash := fmt.Sprintf("sha256:%x", sha256.Sum256(content))
		resolved = append(resolved, model.ResolvedFile{
			RelativePath: relativePath,
			Content:      content,
			Hash:         hash,
		})
	}

	return resolved, nil
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
