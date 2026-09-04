# Governance and visibility

## Purpose

Governance makes registry state explicit.

The server should be able to explain what is visible, what is installable, and why a release or skill changed state.

## What governance covers

- visibility changes
- lifecycle changes
- release history retention
- audit records
- latest published resolution

## Core ideas

- a skill release should not disappear silently
- a visibility change should be explicit
- a governance action should leave an audit trail
- the latest published release should be discoverable without scanning all history

## Operational meaning

Governance is the layer that decides whether a skill is:

- visible
- installable
- archived
- unpublished or yanked

## What not to do

- do not fold governance into bundle storage
- do not let the artifact repository decide visibility
- do not delete history to represent state changes unless the contract explicitly says so

## Scope of this document

This document is the governance companion to:

- [Package and release model](05-package-and-release-model.md)
- [Publish and import workflows](07-publish-and-import-workflows.md)
- [Install resolution and refresh](08-install-resolution-and-refresh.md)

## Design rule

Treat governance as part of the registry contract, not as an implementation detail.
