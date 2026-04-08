# clanchor

[![CI](https://github.com/valon-loshaj/clanchor/actions/workflows/ci.yml/badge.svg)](https://github.com/valon-loshaj/clanchor/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/valon-loshaj/clanchor)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> **Alpha — the CLI interface, manifest format, and lock file format may still change.**

Package manager for `.claude` directories. Think `npm install` but for Claude Code configuration.

clanchor lets you version, distribute, and install complete `.claude` directory packages — skills, agents, commands, and CLAUDE.md files — from a central Git registry into any project or your global `~/.claude/` directory.

## Why

As Claude Code adoption grows, teams build shared skills, agents, and context files. Without tooling, distributing these means ad-hoc install scripts, copy-pasting between repos, and no way to track versions or update cleanly. Even for a solo dev, keeping `.claude` content consistent across projects doesn't scale.

clanchor fixes this. You maintain your `.claude` packages in one place, version them with semver, and install them into any repo with a one-liner. A lock file tracks exactly what's installed so you can detect drift, update to new versions, and remove packages cleanly.

## How it works

1. You keep your `.claude` packages in a **registry repo** — a GitHub repo with a folder structure and git tags
2. You declare what packages you need in a **manifest** (`clanchor.json`) at your repo root
3. `clanchor install` fetches each package and merges its contents into your `.claude/` directory
4. A **lock file** (`clanchor-lock.json`) tracks installed packages with per-file hashes for drift detection

## Prerequisites

- [GitHub CLI](https://cli.github.com/) (`gh`) — installed and logged in (`gh auth login`)
- Read access to whatever registry repo you're pulling from (your own, your team's, or a public one)

## Installation

### Homebrew (macOS)

```bash
brew tap valon-loshaj/tap
brew install clanchor
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

### 1. Create a manifest

Add a `clanchor.json` at your repo root:

```json
{
  "version": 2,
  "registry": "your-username/claude-registry",
  "packages": [
    {
      "name": "mycontext/go-backend",
      "version": "1.0.0",
      "scope": "project"
    }
  ],
  "claude_md": [
    {
      "path": ".",
      "namespace": "mycontext/project-context",
      "version": "1.0.0"
    }
  ]
}
```

- **registry** — default GitHub registry repo for all entries (`org/repo` format)
- **packages** — `.claude` directory packages to install (skills, agents, commands)
- **claude_md** — standalone CLAUDE.md files to place at specific repo paths

### 2. Install

From anywhere inside your git repo:

```bash
clanchor install
```

clanchor reads the manifest, resolves each package and CLAUDE.md entry from the registry, writes the files, and creates a lock file at the repo root.

### 3. Commit

```bash
git add clanchor.json clanchor-lock.json .claude/
git commit -m "add managed .claude packages via clanchor"
```

## Manifest format

The manifest (`clanchor.json`) lives at the repo root and declares everything clanchor should manage:

```json
{
  "version": 2,
  "registry": "myorg/claude-registry",
  "packages": [
    {
      "name": "acme/go-backend",
      "version": "1.2.0",
      "scope": "project"
    },
    {
      "name": "acme/security-agent",
      "version": "0.3.0",
      "scope": "global"
    },
    {
      "name": "other-org/python-skills",
      "version": "2.0.0",
      "registry": "other-org/claude-packages"
    }
  ],
  "claude_md": [
    {
      "path": ".",
      "namespace": "acme/project-context",
      "version": "1.0.0"
    },
    {
      "path": "services/payments",
      "namespace": "acme/payments-context",
      "version": "0.5.0"
    }
  ]
}
```

### Package entries

| Field | Required | Description |
|---|---|---|
| `name` | yes | Package name in the registry (e.g. `acme/go-backend`) |
| `version` | yes | Semver version to pin to (`X.Y.Z`, no `v` prefix) |
| `registry` | no | Override the default registry for this entry |
| `scope` | no | `"project"` (default) installs to `./.claude/`, `"global"` installs to `~/.claude/` |

### CLAUDE.md entries

| Field | Required | Description |
|---|---|---|
| `path` | yes | Directory where the CLAUDE.md should be placed (`.` for repo root) |
| `namespace` | yes | Path to the CLAUDE.md in the registry |
| `version` | yes | Semver version to pin to |
| `registry` | no | Override the default registry for this entry |

## Commands

### `clanchor install`

Resolve and install all packages and CLAUDE.md entries declared in the manifest. New entries are picked up automatically. Already-locked entries are skipped to avoid unnecessary network calls.

### `clanchor install --update`

Reconcile drift between the manifest and lock file. Required when you change a version, switch registries, or remove an entry. This is deliberate — it prevents accidental overwrites of resolved content.

When a package is removed from the manifest, `--update` deletes the files clanchor placed (tracked in the lock file). Only files clanchor owns are removed.

### `clanchor status`

Show installed packages and CLAUDE.md files, and report any drift between the manifest and lock file.

### `clanchor remove <package>`

Remove a package from the manifest, delete its files from disk, and update the lock file in one step.

## Setting up a registry

A registry is a GitHub repo where you publish `.claude` packages and CLAUDE.md files. Versions are git tags.

### 1. Create the registry repo

```bash
mkdir claude-registry && cd claude-registry
git init && gh repo create your-username/claude-registry --public --source=.
```

### 2. Add packages

A package is a directory containing a `.claude/` tree and optionally a `CLAUDE.md`:

```
acme/
  go-backend/
    .claude/
      skills/
        go-review/
          SKILL.md
          reference.md
      agents/
        reviewer.md
      commands/
        lint.md
    CLAUDE.md
  react-frontend/
    .claude/
      skills/
        component-gen/
          SKILL.md
    CLAUDE.md
```

You can also publish standalone CLAUDE.md files without a `.claude/` directory — useful for context files that don't come with skills or agents.

### 3. Tag and push

Tag format is `{namespace}@{version}`:

```bash
git add . && git commit -m "initial packages"
git tag "acme/go-backend@1.0.0"
git tag "acme/react-frontend@1.0.0"
git push origin main --tags
```

### 4. Use it

Point your manifest at the registry and run `clanchor install`. The registry can be public, private, or internal to your org — if `gh` can read it, clanchor can resolve from it.

To publish a new version, update the package contents, commit, tag with the new version, and push:

```bash
git tag "acme/go-backend@1.1.0"
git push origin --tags
```

## Conflict detection

If two packages provide the same file path (e.g. both include `.claude/skills/go-review/SKILL.md`), clanchor refuses to install and reports the collision before writing anything:

```
package conflicts detected:
  .claude/skills/go-review/SKILL.md: provided by acme/go-backend, acme/shared-skills
```

Resolve by removing one of the conflicting packages from your manifest.

## What the output looks like

```
INFO repo root path=/path/to/your/repo
INFO manifest loaded packages=2 claude_md=1
INFO installed package name=acme/go-backend version=1.0.0 files=3
INFO installed package name=acme/security-agent version=0.3.0 files=1
INFO install complete packages_installed=2 package_files_written=4 claude_md_written=1 failed=0
```

If something can't be resolved, clanchor tells you why and keeps going:

```
WARN resolve failed package=acme/missing version=1.0.0 error="tag not found..."
INFO install complete packages_installed=1 package_files_written=3 claude_md_written=1 failed=1
```

## Using clanchor with AI agents

clanchor is designed to work with AI coding agents like Claude Code. The [AGENTS.md](AGENTS.md) file contains a machine-readable reference that agents can use to operate clanchor correctly.

If you're setting up a project where Claude Code should be able to run clanchor, add this to your project's CLAUDE.md:

```
This project uses clanchor for .claude package management.
Run `clanchor install` to install packages and context files.
Run `clanchor install --update` after version changes in clanchor.json.
```

## Architecture

For contributors: see [CLAUDE.md](CLAUDE.md) for the full architecture overview and conventions. The install pipeline is an 8-step process (find root, read manifest, read lock, detect drift, resolve, detect conflicts, write files, update lock), and the primary extension point is the `Resolver` interface in `internal/resolver`.

## Status

Alpha — under active development. The install pipeline, status, and remove commands work end-to-end. The manifest format, lock file format, and CLI interface may still change before v1.0.
