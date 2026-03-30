# clanchor

[![CI](https://github.com/valon-loshaj/clanchor/actions/workflows/ci.yml/badge.svg)](https://github.com/valon-loshaj/clanchor/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/valon-loshaj/clanchor)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> **Alpha — the CLI interface, marker format, and lock file format may still change.**

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

### Homebrew (macOS)

```bash
brew install valon-loshaj/tap/clanchor
```

### Go install

```bash
go install github.com/valon-loshaj/clanchor/cmd/clanchor@latest
```

### Binary download

Grab the latest binary from the [releases page](https://github.com/valon-loshaj/clanchor/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/valon-loshaj/clanchor/releases/latest/download/clanchor-darwin-arm64.tar.gz | tar xz
sudo mv clanchor /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/valon-loshaj/clanchor/releases/latest/download/clanchor-darwin-amd64.tar.gz | tar xz
sudo mv clanchor /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/valon-loshaj/clanchor/releases/latest/download/clanchor-linux-amd64.tar.gz | tar xz
sudo mv clanchor /usr/local/bin/
```

Verify:

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

A registry is just a GitHub repo. Create one, organize your CLAUDE.md files in directories, and tag them.

### 1. Create the registry repo

```bash
mkdir claude-registry && cd claude-registry
git init && gh repo create your-username/claude-registry --public --source=.
```

### 2. Add context files

```
mycontext/
  go-backend/
    CLAUDE.md
  react-frontend/
    CLAUDE.md
  python-scripts/
    CLAUDE.md
```

Each `CLAUDE.md` is a standalone context file — whatever you want Claude to know when working in a directory of that type.

### 3. Tag and push

Tag format is `{namespace}@{version}`:

```bash
git add . && git commit -m "initial context files"
git tag "mycontext/go-backend@1.0.0"
git tag "mycontext/react-frontend@1.0.0"
git tag "mycontext/python-scripts@1.0.0"
git push origin main --tags
```

### 4. Use it

Point your marker files at the registry and run `clanchor install`. The registry can be public, private, or internal to your org — if `gh` can read it, clanchor can resolve from it.

To publish a new version, update the CLAUDE.md, commit, tag with the new version, and push:

```bash
git tag "mycontext/go-backend@1.1.0"
git push origin --tags
```

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

## Using clanchor with AI agents

clanchor is designed to work with AI coding agents like Claude Code. The [AGENTS.md](AGENTS.md) file contains a machine-readable reference that agents can use to operate clanchor correctly.

If you're setting up a project where Claude Code should be able to run clanchor, add this to your project's CLAUDE.md:

```
This project uses clanchor for CLAUDE.md management.
Run `clanchor install` to fetch context files.
Run `clanchor install --update` after version changes in clanchor.json files.
```

## Architecture

For contributors: see [CLAUDE.md](CLAUDE.md) for the full architecture overview and conventions. The install pipeline is a linear 7-step process, and the primary extension point is the `Resolver` interface in `internal/resolver`.

## Status

Alpha — under active development. The install pipeline works end-to-end, but the CLI interface, marker format, and lock file format may still change before v1.0.
