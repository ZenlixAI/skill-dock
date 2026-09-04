# Publish and import workflows

## Purpose

Publishing turns a local skill source into a server-recognized release record.

The server is responsible for release truth. The artifact repository is responsible for storing the bundle.

## Supported flows

- `skills publish`
- `skills import-local`
- `skills import-git`

## Inputs

### `publish`

A local skill directory.

### `import-local`

A local source tree that may need package name or version injection.

### `import-git`

A git source plus optional skill selection.

## Common workflow

1. validate the source
2. normalize package metadata when needed
3. pack the skill bundle
4. store the bundle in the artifact repository
5. create or update the server release record
6. write audit data
7. expose the release through the catalog

## Workflow boundaries

- the CLI can collect and package source input
- the server decides whether the release is publishable
- the artifact repository stores the bundle bytes
- a bundle without a release record is not a published release
- registry access mode (`proxy` / `direct`) does not apply to publish or import; those flows always go through the server

## Import semantics

- `import-local` is for local source trees
- `import-git` is for external git sources
- both flows are publish flows, not refresh flows

## Output

A successful publish should produce a stable release record that the server can resolve later through the catalog.

## What not to do

- do not publish directly to the artifact repository from the CLI
- do not let source shape leak into the server contract
- do not treat import as a separate data model from publish
- do not create release history without audit data
