# API endpoint reference

This reference describes the routes and payload fields implemented by the catalog server. It does not prescribe unimplemented external-source behavior.

## Read routes

| Route | Purpose |
| --- | --- |
| `GET /api/v1/skills` | List or search skills |
| `GET /api/v1/skills/search` | Alias of the skills list/search route |
| `GET /api/v1/skills/curated` | List curated skills |
| `GET /api/v1/skills/:package` | Get skill detail |
| `GET /api/v1/skills/:package/versions` | List versions |
| `GET /api/v1/skills/:package/versions/:version` | Get version detail |
| `GET /api/v1/skills/:package/install-target` | Resolve a published install target |

`GET /api/v1/skills` accepts `q`, `package_name`, `source_type`, `include_retired`, `include_external`, `source`, `registry`, `scope`, `origin`, `sort`, `pageSize`, and `cursor`.

- `source_type` accepts `native`, `imported`, `external`, or `platform_created`.
- `include_external` is boolean and enables the configured external adapter.
- `source=catalog`, `source=skills.sh`, and `source=all` are meaningful with the current adapter.
- `registry`, `scope`, and `origin` are request-scoped values forwarded to the adapter.
- `sort` accepts `relevance`, `updated_at_desc`, or `package_name_asc`.

List responses contain `items` and `pageInfo`. Skill, version, and install responses expose available identity, registry, lifecycle, and provenance fields, including `sourceKind`, `sourceLocator`, origins, `importMode`, `hosted`, `mirrored`, and `installable`.

External `skills.sh` results are descriptive and return `installable: false`; install-target resolution remains a catalog release operation.

## Packaging routes

| Route | Purpose |
| --- | --- |
| `POST /api/v1/skill-packaging/import-review` | Review and package source input |
| `POST /api/v1/skill-packaging/publish` | Package, publish bundle bytes, and create a release record |

Current `source.type` values are:

- `upload_archive` with `archiveBase64`;
- `platform_authoring` with structured `files`.

The CLI's local, git, and npm import paths currently submit `upload_archive` after source collection or extraction. The server rejects `git`, `local`, `npm`, `archive`, and source-mode fields as normalized client source types.

Requests include `target.packageName`, `target.version`, metadata containing display name and description, optional `options.publishRegistry`, and optional `options.strict`. Publish requests can include a review summary with manifest digest and components. Responses include package name, version, manifest digest, components, warnings, errors, import report, and for publish: `publishedToRegistry`, `created`, `idempotentReplay`, `versionStatus`, and `installPolicy`.

## Governance routes

| Route | Purpose |
| --- | --- |
| `PUT /api/v1/skills/:package/versions/:version/release-record` | Create server-owned release truth |
| `POST /api/v1/skills/:package/governance-changes` | Apply skill lifecycle change |
| `POST /api/v1/skills/:package/versions/:version/governance-changes` | Apply version lifecycle change |
| `GET /api/v1/governance-changes` | Read audit/governance records |

Release record input accepts publication registry, manifest digest, and components. It does not currently accept client-supplied source provenance fields.

## Compatibility

Existing fields and routes use camelCase JSON names. New response fields should be additive and must not change package identity, exact-version rules, server authority, or deterministic install resolution.
