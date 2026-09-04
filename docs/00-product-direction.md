# Product direction and scope

Skill Registry is a self-hosted control plane for reusable agent skills.

The product takes the same broad shape as SkillHub, but keeps the storage boundary lighter:

- the **server** is authoritative for search, metadata, release records, visibility, and governance
- the **CLI** is the main user interface for install and publish workflows
- the **artifact repository** stays external and replaceable
- the **database** stores catalog state, releases, and audit data

## Why this product exists

Agent skills are easy to share as folders and hard to operate as a reliable ecosystem. The product closes that gap by providing:

- discovery and search
- reproducible installs
- publishing and import workflows
- lifecycle governance
- auditability
- private repository integration

## Non-goals

This document set does not try to define:

- UI behavior
- a new artifact storage backend
- a social marketplace
- per-agent runtime policy
- execution tracing for skill consumers

## Product principles

- Keep the server authoritative.
- Keep bundles external.
- Keep the CLI aligned with server contracts.
- Make governance visible.
- Prefer replaceable integrations over hard dependencies.
- Only add new concepts when they improve policy, visibility, or review.

## Core contract

A skill is identified by its package name and a versioned release.

- the server decides what is published and installable
- the artifact repository stores the bundle
- the CLI resolves the server contract and performs local install or publish actions

## Install contract

The server may return install targets in one of two access modes:

- `proxy`: the caller should fetch the tarball through the server's npm proxy mount
- `direct`: the caller should fetch the tarball from the upstream npm-compatible registry directly

This is a transport detail of install resolution, not a new catalog identity model. The same package can resolve to either mode depending on deployment policy.

## Current boundaries

Supported workflows today:

- `find`
- `add`
- `list`
- `remove`
- `update`
- `sync`
- `init`
- `inspect`
- `publish`
- `import-local`
- `import-git`

These are the workflows the rest of the docs describe and the CLI help output should preserve.
