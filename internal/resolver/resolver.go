package resolver

// Resolver fetches CLAUDE.md content from a registry.
type Resolver interface {
	Resolve(namespace, version, registry string) (content []byte, hash string, err error)
}
