package conflict

import (
	"fmt"
	"strings"

	"github.com/valon-loshaj/clanchor/internal/model"
)

// Conflict records that two or more packages write to the same relative path.
type Conflict struct {
	Path     string
	Packages []string
}

func (c Conflict) String() string {
	return fmt.Sprintf("%s: provided by %s", c.Path, strings.Join(c.Packages, ", "))
}

// Detect checks resolved packages for file path collisions.
// Returns nil if there are no conflicts.
func Detect(packages []model.ResolvedPackage) []Conflict {
	owners := make(map[string][]string) // relative path -> package names

	for _, pkg := range packages {
		for _, f := range pkg.Files {
			owners[f.RelativePath] = append(owners[f.RelativePath], pkg.Name)
		}
	}

	var conflicts []Conflict
	for path, pkgs := range owners {
		if len(pkgs) > 1 {
			conflicts = append(conflicts, Conflict{Path: path, Packages: pkgs})
		}
	}
	return conflicts
}
