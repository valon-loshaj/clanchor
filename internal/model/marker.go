package model

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// MarkerFile represents the contents of a clanchor.json file.
type MarkerFile struct {
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
	Registry  string `json:"registry"`
}

// DiscoveredMarker pairs a parsed marker with its location in the consuming repo.
type DiscoveredMarker struct {
	Dir    string     // relative directory path from repo root
	Marker MarkerFile
}

var (
	semverRe  = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	orgRepoRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$`)
)

func ParseMarkerFile(data []byte) (MarkerFile, error) {
	var m MarkerFile
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("invalid JSON: %w", err)
	}
	return m, m.Validate()
}

func (m MarkerFile) Validate() error {
	if m.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if m.Version == "" {
		return fmt.Errorf("version is required")
	}
	if !semverRe.MatchString(m.Version) {
		return fmt.Errorf("version %q is not valid semver (expected X.Y.Z)", m.Version)
	}
	if m.Registry == "" {
		return fmt.Errorf("registry is required")
	}
	if !orgRepoRe.MatchString(m.Registry) {
		return fmt.Errorf("registry %q is not valid org/repo format", m.Registry)
	}
	return nil
}
