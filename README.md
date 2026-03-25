# clanchor

> **This project is under active development and not yet ready for use.**

A CLI tool for managing CLAUDE.md files across large, multi-service codebases. clanchor introduces a versioned, composable system for distributing Claude context files — treating them as packageable artifacts that can be authored centrally, versioned, and resolved deterministically into any consuming repository.

## Problem

As Claude Code adoption grows inside engineering organizations, teams independently author CLAUDE.md files across their codebases. Without shared tooling this leads to duplicated effort, inconsistent context, version drift between services, and no clear ownership model.

## How it works

1. A **registry repo** holds canonical CLAUDE.md files, versioned via git tags
2. Developers place **marker files** (`clanchor.json`) in directories that should receive a CLAUDE.md
3. `clanchor install` crawls the repo, resolves files from the registry, and writes them into place
4. A **lock file** (`clanchor-lock.json`) tracks the resolved state for drift detection

## Status

Putzing...
