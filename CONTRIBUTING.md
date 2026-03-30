# Contributing to clanchor

## Building from source

```bash
go build -o bin/clanchor ./cmd/clanchor
```

Or use the Makefile:

```bash
make build
```

## Running tests

```bash
go test ./...
```

## Manual testing

1. Create a registry repo on GitHub with a `CLAUDE.md` file in a namespace directory (e.g., `mycontext/go-backend/CLAUDE.md`)
2. Tag it: `git tag "mycontext/go-backend@1.0.0" && git push origin --tags`
3. In a test repo, add a `clanchor.json` marker pointing at your registry
4. Run `clanchor install`

## Submitting changes

- One logical change per PR
- Include tests for new behavior
- No unrelated changes — keep the diff focused
- All CI checks must pass

## Architecture

The install pipeline is a linear 7-step process. See [CLAUDE.md](CLAUDE.md) for the full architecture overview.

The primary extension point for contributors is the `Resolver` interface in `internal/resolver` — this is how new registry backends (Nexus, Artifactory, etc.) get added.
