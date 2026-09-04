# Package and release model

## Identity

A skill is identified by its npm-compatible package name.

That package name is the public handle used by the registry and the CLI.

## Release model

The server stores release metadata, not just bundle bytes.

A release should answer:

- what package is this
- what version is this
- which bundle belongs to it
- is it visible
- is it installable
- what governance state does it have

## Bundle storage

Bundles live in an external npm-compatible artifact repository.

The server is authoritative for release records. The artifact repository is authoritative for bundle storage. A bundle in storage is not enough by itself to count as a published release.

## Core invariants

- package identity is stable
- release version is immutable once published
- release history should not be silently deleted
- visibility changes should be explicit
- governance changes should be auditable
- the latest published release should be resolvable without guessing

## Scope of this document

This document defines the identity and release model. Workflow details live in:

- [Publish and import workflows](07-publish-and-import-workflows.md)
- [Install resolution and refresh](08-install-resolution-and-refresh.md)
- [Governance and visibility](09-governance-and-visibility.md)

## Design goal

Keep the package model simple enough to reason about and strict enough to support governance.
