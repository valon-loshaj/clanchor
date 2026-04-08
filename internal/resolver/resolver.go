package resolver

import "github.com/valon-loshaj/clanchor/internal/model"

// Resolver fetches content from a registry.
type Resolver interface {
	// ResolveFile fetches a single CLAUDE.md file by namespace, version, and registry.
	ResolveFile(namespace, version, registry string) (content []byte, hash string, err error)

	// ResolvePackage fetches all files in a .claude directory package.
	ResolvePackage(name, version, registry string) ([]model.ResolvedFile, error)
}
