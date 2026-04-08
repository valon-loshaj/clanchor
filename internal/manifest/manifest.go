package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/valon-loshaj/clanchor/internal/model"
)

const FileName = "clanchor.json"

// Read loads and validates the v2 manifest from the repo root.
// Returns an error if the file is missing or invalid.
func Read(repoRoot string) (model.Manifest, error) {
	path := filepath.Join(repoRoot, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.Manifest{}, fmt.Errorf("no %s found in %s", FileName, repoRoot)
		}
		return model.Manifest{}, fmt.Errorf("reading %s: %w", FileName, err)
	}
	m, err := model.ParseManifest(data)
	if err != nil {
		return model.Manifest{}, fmt.Errorf("invalid %s: %w", FileName, err)
	}
	return m, nil
}
