# clanchor

> **Under active development — not yet ready for production use.**

Version and distribute your CLAUDE.md files. Think `npm install` but for Claude context.

Whether you're a solo dev trying to keep your CLAUDE.md files consistent across projects, or a team standardizing context across a monorepo, clanchor gives you a single source of truth for your Claude context files, versioned with semver and resolved from a Git registry.

## Why

CLAUDE.md files are powerful but they drift. You tweak one in a project, forget to update it elsewhere, and suddenly Claude is working with stale or inconsistent context. Copy-pasting between repos doesn't scale even if "scale" just means your own 5 side projects.

clanchor fixes this. You maintain your CLAUDE.md files in one place, version them, and pull them into any repo with a one-liner.

## How it works

1. You keep your CLAUDE.md files in a **registry repo** — just a GitHub repo with a folder structure and git tags
2. You drop a small **marker file** (`clanchor.json`) in any directory that should receive a CLAUDE.md
3. `clanchor install` fetches the right version of each file and writes it into place
4. A **lock file** (`clanchor-lock.json`) tracks what's installed so you can detect drift

## Prerequisites

- [GitHub CLI](https://cli.github.com/) (`gh`) — installed and logged in (`gh auth login`)
- Read access to whatever registry repo you're pulling from (your own, your team's, or a public one)

## Installation

Grab the latest binary from the [releases page](https://github.com/valon-loshaj/clanchor/releases) and drop it on your `PATH`:

```bash
# macOS (Apple Silicon)
curl -L https://github.com/valon-loshaj/clanchor/releases/latest/download/clanchor-darwin-arm64 -o /usr/local/bin/clanchor
chmod +x /usr/local/bin/clanchor

# macOS (Intel)
curl -L https://github.com/valon-loshaj/clanchor/releases/latest/download/clanchor-darwin-amd64 -o /usr/local/bin/clanchor
chmod +x /usr/local/bin/clanchor

# Linux (amd64)
curl -L https://github.com/valon-loshaj/clanchor/releases/latest/download/clanchor-linux-amd64 -o /usr/local/bin/clanchor
chmod +x /usr/local/bin/clanchor
```

Check that it works:

```bash
clanchor --help
```

## Quick start

### 1. Add a marker file

Drop a `clanchor.json` in any directory where you want a managed CLAUDE.md:

```json
{
  "namespace": "mycontext/go-backend",
  "version": "1.0.0",
  "registry": "your-username/claude-registry"
}
```

- **namespace** — path to the CLAUDE.md in your registry repo
- **version** — semver version to pin to (corresponds to a git tag)
- **registry** — the GitHub repo that holds your context files (`org/repo` or `username/repo`)

### 2. Install

From anywhere inside your git repo:

```bash
clanchor install
```

That's it. clanchor finds all your marker files, pulls the right CLAUDE.md for each one, writes them with a managed header, and creates a lock file at the repo root.

### 3. Commit

```bash
git add clanchor-lock.json
git add '**/CLAUDE.md'
git commit -m "add managed CLAUDE.md files via clanchor"
```

## Updating versions

Bump the `version` in a marker file, then run:

```bash
clanchor install --update
```

`--update` is required when you change a version or remove a marker. This is deliberate because it keeps you from accidentally overwriting your resolved context.

## Adding new markers

New marker files are picked up automatically on any `clanchor install` with no `--update` needed. Same goes for retrying after a failed resolve (e.g. you tagged the registry after your first attempt).

## Setting up a registry

A registry is just a GitHub repo. Organize your CLAUDE.md files in directories and tag them:

```
mycontext/
  go-backend/
    CLAUDE.md
  react-frontend/
    CLAUDE.md
  python-scripts/
    CLAUDE.md
```

Tag format is `{namespace}@{version}`:

```bash
git tag "mycontext/go-backend@1.0.0"
git tag "mycontext/react-frontend@1.0.0"
git push origin --tags
```

That's your registry. Point your marker files at it and you're good to go. It can be public, private, or internal to your org. If `gh` can read it, clanchor can resolve from it.

## What the output looks like

```
INFO repo root path=/path/to/your/repo
INFO discovered markers count=3
INFO newly resolved count=3
INFO   resolved path=services/api namespace=mycontext/go-backend version=1.0.0
INFO   resolved path=web namespace=mycontext/react-frontend version=1.0.0
INFO   resolved path=scripts namespace=mycontext/python-scripts version=1.0.0
INFO install complete resolved=3 unchanged=0 skipped=0 failed=0
```

If something can't be resolved, clanchor tells you why and keeps going:

```
WARN resolve failed namespace=mycontext/missing version=1.0.0 error="tag not found..."
INFO install complete resolved=2 unchanged=0 skipped=0 failed=1
```

## Status

Putzing...
