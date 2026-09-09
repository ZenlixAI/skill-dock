# Governance and visibility

## Responsibility

The server owns skill and release lifecycle state. Governance is independent of bundle storage and remains visible through catalog and audit APIs.

## Governed state

A skill can be active or retired. A release can be published, deprecated, or yanked, with an install policy such as `catalog_only`, `resolvable`, or `blocked`.

Governance operations include:

- retiring a skill;
- deprecating a version;
- yanking a version;
- recording a reason and optional replacement recommendation.

## Rules

- state changes are explicit;
- release history is not deleted to represent a lifecycle change;
- the artifact repository does not decide visibility or installability;
- the latest published or resolvable version is exposed by server resolution;
- governance mutations leave an audit record with actor, target, reason, and time.

## API surfaces

The server exposes skill-level and version-level governance mutation routes, plus audit-event reads. Read responses include status, version status, install policy, and provenance fields where applicable.

A deprecated or yanked release must not be silently selected as a replacement. If no installable release can be resolved, the server returns an explicit error.
