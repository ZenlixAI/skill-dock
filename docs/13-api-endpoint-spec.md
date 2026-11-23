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

## Identity and user routes

These routes require a Logto user access token or a configured service token, depending on the deployment mode.

| Route | Purpose |
| --- | --- |
| `GET /api/v1/me` | Current user, organizations, active tenant, roles, scopes, and feature flags. |
| `GET /api/v1/me/orgs` | Organizations known for the current user. |
| `GET /api/v1/me/favorites` | List the caller's favorites. |
| `POST /api/v1/me/favorites` | Upsert a favorite. |
| `DELETE /api/v1/me/favorites?refKey=...` | Remove a favorite. |
| `GET /api/v1/me/favorites/status?refKey=...` | Batch favorite status. |
| `POST /api/v1/me/favorites/merge` | Merge local favorites after sign-in. |
| `GET/POST /api/v1/tags` | List visible tags or create a tenant tag. |
| `PATCH/DELETE /api/v1/tags/:id` | Update or delete a tenant tag. |
| `PUT /api/v1/skills/:package/tags` | Replace a skill's tags. |
| `GET/POST /api/v1/orgs/:org/namespaces` | List or claim tenant namespaces. |
| `POST /api/v1/orgs/:org/namespaces/:id/verify` | Verify a namespace (platform administrator). |
| `GET /api/v1/orgs/:org/artifact-config` | Show the tenant's private artifact scope and proxy mount. |
| `PUT /api/v1/orgs/:org/artifact-config` | Configure the tenant's private artifact target. |
| `PATCH /api/v1/skills/:package/visibility` | Change visibility. |
| `GET/POST /api/v1/skills/:package/grants` | List or create user grants. |
| `DELETE /api/v1/skills/:package/grants/:id` | Revoke a grant. |
| `GET/POST/DELETE /api/v1/skills/:package/maintainers` | Manage maintainers. |

Organization-scoped routes compare the path organization with the verified token's `organization_id`. The catalog does not trust a client-supplied tenant header.

## SCIM routes

SCIM 2.0 provisioning is enabled when `SKILL_REGISTRY_SCIM_TOKEN` is set. It uses a separate bearer token, not a Logto JWT.

| Route | Purpose |
| --- | --- |
| `GET /scim/v2/ServiceProviderConfig` | SCIM capability discovery. |
| `GET /scim/v2/ResourceTypes` | User and Group resource types. |
| `GET/POST /scim/v2/Users` | List or create users. |
| `GET/PUT/PATCH/DELETE /scim/v2/Users/:id` | User lifecycle. |
| `GET/POST /scim/v2/Groups` | List or create groups mapped to tenants. |
| `GET/PUT/PATCH/DELETE /scim/v2/Groups/:id` | Group and membership lifecycle. |

## Public Registry routes

These routes are unauthenticated, read-only, and CORS-enabled for the static Registry page.

| Route | Purpose |
| --- | --- |
| `GET /api/v1/public/registry/capabilities` | Report `externalSearch.enabled` and the provider name |
| `GET /api/v1/public/registry/skills` | List or search catalog summaries |
| `GET /api/v1/public/registry/external/skills` | Search skills.sh (`q`) or browse it (`view`, `page`) |
| `GET /api/v1/public/registry/external/skill` | Return the skills.sh detail used by the drawer (`source`, `skillId`) |

`/external/skills` accepts `q`, `limit`, `view` (`all-time`, `trending`, `hot`), and `page`. `q` requires at least 2 characters. `/external/skill` returns sanitized `summaryHtml` and `contentHtml` plus install metadata.

The site-facing aliases `GET /api/skills-sh/capabilities`, `/api/skills-sh/search`, `/api/skills-sh/browse`, and `/api/skills-sh/skill` map to the same handlers. The Cloudflare Worker serves these paths in the deployed site; the catalog server mounts them for self-hosted/Docker deployments.

When `SKILLS_SH_ENABLED=false`, the external routes return `404 external_search_disabled` and the capabilities response reports the feature as disabled.

`GET /api/v1/skills` accepts `q`, `package_name`, `source_type`, `include_retired`, `include_external`, `source`, `registry`, `scope`, `origin`, `sort`, `pageSize`, and `cursor`.

- `source_type` accepts `native`, `imported`, `external`, and `platform_created`.
- `include_external` enables the configured external adapter.
- The current adapter recognizes `source=catalog`, `source=skills.sh`, and `source=all`.
- `sort` accepts `relevance`, `updated_at_desc`, and `package_name_asc`.

List responses contain `items` and `pageInfo`. Catalog records may include identity, registry, lifecycle, and provenance fields such as `sourceKind`, `sourceLocator`, origin URLs, `importMode`, `hosted`, `mirrored`, and `installable`.

`GET /api/v1/skills/:package/versions` accepts `include_deprecated`, `include_yanked`, `pageSize`, and `cursor`. `GET /api/v1/skills/:package/install-target` accepts an optional exact `version`.

Public install targets may use `registryAccessMode=proxy` or `direct`. Organization/private targets always use the catalog proxy with `artifactAccess=catalog_proxy` and `authorizationRequired=true`. The CLI sends the catalog bearer token to the proxy; the proxy checks the active tenant and install permission on every request and never exposes the upstream npm URL or credential.

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
