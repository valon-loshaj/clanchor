package crawler

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/valon-loshaj/clanchor/internal/model"
)

const markerFileName = "clanchor.json"

var skipDirs = map[string]bool{
	".git":         true,
	".github":      true,
	"node_modules": true,
}

// CrawlResult holds the successfully parsed markers and any errors encountered.
type CrawlResult struct {
	Markers []model.DiscoveredMarker
	Errors  []CrawlError
}

// CrawlError records a parse/validation failure for a specific marker file.
type CrawlError struct {
	Path string
	Err  error
}

func (e CrawlError) Error() string {
	return e.Path + ": " + e.Err.Error()
}

// Crawl walks root and discovers all clanchor.json marker files.
// It returns all valid markers and all errors without stopping on individual failures.
func Crawl(root string) (CrawlResult, error) {
	var result CrawlResult

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Name() != markerFileName {
			return nil
		}

		relDir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			result.Errors = append(result.Errors, CrawlError{Path: relDir, Err: err})
			return nil
		}

		marker, err := model.ParseMarkerFile(data)
		if err != nil {
			result.Errors = append(result.Errors, CrawlError{Path: relDir, Err: err})
			return nil
		}

		// Normalize path separators for consistency.
		relDir = strings.ReplaceAll(relDir, string(os.PathSeparator), "/")

		result.Markers = append(result.Markers, model.DiscoveredMarker{
			Dir:    relDir,
			Marker: marker,
		})
		return nil
	})

	return result, err
}
