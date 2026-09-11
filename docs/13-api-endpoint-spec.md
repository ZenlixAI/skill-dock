# API endpoint reference

Catalog API routes are served under `/api/v1`. Catalog and audit reads require a reader, publisher, or operator principal. Packaging review requires an authoring, publisher, or operator principal; publishing, release records, and governance changes require a publisher or operator principal.

## System routes

| Route | Purpose |
| --- | --- |
| `GET /healthz` | Liveness status |
| `GET /readyz` | Dependency readiness status |
| `GET /metrics` | Prometheus metrics |

## Catalog routes

| Route | Purpose |
| --- | --- |
| `GET /api/v1/skills` | List or search skills |
| `GET /api/v1/skills/search` | Alias of the skills list/search route |
| `GET /api/v1/skills/curated` | List curated skills |
| `GET /api/v1/skills/:package` | Get skill detail |
| `GET /api/v1/skills/:package/versions` | List versions |
| `GET /api/v1/skills/:package/versions/:version` | Get version detail |
| `GET /api/v1/skills/:package/install-target` | Resolve a published install target |
| `GET /api/v1/audit-events` | List audit events |
| `GET /api/v1/governance-changes` | List governance changes |

`GET /api/v1/skills` accepts `q`, `package_name`, `source_type`, `include_retired`, `include_external`, `source`, `registry`, `scope`, `origin`, `sort`, `pageSize`, and `cursor`.

- `source_type` accepts `native`, `imported`, `external`, and `platform_created`.
- `include_external` enables the configured external adapter.
- The current adapter recognizes `source=catalog`, `source=skills.sh`, and `source=all`.
- `sort` accepts `relevance`, `updated_at_desc`, and `package_name_asc`.

List responses contain `items` and `pageInfo`. Catalog records may include identity, registry, lifecycle, and provenance fields such as `sourceKind`, `sourceLocator`, origin URLs, `importMode`, `hosted`, `mirrored`, and `installable`.

`GET /api/v1/skills/:package/versions` accepts `include_deprecated`, `include_yanked`, `pageSize`, and `cursor`. `GET /api/v1/skills/:package/install-target` accepts an optional exact `version`.

## Packaging routes

| Route | Purpose |
| --- | --- |
| `POST /api/v1/skill-packaging/import-review` | Validate and package source input for review |
| `POST /api/v1/skill-packaging/publish` | Package source, publish bundle bytes, and create a release |

The server accepts `source.type` values `upload_archive` with `archiveBase64` and `platform_authoring` with structured `files`. The CLI's local, git, and npm import flows submit `upload_archive` after collecting the source.

Requests provide `target.packageName`, `target.version`, display metadata, and optional `options.publishRegistry` and `options.strict`. A publish request may include a review summary containing its manifest digest and component list. Responses include package identity, manifest digest, components, warnings, errors, and import report; publish responses also include `publishedToRegistry`, `created`, `idempotentReplay`, `versionStatus`, and `installPolicy`.

## Release and governance routes

| Route | Purpose |
| --- | --- |
| `PUT /api/v1/skills/:package/versions/:version/release-record` | Create a server-owned release record |
| `POST /api/v1/skills/:package/governance-changes` | Apply a skill lifecycle change |
| `POST /api/v1/skills/:package/versions/:version/governance-changes` | Apply a version lifecycle change |

Release record input contains the publication registry, manifest digest, and components. Governance requests record explicit lifecycle changes and their reasons.
