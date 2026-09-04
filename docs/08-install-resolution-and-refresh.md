# Install resolution and refresh

## Purpose

The server resolves what can be installed.

The CLI uses that resolution to install or refresh local skills.

## Supported flows

- `skills find`
- `skills add`
- `skills update`
- `skills sync`

## Registry installs

Registry installs are resolved against the server catalog.

The server provides the authoritative release record, and the CLI materializes that release locally for the selected agents or scope.

Registry access mode affects only the tarball fetch step: `proxy` uses the server-rewritten URL, while `direct` uses the upstream npm-compatible URL and may read npm credentials from the user's npm config.

## Source installs

Source installs come from local paths or git sources.

They are refreshed locally, not by asking the registry to re-derive the source tree.

## Refresh semantics

- `update` is for registry-installed skills
- `sync` is for source-installed skills
- `update` tracks published releases
- `sync` tracks local or git source state

## Core boundaries

- `find` reads registry truth
- `add` resolves either registry or source input
- `update` and `sync` must not blur their source models
- `list` and `remove` operate on local installed state, not catalog truth

## Design rule

Keep install resolution deterministic.

If the server cannot resolve a published release, the CLI should not guess a fallback release.
