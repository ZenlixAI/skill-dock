# AGENTS.md

This file provides guidance to AI coding agents working on the `skills` CLI codebase.

## Project Overview

`skills` is the CLI for the open agent skills ecosystem.

## Commands

| Command                       | Description                                         |
| ----------------------------- | --------------------------------------------------- |
| `skills`                      | Show banner with available commands                 |
| `skills add <pkg>`            | Install registry skills by `package_name`           |
| `skills find [query]`         | Search for skills interactively or by keyword       |
| `skills list`                 | List installed skills (alias: `ls`)                 |
| `skills remove [skills]`      | Remove installed skills (alias: `rm`)               |
| `skills update [skills...]`   | Update skills to latest versions                    |
| `skills init [name]`          | Create a new SKILL.md template                      |
| `skills inspect [path]`       | Validate a local skill directory                    |
| `skills publish [path]`       | Validate, pack, and publish a local skill directory |

Aliases: `skills a` routes to `add`. `skills ls` works for `list`. `skills rm` works for `remove`.

## Architecture

```
src/
├── cli.ts           # Main entry point, command routing, init/update
├── cli.test.ts      # CLI tests
├── add.ts           # Core add command logic
├── add-prompt.test.ts # Add prompt behavior tests
├── add.test.ts      # Add command tests
├── constants.ts      # Shared constants
├── find.ts           # Find command
├── list.ts          # List installed skills command
├── list.test.ts     # List command tests
├── remove.ts         # Remove command implementation
├── remove.test.ts    # Remove command tests
├── agents.ts        # Agent definitions and detection
├── installer.ts     # Skill installation logic (symlink/copy)
├── installed-view.ts # Installed-skill read model for list/remove/update
├── skills.ts        # Skill discovery and parsing
├── skill-lock.ts    # Global lock file management (~/.agents/.skill-lock.json)
├── local-lock.ts    # Local lock file management (skills-lock.json, checked in)
├── source-parser.ts # Parse git URLs, GitHub shorthand, local paths
├── git.ts           # Git clone operations
├── telemetry.ts     # Anonymous usage tracking
├── types.ts         # TypeScript types
├── mintlify.ts      # Mintlify skill fetching (legacy)
├── plugin-manifest.ts # Plugin manifest discovery support
├── prompts/         # Interactive prompt helpers
│   └── search-multiselect.ts
├── providers/       # Remote skill providers (GitHub, HuggingFace, Mintlify)
│   ├── index.ts
│   ├── registry.ts
│   ├── types.ts
│   ├── huggingface.ts
│   ├── mintlify.ts
│   └── wellknown.ts
├── init.test.ts     # Init command tests
└── test-utils.ts    # Test utilities

tests/
├── cross-platform-paths.test.ts # Path normalization across platforms
├── full-depth-discovery.test.ts # --full-depth skill discovery tests
├── openclaw-paths.test.ts       # OpenClaw-specific path tests
├── plugin-manifest-discovery.test.ts # Plugin manifest skill discovery
├── sanitize-name.test.ts     # Tests for sanitizeName (path traversal prevention)
├── skill-matching.test.ts    # Tests for filterSkills (multi-word skill name matching)
├── source-parser.test.ts     # Tests for URL/path parsing
├── installer-symlink.test.ts # Tests for symlink installation
├── skill-path.test.ts        # Tests for skill path handling
├── wellknown-provider.test.ts # Tests for well-known provider
├── xdg-config-paths.test.ts   # XDG global path handling tests
└── dist.test.ts               # Tests for built distribution
```

## Update Checking System

### How `skills update` Works

1. Read the project/global lock file for installed skills
2. Filter to registry-tracked entries with a stable `packageName`
3. Resolve the latest install target from the Catalog
4. Reinstall only when the resolved version differs
5. Ignore non-registry historical lock entries during update resolution

### Lock File Compatibility

The global lock file format is v3 and the project lock file format is v1.

Current steady-state entries are registry-first and center on:

- `packageName`
- `version`
- `registry`
- integrity / digest fields
- folder/content hash fields

If older lock files or historical source-based fields are read, they are normalized into compatibility-only metadata during parsing. Update resolution ignores those non-registry historical entries.

## Key Integration Points

| Feature                    | Implementation                                                |
| -------------------------- | ------------------------------------------------------------- |
| `skills add`               | `src/add.ts` - full implementation                            |
| `skills update`            | `src/update.ts` - resolve latest registry target + reinstall  |

## Development

```bash
# Install dependencies
pnpm install

# Build
pnpm build

# Test locally
pnpm dev add @platform.skill/pdf-skill
pnpm dev update
pnpm dev init my-skill

# Run all tests
pnpm test

# Run specific test file(s)
pnpm test tests/sanitize-name.test.ts
pnpm test tests/skill-matching.test.ts tests/source-parser.test.ts

# Type check
pnpm type-check

# Format code
pnpm format

# Check formatting
pnpm format:check

# Validate and sync agent metadata/docs
pnpm run -C scripts validate-agents.ts
pnpm run -C scripts sync-agents.ts
```

## Code Style

This project uses Prettier for code formatting. **Always run `pnpm format` before committing changes** to ensure consistent formatting.

```bash
# Format all files
pnpm format

# Check formatting without fixing
pnpm format:check
```

CI will fail if code is not properly formatted.

## Publishing

```bash
# 1. Bump version in package.json
# 2. Build
pnpm build
# 3. Publish
npm publish
```

## Adding a New Agent

1. Add the agent definition to `src/agents.ts`
2. Run `pnpm run -C scripts validate-agents.ts` to validate
3. Run `pnpm run -C scripts sync-agents.ts` to update README.md and package keywords
