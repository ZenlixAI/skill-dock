# Identity, ownership, authorization, tags, and favorites

Status: **Implemented baseline (Phases 0–6)**. The catalog, site, and CLI now include the identity, tenant, ownership, visibility, proxy, tags, favorites, and SCIM foundations described here. Remaining hardening items are called out explicitly.

Date: 2026-11-23

Related documents:

- [Auth and access](04-auth-and-access.md)
- [Governance and visibility](09-governance-and-visibility.md)
- [API endpoint reference](13-api-endpoint-spec.md)
- [CLI contract](02-cli-contract.md)

## 1. Summary

Skill Dock currently has machine credentials, not user identities. The catalog knows a caller's role (`reader`, `authoring`, `publisher`, `operator`) and an optional actor label, but it does not know which person is publishing, retiring, or yanking a skill. It also has no concept of a skill owner, a private skill, a team, a tag, or a favorite.

This document proposes one identity foundation that supports all of the requested capabilities:

| Capability | Design decision |
| --- | --- |
| Authentication | Logto is the identity provider. Skill Dock does not store passwords. |
| Browser and CLI | Logto Single Page App for the site; Logto device flow for the CLI. Both use Bearer tokens. |
| Organizations | One Logto organization is one Skill Dock tenant. Tenant data is isolated. |
| Publisher ownership | Public skills are organization-owned. User-owned records are private drafts only. |
| Skill authorization | Explicit user grants plus tenant-wide visibility, evaluated by one access module. Cross-tenant grants are disabled in v1. |
| Private skills | `public`, `organization`, and `private` visibility, deny-by-default. `unlisted` is deferred. Private bundle bytes are protected by the catalog proxy, which checks the caller and tenant before forwarding to npm. |
| Audit to person | Every mutation records both the human actor, the tenant, and the technical credential. |
| Tags | System tags, tenant tags, and personal favorite labels are separate concepts. |
| Favorites | Per-user records with a local-first fallback; tenant-private favorites are filtered by the active tenant. |
| Backward compatibility | Existing bearer tokens keep working as service principals under feature flags. |

The most important boundary is:

> Logto answers **who the caller is**. Skill Dock answers **what the caller may do to this skill, in this tenant**.

Logto roles and scopes provide coarse product capabilities. They cannot express "the owner of `@acme.skill/pdf`", "this user is a maintainer of one skill", or "this user was granted install access to that private skill". Those are resource-level decisions and must stay in the Skill Dock catalog.

**Tenant model:** a tenant is a Logto organization. The active tenant is carried by the organization-scoped access token (`organization_id` claim), never by a client-supplied header or a user profile field. A user can belong to several organizations; the client chooses an active organization and obtains a token for it. A request only sees data from the active tenant, plus intentionally global public skills. The catalog applies this rule at every read, write, cache, search, and audit boundary.

**Artifact boundary:** Skill Dock is the policy enforcement point for catalog metadata and private bundle bytes. npm remains the artifact store. Public packages may be served through the catalog proxy or a public direct URL. Organization and private packages are served **only** through the catalog proxy, which validates the caller's identity, active tenant, visibility, and install permission before forwarding to npm. This gives users one identity (Logto) and makes revocation immediate, at the cost of making the catalog proxy a critical, security-sensitive service.

### 1.1 Decisions recorded on 2026-11-23

| Decision | Choice |
| --- | --- |
| Public skill ownership | Organization-only. Individuals may create private drafts, but cannot own a public skill. |
| `unlisted` visibility | Deferred. V1 supports `public`, `organization`, and `private`. |
| Namespace verification | Administrator approval only in v1. DNS/SSO verification is a later option. |
| Artifact authorization | Catalog proxy with per-request authorization. npm is storage. Public packages may be anonymous; organization/private packages require a valid Logto user or service token with tenant + install permission. Revocation takes effect on the next request. |
| Browser authentication | Bearer tokens only. No cookie BFF in v1. |
| Multi-tenancy | Strict isolation per Logto organization. Active tenant comes from the organization token. |

Non-goals for this design:

- building a password, MFA, or password-reset implementation in Skill Dock;
- SCIM/HR-driven user lifecycle in the first release;
- billing, plans, or marketplace payments;
- public comments, ratings, or social feeds;
- public per-skill favorite counts;
- a cookie-based BFF session store (kept as a later option);
- replacing the npm-compatible artifact repository.

## 2. Current state and gaps

| Area | Today | Consequence |
| --- | --- | --- |
| Authentication | Static bearer tokens in `SKILL_REGISTRY_READER_TOKEN`, `SKILL_REGISTRY_AUTHORING_TOKEN`, `SKILL_REGISTRY_PUBLISHER_TOKEN`, or `SKILL_REGISTRY_AUTH_TOKENS_JSON`. Development `X-Role` header when no token is configured. | A shared token cannot identify a person. A leaked token cannot be revoked per user. |
| Principal | `auth.Principal` has `Role`, `Actor`, `AuthMechanism`, and `AllowedPackages` in [`internal/platform/auth/authorizer.go`](../internal/platform/auth/authorizer.go). | `Actor` defaults to the role name. There is no stable user ID, organization ID, display name, or email. |
| Audit | `audit_events.actor` and `role` in [`internal/platform/persistence/audit_store.go`](../internal/platform/persistence/audit_store.go). | Multiple humans sharing a publisher token produce indistinguishable audit rows. |
| Governance | `governance_records.changed_by` is a string passed from `principal.Actor`. | "Who retired this version?" resolves to `publisher`, not a person. |
| Catalog | `skills` contains display metadata, status, and latest version. No owner, visibility, tags, or grants. | Any publisher token can publish any package unless a package allow-list is configured. |
| Public reads | `/api/v1/public/registry/*` is unauthenticated and CORS-enabled in [`internal/backend/httpserver/public_registry.go`](../internal/backend/httpserver/public_registry.go). | There is no safe way to add private skills without changing this surface. |
| Artifact proxy | `/repository/skills-npm/*` forwards reads to the artifact repository and strips the caller's token in [`internal/backend/httpserver/npm_registry_proxy.go`](../internal/backend/httpserver/npm_registry_proxy.go). | A private bundle could be fetched by anyone who knows its URL unless npm protects the package or the proxy is restricted to public packages. |
| Read cache | Cache key contains method, path, query, role, and allowed packages in [`internal/backend/httpserver/server.go`](../internal/backend/httpserver/server.go). | It cannot distinguish two users in the same role, or two organizations. |
| Site | The Registry page reads the public skills.sh directory through `/api/skills-sh/*`; there is no account UI. | Favorites can only be local until the site has an identity and a catalog connection. |
| CLI | `SKILLS_REGISTRY_TOKEN` is a static environment token. | No `skills login`, no per-user revocation, no organization selection. |
| Tags | Not present. | No filtering, curation, or tenant taxonomy. |
| Favorites | Not present. | No personal collection, no cross-device continuity. |

## 3. Features that need an identity system

This is the direct answer to "what else needs identity?" The table separates hard requirements from features that can work anonymously but become materially better with identity.

| Feature | Needs identity? | Why | Priority |
| --- | --- | --- | --- |
| Publisher ownership and namespace claims | **Hard** | An owner must be a durable subject. Otherwise any publisher can impersonate `@acme.skill/*`. | P0 |
| Per-person audit and non-repudiation | **Hard** | A shared token can only identify a role. Governance actions need a person or a named service account. | P0 |
| Private and tenant-only skills | **Hard** | Access is evaluated per viewer and per tenant. Anonymous callers cannot be granted membership. | P0 |
| Skill authorization and grants | **Hard** | Grants are attached to users within an owning tenant. Cross-tenant grants are deliberately disabled in v1. | P0 |
| Tenant membership and roles | **Hard** | Tenant-scoped publishing, admin, and approval workflows require members. | P0 |
| Tenant isolation | **Hard** | One Logto organization is one tenant. Every tenant-owned row, query, cache entry, and artifact check needs a tenant boundary. | P0 |
| CLI login and personal access tokens | **Hard** | Replacing shared environment tokens with revocable per-user credentials. | P1 |
| Favorites with cross-device sync | **Soft** | Local storage works for one browser. Server sync needs a stable user. | P1 |
| Tags | **Soft** for reading; **hard** for authoring | Reading a system tag is public. Creating tenant tags, assigning them, and attributing changes needs an actor. | P1 |
| Notifications and subscriptions | **Hard** | "Notify me when this skill changes" needs a recipient and preferences. | P1 |
| Ratings, reviews, comments, reports | **Hard** | One-person-one-vote and abuse handling require identity. | P2 |
| Install analytics by unique user or org | **Hard** | Unique adoption cannot be computed from anonymous downloads without a stable or pseudonymous ID. | P2 |
| Quotas, plans, billing | **Hard** | Usage must attach to a paying account or organization. | P2 |
| Public publisher profiles | **Hard** | A profile is a durable identity with owned skills. | P2 |
| Moderation and takedowns | **Hard** | Reports, decisions, and appeals need accountable actors. | P2 |
| Data export, account deletion, consent | **Hard** | These are user rights that only exist once there is a user record. | P1 |

The practical conclusion is that favorites alone do not justify an identity system. Ownership, private visibility, grants, and per-person audit do. Once those exist, favorites and notifications are cheap additions.

## 4. Design principles

1. **One identity provider.** Logto owns authentication, user profiles, organizations, and coarse roles. Skill Dock does not implement passwords, MFA, or password reset.
2. **Skill Dock owns resource authorization.** Ownership, visibility, grants, and package namespaces live in the catalog database and are evaluated by one module.
3. **Deny by default.** A skill with no access record is legacy-public only during migration. New private resources are invisible until a grant exists.
4. **Separate user and service principals.** CI and automation keep token-based service accounts. Humans use Logto. Both produce audit records.
5. **Do not couple ACL state to the release aggregate.** `skills`, `skill_versions`, and governance records are release state. Ownership, visibility, tags, grants, and favorites are separate tables so aggregate writes do not accidentally rewrite access policy.
6. **Tags are metadata, not authorization.** Explicit grants are the source of truth. Tag-based policy may be added later as a tenant policy selector, never as the only access control.
7. **Additive migration.** Existing public reads, tokens, and CLI workflows keep working while feature flags are rolled out.
8. **One access seam.** Handlers ask one module for a decision. They do not query grants, roles, and visibility independently.
9. **Tenant context is explicit.** Tenant-owned repository methods require a tenant ID. There is no "all tenants" default outside platform administration.
10. **Public skills are the deliberate exception.** A public skill is globally readable and installable, but it is still owned and managed by exactly one tenant. Public does not mean unowned.
11. **Personal drafts are not tenant data.** User-owned records are private drafts in a personal workspace, not members of a tenant. Promoting a draft to a tenant creates a new package identity under that tenant's namespace.

## 5. Identity with Logto

### 5.1 What Logto owns and what Skill Dock owns

| Concern | Owner |
| --- | --- |
| Sign-in, sign-up, MFA, social/SSO connectors | Logto |
| User profile, email, display name, avatar | Logto (cached in Skill Dock) |
| Tenant (organization) creation, membership, invitations | Logto |
| Global roles and organization role templates | Logto |
| Coarse API scopes | Logto |
| Tenant metadata projection and tenant lifecycle guards | Skill Dock |
| Package namespace ownership | Skill Dock |
| Per-skill owner, maintainer, visibility, grants | Skill Dock |
| Tenant tags and tag assignments | Skill Dock |
| Favorites | Skill Dock |
| Audit records and governance history | Skill Dock |

### 5.2 Logto applications

| Application | Logto type | Used by | Notes |
| --- | --- | --- | --- |
| `skill-dock-web` | Single Page App | The React site | Public client, Authorization Code + PKCE, refresh-token rotation. No client secret in the browser. |
| `skill-dock-cli` | Native app with device flow | `skills` CLI | Public client. `skills login` displays a user code and polls the token endpoint. |
| `skill-dock-cicd` | Machine-to-machine | CI publishing, optional sync jobs | Client credentials. Secret stored in the deployment secret manager. |
| `skill-dock-sync` | Machine-to-machine | Logto Management API reconciliation | Optional if webhooks are not enough. Needs a Management API role. |

The web and CLI apps should not share a client ID. Different clients make it possible to revoke one surface without breaking the other and keep redirect rules narrow.

### 5.3 API resource and scopes

Register one Logto API resource whose identifier is the public catalog API base, for example:

```text
https://catalog.example.com/api
```

Suggested permissions:

| Scope | Meaning |
| --- | --- |
| `catalog:read` | Read catalog metadata visible to the caller. |
| `catalog:install` | Resolve install targets and download allowed bundles. |
| `skill:publish` | Create a release under an owned namespace. |
| `skill:manage` | Edit skill metadata, tags, and visibility. |
| `skill:govern` | Deprecate, yank, retire, or transfer a skill. |
| `org:manage` | Manage namespace claims, grants, tags, and tenant settings in Skill Dock. |
| `audit:read` | Read audit events for the caller's scope. |
| `favorites:read` | Read the caller's favorites. |
| `favorites:write` | Add or remove the caller's favorites. |

Request only what a surface needs. The Registry page needs `catalog:read` and favorites scopes. The CLI login needs `catalog:read`, `catalog:install`, `skill:publish`, and optionally `skill:govern` for maintainers.

### 5.4 Tenant roles

Use a single Logto organization template. A Logto organization is a tenant. The role names are part of the API contract:

| Logto organization role | Skill Dock capabilities within that tenant |
| --- | --- |
| `owner` | Everything `admin` can do, plus namespace claims, ownership transfer, and tenant settings in Skill Dock. |
| `admin` | `skill:publish`, `skill:manage`, `skill:govern`, `org:manage`, `audit:read`. |
| `publisher` | `skill:publish` on owned namespaces, `skill:manage` on skills they maintain, `favorites:*`. |
| `member` | `catalog:read`, `catalog:install`, `favorites:*`. |
| `viewer` | `catalog:read` only; no install. |

Skill Dock should not trust a role name blindly for high-risk actions. The access module verifies the request's organization token and the local ownership record; the role is an input, not the whole decision.

Global roles:

| Global role | Purpose |
| --- | --- |
| `platform-admin` | Break-glass administration across tenants. Every use is audited. |
| `platform-auditor` | Read-only audit access across tenants. |

Avoid assigning `platform-admin` to normal users. Prefer the tenant `owner` role. A tenant role never grants access to another tenant; only `platform-admin` can cross the boundary, and only through an explicit break-glass path.

### 5.5 Token validation

The catalog validates every Logto access token before using it:

1. Accept only `Authorization: Bearer <token>`.
2. Fetch the OpenID configuration from `{LOGTO_ENDPOINT}/oidc/.well-known/openid-configuration` at startup, or use configured issuer/JWKS values.
3. Verify the JWT signature with Logto's JWKS. Cache keys and refresh on an unknown `kid`.
4. Verify `iss` matches the configured Logto issuer, including the trailing `/oidc`.
5. Verify `aud` contains the registered API resource identifier.
6. Verify `exp` and `nbf` with a small clock skew.
7. Reject `alg: none` and algorithms outside the allow-list.
8. Reject ID tokens. Only access tokens may authenticate API calls.
9. Read `sub`, `client_id`, `scope`, and optional `organization_id`.
10. For a tenant-scoped request, require `organization_id` and reject the request if it does not match the target tenant. There is no tenant fallback and no `X-Tenant-ID` header.
11. For a personal or global request (favorites, public reads), allow a token without `organization_id`; never infer a tenant from it.

JWT validation must fail closed. If the JWKS cannot be fetched at startup, the server starts in a degraded state where existing cached keys may still validate unexpired tokens, but new logins fail. Public reads do not depend on Logto.

### 5.6 User and tenant provisioning

Skill Dock needs local projections for ownership, audit, and joins. It should not call Logto on every request.

**Users**

- JIT provision on the first authenticated request: insert a row keyed by `(issuer, subject)` with a minimal profile.
- Enrich display name, email, and avatar from Logto webhooks (`User.Created`, `User.Data.Updated`) or the Management API.
- Never use an email address as the primary key. People change email addresses. `(issuer, subject)` is stable.
- If the profile is not available yet, audit can show `subject` until enrichment completes.

**Tenants (Logto organizations)**

- Store the Logto organization ID as the external tenant key plus a local tenant UUID.
- Sync membership from `Organization.Membership.Updated` webhooks, with a periodic full reconciliation through the Management API.
- The organization token is authoritative for the *active* tenant. The local projection is used for admin views, notifications, audit joins, and consistency checks, not to widen a request beyond the active tenant.
- An organization role change is reflected in the next token. Keep access token lifetime short (for example 15 minutes) and trust token scopes for coarse capabilities.
- For high-risk mutations (transfer, visibility change, grant change), re-check the local membership projection or Logto Management API, not only the token scope.
- A tenant deletion is blocked while it owns skills, namespaces, or grants. The operator must transfer or retire those resources first.

**Active tenant selection**

- A user with exactly one organization can select it automatically in the client.
- A user with multiple organizations chooses an active organization in the site or with `skills org use <slug>`.
- A user with no organization can browse public skills and work on personal private drafts, but cannot publish into a tenant.
- The server does not infer the tenant from the user's email domain, the first organization in a list, or a query parameter. The client must obtain an organization-scoped access token.
- Switching tenants is a client-side token acquisition. There is no server session to switch and no tenant cookie.

**Why the local membership projection exists**

It is a read model for tenant admin views, audit filtering, notification fan-out, and periodic consistency checks. It is not an authorization shortcut across tenants. If a request carries tenant A's token, it cannot use a projection row from tenant B to read tenant B data. Cross-tenant grants are disabled in v1.

### 5.7 Browser session and logout

Use the Logto SPA SDK (or a standards-based OIDC client) in the site:

- Authorization Code + PKCE.
- Request `openid profile email offline_access` plus API scopes.
- Keep access tokens short-lived; use refresh-token rotation.
- Request an organization token with `organization_id` when the user selects an organization.
- Logout uses the Logto end-session endpoint and clears local state.
- Do not store tokens in `localStorage` if the SDK supports in-memory storage plus a refresh token. If a refresh token must be persisted, use the SDK's recommended storage and document the XSS trade-off.

The first version uses Bearer tokens, not cookies. This keeps the API stateless, matches the existing CLI/token model, and avoids CSRF/session storage in the catalog. A BFF with HttpOnly cookies can be added later for deployments that require it; it is not required for the first rollout.

### 5.8 CLI login

`skills login` uses the OAuth 2.0 device authorization grant:

1. `POST {LOGTO_ENDPOINT}/oidc/device/auth` with the CLI client ID, API resource, scopes, and `offline_access`.
2. Display `user_code` and `verification_uri` (or `verification_uri_complete`).
3. Poll `POST {LOGTO_ENDPOINT}/oidc/token` with `grant_type=urn:ietf:params:oauth:grant-type:device_code` at least every 5 seconds.
4. On success, store the refresh token in the OS keychain when available. Fall back to a `0600` file under `${XDG_CONFIG_HOME:-~/.config}/skill-dock/credentials.json`.
5. Access tokens are used for API calls and refreshed as needed.
6. `skills whoami` calls `GET /api/v1/me`.
7. `skills org list` lists organizations from the ID token or `/api/v1/me/orgs`; `skills org use <slug>` stores the active organization locally.
8. `skills logout` revokes the refresh token when the provider supports revocation and deletes the local credential.

Keep `SKILLS_REGISTRY_TOKEN` for CI and headless environments. Do not silently fall back from a user login to a shared token; require an explicit flag or environment variable.

### 5.9 Example token claims

Global user access token:

```json
{
  "iss": "https://logto.example.com/oidc",
  "aud": "https://catalog.example.com/api",
  "sub": "user_01H...",
  "client_id": "skill-dock-web",
  "scope": "openid profile email catalog:read favorites:read favorites:write",
  "exp": 1790000000
}
```

Tenant (organization) access token:

```json
{
  "iss": "https://logto.example.com/oidc",
  "aud": "https://catalog.example.com/api",
  "sub": "user_01H...",
  "client_id": "skill-dock-web",
  "scope": "catalog:read catalog:install skill:publish favorites:read favorites:write",
  "organization_id": "org_01H...",
  "exp": 1790000000
}
```

ID token (used by the client for organization discovery, not sent to the API as authentication):

```json
{
  "sub": "user_01H...",
  "organizations": ["org_01H...", "org_02H..."],
  "organization_roles": ["org_01H...:admin", "org_02H...:member"]
}
```

Request `urn:logto:scope:organizations` when the client needs the `organizations` claim, and `urn:logto:scope:organization_roles` when it needs `organization_roles`. These are discovery aids for the UI; the API still verifies the active organization token and the local access record.

Logto does not currently issue organization tokens directly from the authorization code flow. The SPA SDK and CLI obtain them through the refresh-token flow. This is one reason the design keeps a refresh token and does not try to make organization tokens optional.

## 6. Domain model

### 6.1 Ownership

A skill has exactly one owner. The owner is a tenant (Logto organization), a user for a private draft, or the platform for legacy data.

- `owner_type`: `organization`, `user`, or `platform`.
- `owner_id`: tenant ID, user ID, or `NULL` for platform.
- `created_by`: the user who first published the skill, when known.
- `maintainers`: additional users in the owning tenant who can publish and manage metadata.

**Public skills are organization-owned.** A public skill must have `owner_type=organization`, `owner_id=<tenant>`, a verified namespace, and `visibility=public`. A user cannot own a public skill. This is the decision recorded for v1.

**User-owned records are private drafts.** They are not tenant data and are not public:

- `owner_type=user`, `owner_id=<user>`, `visibility=private`.
- They live under a reserved personal namespace such as `@draft.<user-id>/` so they cannot collide with a tenant namespace.
- Only the owner can read, edit, publish, or delete a draft.
- A draft is not installable by other users.
- Promoting a draft to a tenant creates a **new package identity** under the tenant's verified namespace. The draft source is copied into the new package; the draft is not renamed in place. This avoids changing the identity of an immutable release.
- A draft can be exported or deleted without affecting tenant data.

**Maintainers** are separate from ownership:

- `owner` can transfer, change visibility, manage grants, and retire.
- `maintainer` can publish new versions, edit metadata, and assign tags.
- A maintainer must be a member of the owning tenant. Cross-tenant maintainers are not supported in v1.
- `publisher` role is required to publish; maintainer status grants the resource-level permission.
- A maintainer does not automatically get governance rights. Governance (`deprecate`, `yank`, `retire`) belongs to `admin`/`owner` or a platform administrator.
- Ownership transfer is within the same tenant. Moving a skill between tenants is a controlled migration, not a normal transfer, and is out of scope for v1.

### 6.2 Package namespaces

Package names remain the canonical skill identity: `@<company>.skill/<skill-name>`. A namespace is a prefix such as `@acme.skill/`.

- `package_namespaces` stores the prefix, owning tenant, status, and verification method.
- Namespace prefixes are unique and matched longest-prefix-first.
- A namespace belongs to exactly one tenant. Two tenants cannot claim the same prefix.
- A verified namespace is required to publish a public skill under that prefix.
- An unverified or pending namespace can publish `organization` or `private` skills only.
- Existing unowned packages remain `platform` owned and public until an administrator assigns them to a tenant.
- Verification is **administrator approval only** in v1. A platform administrator reviews the claim and marks it verified. DNS and SSO verification are later options; the schema keeps a `verification` column so they can be added without a migration.
- Personal drafts use the reserved prefix `@draft.<user-id>/`, which is never a tenant namespace and can never be public.
- A tenant's private npm scope must be a subset of a namespace it owns. Publishing an organization/private skill outside that scope fails. This keeps the catalog namespace and the npm authorization boundary aligned.

Namespace enforcement has three modes:

| Mode | Behavior |
| --- | --- |
| `off` | Current behavior. Any allowed publisher token can publish. |
| `warn` | Publishing outside an owned namespace is allowed but produces an audit warning. |
| `on` | Publishing requires an owner and a matching verified namespace for public skills. |

This lets existing deployments migrate without a flag day.

### 6.3 Visibility

`skill_access.visibility` is one of three values in v1:

| Visibility | Who can read metadata | Who can install | Listed in public search | Tenant scope |
| --- | --- | --- | --- | --- |
| `public` | Anyone, including anonymous | Anyone, subject to version status and install policy | Yes | Global, but owned by one tenant |
| `organization` | Members of the owning tenant | Members of the owning tenant | No | Active tenant must match the owner |
| `private` | Explicit user `view` grants, owner, maintainers | Explicit `install` grants, owner, maintainers | No | Active tenant must match the owner |

Notes:

- **`unlisted` is deferred.** It would mean "not in search or browse, but anyone who knows the exact package name can read and install it". It is useful for share-by-link, but with multi-tenancy it creates an ambiguous security boundary: is it global-unlisted or tenant-unlisted? V1 avoids that ambiguity. Use `organization` for tenant-wide access and `private` plus user grants for narrower sharing.
- Public skills are intentional global resources. A public skill is still owned and managed by exactly one tenant. Other tenants can read, install, and favorite it; they cannot edit, govern, transfer, or see its tenant-private tags.
- `organization` means the owning tenant only. It never means "any organization the user belongs to".
- `private` means explicit grants within the owning tenant. It never crosses a tenant boundary in v1.
- Visibility and install permission protect both **metadata and private bundle bytes** through the catalog proxy. Public packages may be fetched anonymously; organization/private packages require a valid token and tenant context on every request; see [7.6](#76-install-authorization-and-the-catalog-proxy).
- Visibility is independent of release status. A public skill can have a yanked version; a private skill can be resolvable.
- Anonymous public routes must only return `public` skills.
- A private or organization skill that the caller cannot see should return `404 skill_not_found` on reads to avoid leaking existence. Mutations by an authenticated but unauthorized caller may return `403 forbidden` when the resource is already known through another authorized surface; document the choice per endpoint.
- A request for a tenant-owned skill with a different active tenant returns `404` on reads and `403 tenant_mismatch` on mutations.

### 6.4 Grants and permissions

`skill_grants` is an explicit allow-list within the owning tenant:

| Column | Meaning |
| --- | --- |
| `id` | Stable grant ID. |
| `organization_id` | Owning tenant. Required; grants never cross tenants. |
| `package_name` | Target skill. |
| `subject_type` | `user` in v1; `team` is reserved for a future app-level team model. |
| `subject_id` | Local user ID (or future team ID). |
| `permission` | `view`, `install`, `publish`, or `manage`. |
| `granted_by` | User who created the grant. |
| `expires_at` | Optional expiry. |
| `created_at` | Creation time. |

Grant semantics:

- One grant row per subject per skill. The `permission` value is the highest granted level; updating a grant replaces the level.
- `install` implies `view`.
- `publish` implies `view` and `install`.
- `manage` implies `publish`.
- `govern` is not grantable in the first release. It stays with owner/admin/platform administrator.
- The grant subject must be a member of the owning tenant. A user outside the tenant cannot receive a grant in v1.
- **Cross-tenant grants are disabled in v1.** If cross-tenant sharing is required later, it should be modeled as an explicit share that creates a read-only projection inside the recipient tenant, with its own audit and revocation path. It must not be implemented by adding a row that lets one tenant read another tenant's tables.
- Revoking a grant takes effect on the next request. Do not rely on a cached authorization decision for mutations.

**Permission matrix**

The `Member`, `Publisher`, `Admin`, and `Owner` columns mean roles in the **owning tenant**. A role in another tenant grants nothing here.

| Action | Anonymous | Tenant member | Publisher | Admin | Owner | Platform admin | Service publisher |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Read public skill | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Read organization skill | No | Yes | Yes | Yes | Yes | Yes | If scoped |
| Read private skill | No | Grant only | Grant only | Grant only | Yes | Yes | If scoped |
| Install public skill | Yes | Yes | Yes | Yes | Yes | Yes | Yes |
| Install private skill | No | Grant only | Grant only | Grant only | Yes | Yes | If scoped |
| Publish new version | No | No | Yes, owned namespace | Yes | Yes | Yes | If allowed packages |
| Edit metadata/tags | No | No | Maintained skills | Tenant skills | Tenant skills | Yes | No |
| Change visibility/grants | No | No | No | Tenant skills | Tenant skills | Yes | No |
| Deprecate/yank/retire | No | No | No | Tenant skills | Tenant skills | Yes | No |
| Transfer ownership | No | No | No | No | Yes, same tenant | Yes | No |
| Manage namespace | No | No | No | No | Yes | Yes | No |

The matrix is intentionally conservative. A missing permission is a denial, not an invitation to infer one from role names. `Platform admin` is a break-glass role; every cross-tenant action is audited with `actor_type=platform_admin` and the target tenant.

`Install` in this matrix means "Skill Dock may issue an install target **and** the catalog proxy may forward the private bundle for this caller". For public skills, anyone may install. For organization/private skills, the proxy re-checks the caller and tenant on every request, so a revoked grant stops the next download without an npm credential change.

### 6.5 Tags

There are three different tagging concepts. Keeping them separate avoids confusing metadata with access control.

| Concept | Scope | Purpose |
| --- | --- | --- |
| System tags | Platform | Curated discovery and policy labels, for example `security`, `frontend`, `official`. |
| Tenant tags | One tenant | Internal taxonomy, for example `team:payments`, `tier:internal`. Visible only to tenant members. |
| Personal favorite labels | One user | Private organization of saved skills, for example `to-try`, `for-work`. |

Catalog tags:

- `tags` table: `id`, `slug`, `label`, `scope` (`system` or `tenant`), `organization_id`, `visibility` (`public` or `private`), `created_by`, timestamps.
- `skill_tags` table: `organization_id`, `package_name`, `tag_id`, `assigned_by`, `assigned_at`.
- Unique `(scope, organization_id, slug)` and `(organization_id, package_name, tag_id)`.
- Public responses show system tags and tags explicitly marked public. Tenant-private tags are only returned to members of that tenant.
- A skill maintainer can assign tags they can see. Creating a tenant tag requires `org:manage` or a tenant admin.
- Tag filters use `tag=<slug>` with `tag_mode=all|any`; default is `all`.
- Tags are not ACLs. A tenant policy may later say "only allow skills tagged `security`", but that is a policy layered on top of explicit grants, not a replacement for them.
- Tenant tags never cross the tenant boundary. A public skill shows system tags and only those tenant tags explicitly marked public; a tenant-private tag is never returned to another tenant.

Why tags need identity for authoring: a tag assignment needs an attributed actor, and tenant tags need a tenant. Read-only system tags do not require identity.

### 6.6 Favorites

Favorites are per user. They can point to a catalog skill or an external skills.sh skill.

- Catalog reference: `package_name`.
- External reference: provider (`skills.sh`), source (`owner/repo`), and skill ID.
- A favorite stores a small snapshot for display (`name`, `source`, `install command`, timestamp) but the package name or external tuple is authoritative.
- Each favorite has a context derived from the referenced skill, not supplied by the client: `global` for public/external skills, `tenant` for organization/private skills, and `personal` for the user's own drafts.
- A favorite of a private or organization skill is tenant-scoped. It is only returned while the active tenant matches the skill's owner and the user can view that skill. If the user switches to another tenant, that favorite is hidden, not shown as a cross-tenant reference.
- A favorite of a public or external skill is `global` and can be shown in any tenant context.
- A personal draft favorite is `personal` and only shown in the personal workspace, never in a tenant context.
- If access is revoked, the favorite remains but is returned with `available: false` and no private metadata. It is never used to leak the skill name or owner to another tenant.
- Favorites do not imply install or publish permission. `favorites:write` only writes the caller's own list.
- Favorite counts are not exposed in the first release. They are private data and would need abuse controls.

Anonymous visitors can favorite locally in the browser. On sign-in, the client sends the local references to a merge endpoint. The server deduplicates by reference key, keeps the earliest creation time, and only accepts tenant-private references that are valid for the active tenant. On explicit logout, clear the local mirror unless the user chose to keep it on that device.

### 6.7 Audit

Audit records must answer both "what happened" and "who caused it". Extend `audit_events` with:

| Column | Meaning |
| --- | --- |
| `actor_type` | `user`, `service`, `anonymous`, `system`, `platform_admin`. |
| `actor_user_id` | Local user ID, when the actor is a person. |
| `actor_subject` | Logto `sub`, when available. |
| `actor_organization_id` | Active tenant context in the token, when present. |
| `organization_id` | Tenant that owns the affected resource. Required for tenant-scoped events. |
| `actor_display` | Display name or email snapshot. |
| `token_id` | Personal token or service-token ID. |
| `client_id` | Logto application/client ID. |
| `auth_mechanism` | `logto_browser`, `logto_device`, `m2m`, `legacy_token`, `dev_header`. |

Keep `actor` and `role` for backward-compatible API responses. Set `actor` to a human-readable display value, but never parse it for authorization.

Governance records get `changed_by_user_id`, `changed_by_organization_id`, and `changed_by_display` in addition to the existing `changed_by`. Existing rows are migrated with `actor_type=legacy`.

Audit policy:

- Audit every mutation, every grant/visibility change, every namespace claim, and every authentication/authorization denial.
- Audit reads of private skills and every platform-admin cross-tenant action. Do not write an audit row for every public read; use access logs and metrics for volume.
- Always record both the actor's active tenant and the resource tenant. An audit query for tenant A must never return tenant B events unless the caller is a platform auditor with an explicit cross-tenant scope.
- Never store raw bearer tokens, refresh tokens, or passwords in audit details.
- Define a retention period and a redaction policy for email addresses before production.

## 7. Authorization

### 7.1 Principal types

Extend `auth.Principal` without breaking existing callers:

| Field | Anonymous | User | Service |
| --- | --- | --- | --- |
| `Kind` | `anonymous` | `user` | `service` |
| `UserID` | empty | local user UUID | empty |
| `Subject` | empty | Logto `sub` | empty |
| `TenantID` | empty | active Logto organization ID, optional | optional service tenant |
| `Role` | empty | mapped global/tenant role | legacy role |
| `Scopes` | empty | token scopes | configured scopes |
| `Actor` | empty | display name or `user:<id>` | configured actor |
| `TokenID` | empty | empty | service-token ID |
| `AuthMechanism` | `anonymous` | `logto_browser` / `logto_device` | `m2m` / `legacy_token` |
| `PlatformAdmin` | false | from global role | configured |

The authentication middleware becomes a composite:

1. If the bearer token matches a configured service token, return a service principal.
2. Otherwise, if Logto is configured, validate the JWT and return a user principal.
3. If no credential is present, return an anonymous principal.
4. In development only, allow `X-Role`/`X-Actor` behind an explicit flag. Never enable that path in production.

### 7.2 One access seam

Add one module, for example `internal/access`, with a deliberately small interface:

```go
type Action string

const (
    ActionRead    Action = "read"
    ActionInstall Action = "install"
    ActionPublish Action = "publish"
    ActionManage  Action = "manage"
    ActionGovern  Action = "govern"
    ActionGrant   Action = "grant"
    ActionTransfer Action = "transfer"
)

type TenantID string

type Resource struct {
    TenantID    TenantID
    PackageName string
    OwnerType   string
    OwnerID     string
    Visibility  string
}

type Decider interface {
    Decide(ctx context.Context, principal auth.Principal, action Action, resource Resource) (Decision, error)
    VisibleFilter(ctx context.Context, principal auth.Principal, tenant TenantID) (Filter, error)
}
```

`Decision` is `Allowed` or `Denied` with a reason code for audit and metrics. The module hides:

- tenant boundary enforcement;
- platform administrator break-glass override;
- ownership and maintainer checks;
- tenant membership projection;
- explicit user grants;
- visibility rules;
- namespace ownership for publish;
- legacy token compatibility.

Handlers and services call this module. They do not join `skill_grants` themselves. List endpoints use `VisibleFilter` to build a SQL predicate, so the same rules apply to list, search, detail, and install. A tenant-scoped call must pass the tenant from the verified token, not from a query parameter.

**Deletion test:** if this module were removed, tenant isolation, ownership, visibility, grants, namespace, role, and legacy compatibility logic would reappear in read handlers, write handlers, CLI-facing install resolution, and public registry routes. That is exactly the complexity the module exists to hide.

### 7.3 Decision algorithm

For a resource-level action:

1. Resolve the resource tenant. Public skills have a global read surface but still have an owning tenant for management. Personal drafts have no tenant.
2. If the action is `read` or `install` and visibility is `public`, allow, including anonymous callers.
3. If the action is a management action (`publish`, `manage`, `govern`, `grant`, `transfer`) and the resource is a personal draft, allow only the owner user.
4. If the resource belongs to a tenant and the action is not a global public read, require a user principal with `TenantID == resource.TenantID`. A mismatch is a denial, not a reason to consult another tenant.
5. If the principal is a platform administrator, allow through the break-glass path and record the target tenant in audit.
6. If the principal is anonymous, deny.
7. If the principal is the owner user, allow `read`, `install`, `publish`, `manage`, and `grant` according to owner rights.
8. If the resource is owned by a tenant:
   - If the principal's active tenant matches the owner and the user's tenant role allows the action, allow.
   - If the user is a member of the owning tenant and the action is `read`/`install`, allow for `organization` visibility.
   - If the role allows `publish`, `manage`, or `govern`, allow within that tenant.
9. If a user grant in the owning tenant matches, allow the granted action and implied actions.
10. If a service principal is bound to the owning tenant and its allowed packages/namespaces match, allow the configured action.
11. Deny.

For `publish` on a new skill, also require:

- a matching owned namespace if namespace enforcement is `on`;
- the `skill:publish` scope;
- for public visibility, an organization owner and a verified namespace;
- an owner assignment. A user-owned result is only allowed for `private` drafts.

For `transfer`, require the current owner's `owner` role and the target owner's existence **in the same tenant**. Do not allow a self-service transfer into another tenant or a namespace the target does not own. Moving a skill between tenants is a controlled migration outside normal transfer.

For `grant`, require `manage` on the resource and a subject who is a member of the owning tenant. Cross-tenant subjects are rejected in v1.

### 7.4 List filtering and search

`VisibleFilter` should produce a SQL predicate equivalent to:

```sql
(
  visibility = 'public'
  OR (visibility = 'organization' AND organization_id = :active_tenant)
  OR (visibility = 'private' AND organization_id = :active_tenant
      AND EXISTS (SELECT 1 FROM skill_grants g
                  WHERE g.organization_id = :active_tenant
                    AND g.package_name = skill_access.package_name
                    AND g.subject_type = 'user'
                    AND g.subject_id = :user_id
                    AND g.permission IN ('view','install','publish','manage')))
)
```

The actual implementation may use joins or a materialized visibility table. The invariant is that list, search, detail, and install use the same rule, and every tenant-owned branch filters on `organization_id = :active_tenant`.

Public search and the public registry routes only use `visibility = 'public'`. They must not run the authenticated filter and must not accept a caller token to widen results. A caller from tenant A cannot see tenant B's `organization` or `private` rows even if they know a package name.

### 7.5 Cache behavior

The current shared read cache is safe only for public data.

- Public list/detail responses: keep the shared cache with a short TTL. Public data is global by design.
- Authenticated or tenant-private responses: do not store in the shared cache. Set `Cache-Control: private, no-store`.
- If a per-user or per-tenant cache is added later, key it by user ID, active tenant ID (from the verified token), and an access revision. Keep it small and short-lived.
- When ownership, visibility, grants, tags, or tenant assignment change, bump a package-level access revision or invalidate the public cache entry. A 2–5 second stale window is acceptable for public metadata, but not for tenant-private data.
- Never cache a response for one principal or tenant and serve it to another. A cache key must include the tenant for every non-public path, even if the current response happens to contain only public rows.

### 7.6 Install authorization and the catalog proxy

Skill Dock authorizes both the **install target** and the **private bundle bytes**. npm remains the artifact store. This gives users one identity (Logto) and makes access revocation immediate, but it makes the catalog proxy a security-critical service.

**Public skills**

- The catalog may return a public npm registry URL or a public catalog proxy URL.
- The catalog proxy may serve public packages anonymously. It can use the server's read credential because the package is public.
- Public installation remains usable without a Logto account.

**Organization and private skills**

- The install-target request checks tenant, visibility, and `ActionInstall` before issuing a URL.
- The response uses the catalog proxy, `registryAccessMode=proxy`, and `authorizationRequired: true`. It carries `artifactAccess: "catalog_proxy"`.
- The CLI sends the catalog Bearer token when downloading. It does not send an npm credential. The catalog token is never forwarded to the upstream npm registry.
- The catalog proxy authenticates the token, parses the npm path to a package name and version, loads the owning tenant, and checks that the caller can install that package **in the active tenant**. It then forwards the request to the upstream npm repository with the server's read credential.
- The upstream URL and credential are never returned to the client.

**Proxy authorization flow**

1. `GET /api/v1/skills/:package/install-target` checks `ActionInstall`.
2. For a non-public skill, the response contains a proxy URL and `authorizationRequired: true`.
3. The CLI downloads from the proxy with `Authorization: Bearer <catalog-token>`.
4. The proxy validates the token using the same Logto/service-token authenticator as the API.
5. The proxy parses the npm path into a package name and version. It rejects malformed, encoded traversal, and non-GET/HEAD requests.
6. The proxy loads `skill_access` and the tenant. It checks:
   - the package's current visibility allows this request;
   - the caller's `TenantID` matches the owning tenant for organization/private skills;
   - the caller has `ActionInstall` (owner, tenant member for `organization`, explicit grant for `private`, or a scoped service principal).
7. The proxy resolves the package in the catalog. Unknown packages are `404`; the proxy is not an open npm mirror.
8. The proxy forwards the request to the configured upstream npm repository with a read-only server credential.
9. The proxy streams the response. It does not cache private bytes in a shared cache.
10. The proxy records the download event (package, version, tenant, actor, status) in access logs and metrics. High-volume raw download rows are not written to the audit table; private install denials and administrative downloads are audited.

**Tenant isolation**

- The token's `organization_id` is authoritative. A tenant A token cannot download tenant B's private package even if it knows the exact URL.
- Public packages have no tenant check.
- A service principal must be bound to the owning tenant or explicitly allowed for the package. A platform-wide service credential must not be used as the normal proxy read credential.
- The proxy must resolve the package from the request, not trust a `X-Tenant-ID` header or a query parameter.

**Path parsing and request safety**

- Support scoped npm names (`@scope/name`), URL-encoded scopes, metadata paths, and tarball paths.
- Reject path traversal, double encoding, `..`, null bytes, absolute URLs, and non-GET/HEAD methods.
- Do not forward the caller's `Authorization` header upstream. Do not forward arbitrary request headers.
- Use a dedicated upstream read credential with read-only access and the smallest possible scope.
- Apply per-user and per-tenant rate limits to proxy downloads.

**Signed URLs (optional)**

For clients that cannot send an Authorization header (for example a browser download), the catalog can issue a short-lived signed URL bound to package, version, tenant, user, and expiry. The proxy validates the signature and still checks current access before streaming. Bearer-based proxy downloads are the primary path; signed URLs are an optimization, not a second authorization model.

**Upstream isolation**

- The upstream npm repository must not be publicly reachable. If clients can reach it directly, they can bypass the proxy.
- If private and public packages share one registry, the proxy's server credential must be restricted to the paths needed for the packages it serves. A broad credential that can read every private tenant is not acceptable.
- A separate private npm repository per tenant, or per security domain, is the strongest isolation. It is optional for v1 but recommended for untrusted tenants.
- If the upstream registry is exposed, the only remaining protection is npm's own auth, which brings back the two-identity problem.

**Revocation and lifecycle**

- Revocation is immediate for the next proxy request because the proxy checks `skill_access` and grants on every download.
- A public package changed to private stops new anonymous downloads through the proxy. Already downloaded copies cannot be recalled.
- A yanked or retired version is rejected by the install-target resolver. The proxy should also reject a version that is no longer installable when it can resolve it.
- The proxy is on the critical path for private installs. Document the availability and failure behavior; public direct URLs can remain available if configured.

**Failure modes**

- Catalog proxy down: private installs fail with `503 artifact_proxy_unavailable`; public direct installs may continue.
- Missing/expired token: `401 unauthorized`.
- Cross-tenant request: `403 tenant_mismatch` (or `404` for metadata to avoid leaking existence).
- No install permission: `403 private_skill_forbidden`.
- Upstream registry unavailable: `502 artifact_upstream_unavailable`.
- Malformed package path: `400 invalid_artifact_path`.

**Configuration guardrails**

- A private skill must be published into a configured private artifact target. If it is not, the server rejects the publish with `private_artifact_scope_required`.
- The proxy must be reachable over TLS and must not be exposed without authentication for private paths.
- The proxy must not act as an open npm mirror. Only packages known to the catalog are proxied; unknown packages return `404`.
- The server's upstream read credential must be read-only and limited to the private artifact target.
- The CLI must not fall back to a public proxy URL or to npm credentials when a private proxy download fails.

### 7.7 Status codes

| Situation | Response |
| --- | --- |
| No credentials on a protected endpoint | `401 unauthorized` |
| Valid credentials, insufficient permission for a mutation | `403 forbidden` with an action and resource code |
| Private or tenant skill read by a caller without access | `404 skill_not_found` |
| Tenant-scoped request without an organization token | `403 tenant_required` |
| Organization token used for a different tenant | `403 tenant_mismatch` |
| Expired or invalid token | `401 invalid_token` |
| Namespace not owned or not verified | `403 namespace_forbidden` or `409 namespace_unverified` |
| Private publish without a configured private artifact target | `409 private_artifact_scope_required` |
| Cross-tenant grant attempt | `403 cross_tenant_forbidden` |
| Proxy download without permission | `403 private_skill_forbidden` |
| Proxy download for a different tenant | `403 tenant_mismatch` |
| Malformed npm artifact path | `400 invalid_artifact_path` |
| Proxy upstream unavailable | `502 artifact_upstream_unavailable` |
| Proxy unavailable | `503 artifact_proxy_unavailable` |
| Version or skill state conflict | `409` with a stable error code |

Do not include private skill names, owner names, or grant details in an unauthorized response.

## 8. Data model

The schema below is illustrative. It follows the existing SQLite/PostgreSQL dual-dialect style and the existing `TEXT` timestamp convention.

### 8.1 Identity and organization projections

```sql
CREATE TABLE users (
  id                 TEXT PRIMARY KEY,
  issuer             TEXT NOT NULL,
  subject            TEXT NOT NULL,
  display_name       TEXT NOT NULL DEFAULT '',
  email              TEXT NOT NULL DEFAULT '',
  avatar_url         TEXT NOT NULL DEFAULT '',
  status             TEXT NOT NULL DEFAULT 'active',
  created_at         TEXT NOT NULL,
  updated_at         TEXT NOT NULL,
  UNIQUE (issuer, subject)
);

CREATE TABLE organizations (
  id                 TEXT PRIMARY KEY,          -- local UUID
  logto_org_id       TEXT NOT NULL UNIQUE,      -- external ID
  slug               TEXT NOT NULL UNIQUE,
  name               TEXT NOT NULL,
  status             TEXT NOT NULL DEFAULT 'active',
  synced_at          TEXT NOT NULL,
  created_at         TEXT NOT NULL,
  updated_at         TEXT NOT NULL
);

CREATE TABLE organization_members (
  organization_id    TEXT NOT NULL,
  user_id            TEXT NOT NULL,
  roles_json         TEXT NOT NULL DEFAULT '[]',
  synced_at          TEXT NOT NULL,
  PRIMARY KEY (organization_id, user_id)
);
```

`organization_members` is a read model for tenant administration, audit filtering, notifications, and consistency checks. The token's `organization_id` and scopes are authoritative for the active request context. The projection must never be used to let a request from tenant A read tenant B data; cross-tenant grants are disabled.

### 8.2 Ownership and visibility

Keep ACL state out of the `skills` aggregate row:

```sql
CREATE TABLE skill_access (
  package_name       TEXT PRIMARY KEY,
  organization_id    TEXT,                       -- owning tenant; NULL for platform/personal
  owner_type         TEXT NOT NULL,              -- organization | user | platform
  owner_id           TEXT,
  visibility         TEXT NOT NULL DEFAULT 'private', -- public | organization | private
  access_revision    INTEGER NOT NULL DEFAULT 1,
  legacy_unowned     INTEGER NOT NULL DEFAULT 0,
  created_by         TEXT,
  created_at         TEXT NOT NULL,
  updated_at         TEXT NOT NULL,
  CHECK (
    (owner_type = 'organization' AND organization_id IS NOT NULL AND owner_id IS NOT NULL)
    OR (owner_type = 'user' AND organization_id IS NULL AND visibility = 'private')
    OR (owner_type = 'platform' AND organization_id IS NULL)
  )
);

CREATE TABLE skill_maintainers (
  package_name       TEXT NOT NULL,
  organization_id    TEXT NOT NULL,
  user_id            TEXT NOT NULL,
  granted_by         TEXT NOT NULL,
  created_at         TEXT NOT NULL,
  PRIMARY KEY (package_name, user_id)
);

CREATE TABLE skill_grants (
  id                 TEXT PRIMARY KEY,
  organization_id    TEXT NOT NULL,              -- owning tenant; no cross-tenant grants
  package_name       TEXT NOT NULL,
  subject_type       TEXT NOT NULL,              -- user (team reserved)
  subject_id         TEXT NOT NULL,
  permission         TEXT NOT NULL,              -- view | install | publish | manage
  granted_by         TEXT NOT NULL,
  expires_at         TEXT,
  created_at         TEXT NOT NULL,
  UNIQUE (organization_id, package_name, subject_type, subject_id)
);

CREATE INDEX idx_skill_access_org ON skill_access(organization_id);
CREATE INDEX idx_skill_grants_package ON skill_grants(organization_id, package_name);
CREATE INDEX idx_skill_grants_subject ON skill_grants(organization_id, subject_type, subject_id);
```

On the first successful release for a new package, the write path inserts a `skill_access` row **in the same transaction as the release record**, using the publisher's active tenant (or `platform` while enforcement is off). A user-owned row is only valid for a private draft. If the access row cannot be written, the publish fails; it must not fall back to a default visibility.

Existing packages are backfilled as `platform`/`public` with a `legacy_unowned` marker. After the migration window, a missing access row for a package created later is treated as `private` and denied, never as public. The marker is removed once every legacy package has an explicit owner decision.

The `CHECK` constraint is illustrative; SQLite and PostgreSQL enforce it differently. The domain layer must reject an invalid combination before writing.

Why a separate `skill_access` table instead of columns on `skills`:

- `skills` is updated by the aggregate write path. ACL changes should not force a release revision bump.
- A visibility change should not race with a version publish.
- Read queries can join a small table without loading the full aggregate.
- Legacy skills can be backfilled without touching release data.

### 8.3 Namespaces

```sql
CREATE TABLE package_namespaces (
  id                 TEXT PRIMARY KEY,
  prefix             TEXT NOT NULL UNIQUE,       -- e.g. @acme.skill/
  organization_id    TEXT NOT NULL,              -- owning tenant
  status             TEXT NOT NULL,              -- pending | verified | rejected
  verification       TEXT NOT NULL DEFAULT 'admin', -- admin in v1
  verified_at        TEXT,
  created_by         TEXT NOT NULL,
  created_at         TEXT NOT NULL,
  updated_at         TEXT NOT NULL
);

CREATE INDEX idx_package_namespaces_org ON package_namespaces(organization_id);
```

Namespace matching is longest-prefix-first. A prefix must end with `/` and must start with `@`. A prefix is globally unique; it belongs to exactly one tenant. The reserved `@draft.` prefix is never stored as a tenant namespace.

A tenant also needs a private artifact target for organization/private skills:

```sql
CREATE TABLE tenant_artifact_configs (
  organization_id          TEXT PRIMARY KEY,
  upstream_npm_registry_url TEXT NOT NULL,      -- server-side only; never returned to clients
  private_scope_prefix      TEXT NOT NULL,      -- e.g. @acme.skill/
  proxy_mount_path          TEXT NOT NULL DEFAULT '/repository/skills-npm',
  auth_mode                 TEXT NOT NULL DEFAULT 'catalog_proxy',
  created_at                TEXT NOT NULL,
  updated_at                TEXT NOT NULL
);
```

The publish path rejects a non-public skill when this row is missing or the package is outside the configured private scope. The install-target path uses the row to build a **catalog proxy** URL, not the upstream URL. The upstream URL and its credential stay server-side. The proxy maps the public proxy path to the tenant's upstream registry after authorization.

### 8.4 Tags

```sql
CREATE TABLE tags (
  id                 TEXT PRIMARY KEY,
  slug               TEXT NOT NULL,
  label              TEXT NOT NULL,
  scope              TEXT NOT NULL,              -- system | tenant
  organization_id    TEXT NOT NULL DEFAULT '',   -- tenant ID; empty for system tags
  visibility         TEXT NOT NULL DEFAULT 'private', -- public | private
  created_by         TEXT,
  created_at         TEXT NOT NULL,
  updated_at         TEXT NOT NULL,
  UNIQUE (scope, organization_id, slug)
);

CREATE TABLE skill_tags (
  organization_id    TEXT NOT NULL,              -- owning tenant; empty for platform/personal
  package_name       TEXT NOT NULL,
  tag_id             TEXT NOT NULL,
  assigned_by        TEXT NOT NULL,
  assigned_at        TEXT NOT NULL,
  PRIMARY KEY (organization_id, package_name, tag_id)
);

CREATE INDEX idx_skill_tags_tag ON skill_tags(tag_id);
```

For personal favorite labels, use separate tables so catalog tags cannot be polluted by personal names:

```sql
CREATE TABLE favorite_labels (
  id                 TEXT PRIMARY KEY,
  user_id            TEXT NOT NULL,
  name               TEXT NOT NULL,
  color              TEXT NOT NULL DEFAULT '',
  created_at         TEXT NOT NULL,
  UNIQUE (user_id, name)
);

CREATE TABLE favorite_label_assignments (
  user_id            TEXT NOT NULL,
  favorite_id        TEXT NOT NULL,
  label_id           TEXT NOT NULL,
  PRIMARY KEY (favorite_id, label_id)
);
```

Favorite labels can ship after the core favorites feature. They are listed here to keep the three tag concepts distinct.

### 8.5 Favorites

Use a normalized reference key to make upserts and uniqueness simple:

```sql
CREATE TABLE favorites (
  id                 TEXT PRIMARY KEY,
  user_id            TEXT NOT NULL,
  context_type       TEXT NOT NULL,              -- global | tenant | personal
  organization_id    TEXT,                       -- tenant for context_type=tenant; NULL otherwise
  ref_key            TEXT NOT NULL,              -- catalog:<package> | external:<provider>:<source>:<skillId>
  ref_type           TEXT NOT NULL,              -- catalog | external
  package_name       TEXT,
  external_provider  TEXT,
  external_source    TEXT,
  external_skill_id  TEXT,
  snapshot_json      TEXT NOT NULL DEFAULT '{}',
  created_at         TEXT NOT NULL,
  updated_at         TEXT NOT NULL,
  UNIQUE (user_id, ref_key),
  CHECK (
    (context_type = 'tenant' AND organization_id IS NOT NULL)
    OR (context_type IN ('global','personal') AND organization_id IS NULL)
  )
);

CREATE INDEX idx_favorites_user_context_created ON favorites(user_id, context_type, organization_id, created_at);
```

`snapshot_json` is display-only. Authorization always uses `package_name` for catalog skills and the external tuple for external skills. The server derives `context_type` from the skill's access record. A `tenant` favorite is only returned when the active tenant matches `organization_id`; a `global` favorite is returned in any context; a `personal` favorite is only returned in the personal workspace.

### 8.6 Audit extensions

```sql
ALTER TABLE audit_events ADD COLUMN actor_type TEXT NOT NULL DEFAULT 'legacy';
ALTER TABLE audit_events ADD COLUMN actor_user_id TEXT;
ALTER TABLE audit_events ADD COLUMN actor_subject TEXT;
ALTER TABLE audit_events ADD COLUMN actor_organization_id TEXT;
ALTER TABLE audit_events ADD COLUMN actor_display TEXT;
ALTER TABLE audit_events ADD COLUMN token_id TEXT;
ALTER TABLE audit_events ADD COLUMN client_id TEXT;
ALTER TABLE audit_events ADD COLUMN organization_id TEXT;

CREATE INDEX idx_audit_events_actor_user ON audit_events(actor_user_id, created_at);
CREATE INDEX idx_audit_events_actor_org ON audit_events(actor_organization_id, created_at);
CREATE INDEX idx_audit_events_org_created ON audit_events(organization_id, created_at);
```

Governance records:

```sql
ALTER TABLE governance_records ADD COLUMN changed_by_user_id TEXT;
ALTER TABLE governance_records ADD COLUMN changed_by_organization_id TEXT;
ALTER TABLE governance_records ADD COLUMN changed_by_display TEXT;
ALTER TABLE governance_records ADD COLUMN organization_id TEXT;
```

### 8.7 Migrations

The current schema is created with `CREATE TABLE IF NOT EXISTS` plus tolerant `ALTER TABLE` statements. This design adds enough tables and columns that a small versioned migration runner is warranted:

- `schema_migrations(version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`.
- Each migration is idempotent and runs inside a transaction where the dialect allows it.
- Backfill `skill_access` for every existing `skills` row with `owner_type='platform'`, `organization_id=NULL`, `visibility='public'`, `legacy_unowned=1`.
- After backfill, remove the fallback that treats a missing access row as public; a missing row denies.
- Backfill `audit_events.actor_type='legacy'`, copy `actor` to `actor_display`, and leave `organization_id` NULL until an administrator backfills tenant ownership.
- Create tenant projections from Logto on first sync; do not invent a tenant for an existing token.
- Do not rewrite existing release rows.
- Tenant ownership backfill is an explicit operator step: map legacy packages to a tenant or leave them platform-owned. There is no automatic guess from a package name.

## 9. API surface

All new routes are additive under `/api/v1`. Public registry routes keep their current names.

### 9.1 Identity

| Route | Purpose |
| --- | --- |
| `GET /api/v1/me` | Current user, active tenant, organizations, roles, scopes, and feature flags. |
| `GET /api/v1/me/orgs` | Organizations the user belongs to, with slug and role. The client uses this list to offer tenant switching. |
| `GET /api/v1/me/favorites` | Paginated favorites. |
| `POST /api/v1/me/favorites` | Upsert a favorite. |
| `DELETE /api/v1/me/favorites?refKey=...` | Remove a favorite. |
| `GET /api/v1/me/favorites/status?refKey=...&refKey=...` | Batch status for cards and drawers. |
| `POST /api/v1/me/favorites/merge` | Merge anonymous local favorites after login. |
| `GET /api/v1/me/favorites/export` | Export the caller's favorites (optional, later). |

`GET /api/v1/me` response example:

```json
{
  "user": {
    "id": "usr_01H...",
    "displayName": "Ada",
    "email": "ada@example.com"
  },
  "activeTenant": {
    "id": "org_01H...",
    "slug": "acme",
    "role": "admin"
  },
  "organizations": [
    { "id": "org_01H...", "logtoOrgId": "logto_org_01H...", "slug": "acme", "name": "Acme", "role": "admin" },
    { "id": "org_02H...", "logtoOrgId": "logto_org_02H...", "slug": "partner", "name": "Partner", "role": "member" }
  ],
  "personal": {
    "draftsEnabled": true
  },
  "scopes": ["catalog:read", "favorites:read", "favorites:write"],
  "features": {
    "favorites": true,
    "privateSkills": true,
    "tags": true,
    "crossTenantGrants": false
  }
}
```

`activeTenant` is derived from the organization token used for the request, not from a client preference. If the request has no organization token, `activeTenant` is `null` and only public/personal features are available.

### 9.2 Ownership and namespaces

| Route | Purpose |
| --- | --- |
| `GET /api/v1/orgs/:org/namespaces` | List namespace claims for the active tenant. `:org` must match the token's `organization_id`. |
| `POST /api/v1/orgs/:org/namespaces` | Claim a namespace for the active tenant. |
| `POST /api/v1/orgs/:org/namespaces/:id/verify` | Verify a claim. Platform administrator only in v1. |
| `PATCH /api/v1/skills/:package/ownership` | Transfer ownership within the same tenant. |
| `GET /api/v1/skills/:package/maintainers` | List maintainers in the owning tenant. |
| `POST /api/v1/skills/:package/maintainers` | Add a maintainer from the owning tenant. |
| `DELETE /api/v1/skills/:package/maintainers/:subject` | Remove a maintainer. |

The `:org` path segment is a routing convenience, not a trust source. The server always compares it with the verified token's `organization_id` and returns `403 tenant_mismatch` on a mismatch. Cross-tenant ownership transfer is not exposed.

### 9.3 Visibility and grants

| Route | Purpose |
| --- | --- |
| `PATCH /api/v1/skills/:package/visibility` | Change `public`, `organization`, or `private`. Public requires an organization owner and a verified namespace. |
| `GET /api/v1/skills/:package/grants` | List grants in the owning tenant. |
| `POST /api/v1/skills/:package/grants` | Create a grant for a user in the owning tenant. |
| `DELETE /api/v1/skills/:package/grants/:id` | Revoke a grant. |

Visibility changes and grant changes are audited with the human actor, the active tenant, and the resource tenant. Cross-tenant grants return `403 cross_tenant_forbidden` and are not silently dropped.

### 9.4 Tags

| Route | Purpose |
| --- | --- |
| `GET /api/v1/tags` | List system tags and tags for the active tenant. |
| `POST /api/v1/tags` | Create a tenant tag. |
| `PATCH /api/v1/tags/:id` | Rename or change visibility of a tenant tag. |
| `DELETE /api/v1/tags/:id` | Delete a tenant tag. |
| `PUT /api/v1/skills/:package/tags` | Replace the skill's tag set. Only tags visible to the active tenant are accepted. |
| `GET /api/v1/skills?tag=security&tag=mcp&tag_mode=all` | Filter by tags within the active tenant scope. |

### 9.5 Catalog responses

Add these fields to skill summaries and details when the feature is enabled:

```json
{
  "packageName": "@acme.skill/pdf",
  "visibility": "organization",
  "owner": {
    "type": "organization",
    "tenantId": "org_01H...",
    "slug": "acme",
    "displayName": "Acme"
  },
  "tags": [
    { "slug": "security", "label": "Security", "scope": "system" },
    { "slug": "team-payments", "label": "Payments", "scope": "tenant" }
  ],
  "permissions": {
    "canView": true,
    "canInstall": true,
    "canPublish": false,
    "canManage": false
  }
}
```

`permissions` is a convenience for the UI. The server still enforces every action on every request.

### 9.6 Install target

Add the visibility and transport decision:

```json
{
  "packageName": "@acme.skill/pdf",
  "version": "1.4.0",
  "tenantId": "org_01H...",
  "visibility": "private",
  "registryAccessMode": "proxy",
  "artifactAccess": "catalog_proxy",
  "tarballUrl": "https://catalog.example.com/repository/skills-npm/@acme%2Fpdf/-/pdf-1.4.0.tgz",
  "authorizationRequired": true
}
```

`registryAccessMode=proxy` and `artifactAccess=catalog_proxy` tell the CLI to send the catalog Bearer token when fetching the tarball. The proxy re-checks the token, tenant, visibility, and install permission; the install target alone is not a capability. The upstream npm URL and credential are never returned.

For a public skill, the catalog may return `registryAccessMode=proxy` (anonymous) or a public `direct` URL. Private and organization skills must always use the catalog proxy.

### 9.7 Error codes

Add stable codes:

| Code | Meaning |
| --- | --- |
| `invalid_token` | JWT signature, issuer, audience, or expiry failed. |
| `tenant_required` | The route requires an organization-scoped token, but none was supplied. |
| `tenant_mismatch` | Token tenant does not match the target tenant. |
| `cross_tenant_forbidden` | A grant, transfer, or share attempt crossed a tenant boundary. |
| `namespace_forbidden` | Publisher does not own the package namespace. |
| `namespace_unverified` | Namespace claim is not verified for public publishing. |
| `private_skill_forbidden` | Authenticated caller lacks access. |
| `grant_not_found` | Grant does not exist or is not visible. |
| `favorite_limit_reached` | Per-user favorite cap reached. |

## 10. End-to-end flows

### 10.1 Anonymous visitor favorites a skills.sh skill

1. The Registry page loads `/api/skills-sh/browse`.
2. The user clicks the star on a card.
3. The site writes `{refType: "external", provider: "skills.sh", source, skillId, snapshot}` to local storage.
4. The card and the Saved view update immediately.
5. No server call is made. The favorite survives reloads in the same browser.

### 10.2 User signs in and merges local favorites

1. The site redirects to Logto with PKCE and the `skill-dock-web` client ID.
2. Logto returns an authorization code; the SDK exchanges it for tokens.
3. The site calls `GET /api/v1/me` to learn the user, organizations, and feature flags.
4. If the user belongs to several organizations, the site asks which tenant to activate. If there is exactly one, it activates it automatically.
5. The site calls `POST /api/v1/me/favorites/merge` with the local references and the active tenant context.
6. The catalog accepts public and external references in any context, but only accepts tenant-private references that the user can access in the active tenant. It never imports a reference from another tenant.
7. The catalog deduplicates by `ref_key`, keeps the earliest `created_at`, and returns the merged list for the active context.
8. The site stores the merged list as its local mirror and clears the anonymous marker.
9. From then on, favorite writes go to the server; local storage is a cache. Switching tenants re-filters the visible set.

### 10.3 Tenant publisher publishes a public skill

1. The user selects the `acme` organization in the site or CLI.
2. The client obtains an organization token with `organization_id=org_acme` and `skill:publish`.
3. `POST /api/v1/skill-packaging/publish` includes the package name.
4. The catalog authenticates the JWT, verifies the organization claim, loads the namespace, and checks that `@acme.skill/` is verified and owned by `acme`.
5. The access module checks `ActionPublish` on the existing skill or on the new namespace.
6. For a non-public skill, the catalog verifies that the tenant has a configured private artifact target. If not, publishing fails with `private_artifact_scope_required`.
7. The release record and its `skill_access` row are written in one transaction.
8. Audit records the user ID, subject, organization ID, client ID, and service credential. The governance record (if any) stores the same human actor.

### 10.4 Member reads a private skill

1. The user signs in, selects the owning tenant, and obtains an organization token for it.
2. `GET /api/v1/skills/@acme.skill/pdf` carries the organization token.
3. The catalog validates the token and `organization_id`.
4. The access module sees `visibility=private` and checks the user grant or tenant membership.
5. The detail response includes `permissions.canView=true` and `canInstall=true`.
6. The install target response uses `registryAccessMode=proxy`, `artifactAccess=catalog_proxy`, and `authorizationRequired=true`.
7. The CLI downloads through the catalog proxy with the Skill Dock bearer token. The proxy re-checks tenant and install permission, then forwards to npm with the server's read credential.
8. The public cache is not used. The metadata response is `Cache-Control: private, no-store`; private bytes are streamed and not stored in a shared cache. The proxy records the download event.

### 10.5 Tenant admin grants a user access

1. The admin sends `POST /api/v1/skills/@acme.skill/pdf/grants` with `subjectType=user`, `subjectId=usr_partner`, `permission=install`.
2. The catalog verifies that the caller is an `owner` or `admin` of the owning tenant and that the subject is a member of that tenant.
3. The grant is written with `organization_id=org_acme` and audited.
4. The user sees the skill when their active tenant is `acme`. Their membership in another tenant does not grant access.
5. Revoking the grant removes access on the next request. No token reissue is needed for the grant itself.

Cross-tenant sharing is not supported in v1. A request to grant `org_partner` access to `acme`'s skill returns `403 cross_tenant_forbidden`. If this becomes a real requirement, model it as an explicit share that creates a read-only projection inside the recipient tenant, with its own audit and revocation path.

### 10.6 CLI login and publish

1. `skills login --registry https://catalog.example.com`.
2. The CLI starts the device flow and prints a code and verification URL.
3. The user approves in a browser.
4. The CLI stores the refresh token and calls `GET /api/v1/me`.
5. `skills org use acme` stores the active organization.
6. `skills publish` obtains or refreshes an organization token and calls the publish endpoint. A private publish requires a configured private artifact target.
7. `skills add` for a private skill receives a catalog proxy install target and sends the Skill Dock bearer token on download. It never sends an npm credential to the catalog.
8. The proxy records the download with the user, tenant, package, version, and client. Audit shows the user and the CLI client ID, not a shared publisher token.

### 10.7 Logto membership sync

1. Logto sends `Organization.Membership.Updated` to `/api/v1/webhooks/logto`.
2. The catalog verifies the webhook signature or shared secret.
3. It updates `organization_members` for the added/removed user IDs.
4. A periodic reconciliation job calls the Management API for each organization and repairs drift.
5. The projection is used for tenant admin views, audit filtering, notifications, and consistency checks. It never widens a request beyond the active tenant token.
6. A user removed from a tenant loses that tenant's data on the next token refresh. Cached responses for that tenant must not be served to a token that no longer carries it.

### 10.8 User switches active tenant

1. The site or CLI shows the organizations from `GET /api/v1/me/orgs`.
2. The user chooses `partner`.
3. The client obtains a new organization token with `organization_id=org_partner`.
4. Favorites, private skills, tags, and audit views re-filter to `partner`.
5. The previous tenant's private favorites and skills disappear from the view. Public skills remain visible.
6. No server-side session is changed. The old token remains valid for its short lifetime, but the client stops using it.

### 10.9 User promotes a personal draft to a tenant

1. The user works on a private draft under `@draft.<user-id>/<name>`.
2. The user chooses a tenant where they have `publisher` or higher and a verified namespace.
3. The client starts a promotion workflow: copy the draft source, validate it, and publish it as a new package under the tenant namespace.
4. The new package gets a new package identity and its first release. The draft is not renamed and does not become the tenant package.
5. The draft is archived or deleted according to the user's choice. Audit records the promotion, the source draft, and the actor.
6. The tenant package is now owned by the tenant and governed by its roles.

## 11. Deployment and configuration

### 11.1 Topologies

| Topology | Identity flow | Notes |
| --- | --- | --- |
| Self-hosted, single origin | Site and catalog behind one reverse proxy; Logto self-hosted or cloud | Simplest. No cross-origin CORS. Cookies are possible later. |
| Cloudflare site, remote catalog | Site on Workers, catalog on a separate domain, Logto cloud or self-hosted | The site uses Bearer tokens. The catalog must allow the site origin in CORS. Public skills.sh browsing still works without a catalog. |
| Air-gapped enterprise | Self-hosted Logto and catalog, no public internet | JWKS and OIDC discovery must be reachable internally. Cache keys and document the outage behavior. |
| Public directory only | No catalog configured | The site keeps local-only favorites and hides account-only features. |

Authenticated catalog calls need a reachable base URL. The site can use a build-time `VITE_CATALOG_API_URL`, a runtime Worker variable, or a Worker proxy such as `/api/catalog/*`. Do not assume the skills.sh origin is the catalog. Public directory browsing continues to use `/api/skills-sh/*`.

The public site must degrade cleanly. If the catalog capabilities route (`/api/v1/public/registry/capabilities` or the `/api/skills-sh/capabilities` alias) is unavailable or reports no catalog, the site should not show a broken sign-in button or attempt server favorites.

**Multi-tenant deployments** create one Logto organization per tenant. There is no global "current tenant" setting and no tenant cookie. The site and CLI show an organization switcher; the selected organization determines which organization token is used. A user who belongs to no organization can still browse public skills and work on personal drafts. Public skills are visible across tenants by design; tenant-owned private data is not.

### 11.2 Configuration

| Variable | Purpose | Default |
| --- | --- | --- |
| `SKILL_REGISTRY_AUTH_MODE` | `legacy`, `hybrid`, or `logto` | `legacy` |
| `SKILL_REGISTRY_LOGTO_ENDPOINT` | Logto base URL, without `/oidc` | unset |
| `SKILL_REGISTRY_LOGTO_ISSUER` | Expected issuer; discovery is used when unset | unset |
| `SKILL_REGISTRY_LOGTO_API_RESOURCE` | API resource identifier | unset |
| `SKILL_REGISTRY_LOGTO_WEB_CLIENT_ID` | SPA client ID | unset |
| `SKILL_REGISTRY_LOGTO_CLI_CLIENT_ID` | Device-flow client ID | unset |
| `SKILL_REGISTRY_LOGTO_M2M_CLIENT_ID` | Optional M2M client ID | unset |
| `SKILL_REGISTRY_LOGTO_M2M_CLIENT_SECRET` | Optional M2M secret | unset |
| `SKILL_REGISTRY_CORS_ORIGINS` | Comma-separated exact site origins | unset |
| `SKILL_REGISTRY_VISIBILITY_ENABLED` | Enable private/organization visibility | `false` |
| `SKILL_REGISTRY_NAMESPACE_ENFORCEMENT` | `off`, `warn`, or `on` | `off` |
| `SKILL_REGISTRY_TAGS_ENABLED` | Enable catalog tags | `false` |
| `SKILL_REGISTRY_FAVORITES_ENABLED` | Enable server favorites | `false` |
| `SKILL_REGISTRY_LEGACY_TOKEN_FULL_ACCESS` | Let legacy tokens see all skills when visibility is on | `false` |
| `SKILL_REGISTRY_PRIVATE_ARTIFACT_UPSTREAM_URL` | Server-side upstream npm URL for private packages; never returned to clients | unset |
| `SKILL_REGISTRY_PRIVATE_ARTIFACT_AUTH` | Artifact authorization mode for private skills | `catalog_proxy` |
| `SKILL_REGISTRY_ARTIFACT_PROXY_SIGNING_KEY` | Optional signing key for short-lived signed download URLs | unset |
| `SKILL_REGISTRY_ARTIFACT_PROXY_MAX_IN_FLIGHT` | Maximum concurrent private proxy downloads | `32` |
| `SKILL_REGISTRY_LOGTO_WEBHOOK_SECRET` | Verify Logto webhooks | unset |

The site needs its own public settings: `VITE_CATALOG_API_URL`, `VITE_LOGTO_ENDPOINT`, `VITE_LOGTO_WEB_CLIENT_ID`, and `VITE_LOGTO_API_RESOURCE`. These are public client values, not secrets. They may also be injected by the Worker at runtime if the deployment does not want to rebuild for each environment.

Tenant isolation is not a separate toggle once `SKILL_REGISTRY_VISIBILITY_ENABLED=true`. It is enforced by the access module and by every tenant-scoped repository method. The only global/data-crossing roles are `platform-admin` and `platform-auditor`, and their actions are audited.

`SKILL_REGISTRY_PRIVATE_ARTIFACT_UPSTREAM_URL` is a default for deployments that use one private registry. A multi-tenant deployment can map each tenant to its own private registry or scope in `tenant_artifact_configs`. If no private artifact target is configured for a tenant, publishing an `organization` or `private` skill fails with `private_artifact_scope_required` instead of silently using the public proxy.

The proxy needs a read-only upstream credential and network access to the private registry. The upstream URL is server-side configuration; clients only see the catalog proxy URL.

The defaults keep existing deployments unchanged. New behavior is opt-in until the data migration and UI are ready.

### 11.3 Logto setup checklist

1. Create the API resource with the scopes in section 5.3.
2. Create the `skill-dock-web` SPA app. Add the site redirect URI (`https://<site>/callback`) and CORS origin. Enable refresh-token rotation.
3. Create the `skill-dock-cli` Native app with device flow.
4. Create the `skill-dock-cicd` M2M app for CI, if needed.
5. Create the organization template roles `owner`, `admin`, `publisher`, `member`, and `viewer`; attach the scopes.
6. Create one Logto organization per tenant, or let tenant owners create them through the Logto flow. Do not use one shared organization as a catch-all tenant.
7. Create the `platform-admin` and `platform-auditor` global roles if the deployment needs break-glass access.
8. Configure webhooks for user and organization membership events.
9. Configure custom access-token claims only if display name/email are needed without an extra lookup. Do not put authorization decisions in custom claims.
10. Set the catalog environment variables and restart.
11. Configure a private artifact target per tenant for organization/private skills, plus a read-only server-side upstream credential.
12. Ensure the upstream private registry is not publicly reachable. The catalog proxy must be the only path to private bytes.
13. Verify with `GET /api/v1/me`, a tenant switch, and a test publish.
14. Verify that tenant A's organization token cannot read or download tenant B's private skill by package name or proxy URL.
15. Verify that a private install succeeds with a valid Skill Dock token and fails without one. Verify that revocation takes effect on the next proxy request.
16. Verify that public installs still work anonymously and that the proxy does not leak the upstream private URL.

### 11.4 CORS

- Public registry routes may keep `Access-Control-Allow-Origin: *` because they are anonymous and read-only.
- Authenticated routes must use an exact allow-list from `SKILL_REGISTRY_CORS_ORIGINS`.
- Allow `Authorization` and `Content-Type` headers. Do not allow credentials unless a cookie-based BFF is added.
- Do not reflect arbitrary `Origin` values.

### 11.5 CLI credentials

- Prefer the OS keychain.
- Fall back to a `0600` file under the XDG config directory.
- Never print the refresh token.
- Provide `skills logout` and a clear error when the refresh token is revoked.
- Keep `SKILLS_REGISTRY_TOKEN` for CI; document that it is a service credential, not a user login.

## 12. Security and privacy

- **JWT validation:** signature, issuer, audience, expiry, algorithm allow-list, clock skew. Cache JWKS; refresh on unknown key ID.
- **ID tokens:** never accept an ID token as an API access token.
- **Tenant context:** use the token's `organization_id` claim, not a client-supplied header or query parameter.
- **Tenant isolation:** every tenant-owned repository method takes a tenant ID. Every list, search, detail, install, tag, audit, and favorite query filters on it. Add explicit IDOR tests for cross-tenant package names and IDs.
- **Namespace impersonation:** require verified namespace ownership before public publishing.
- **Personal drafts:** user-owned records are private, cannot be public, and use the reserved `@draft.<user-id>/` prefix. They never appear in tenant lists or public search.
- **Private metadata:** 404 for unauthorized reads; do not leak names, counts, or tenant IDs.
- **Artifact authorization:** the catalog proxy is the enforcement point for private bytes. It validates the token, tenant, package, version, visibility, and install permission on every request.
- **Proxy tenant check:** the token's `organization_id` must match the owning tenant for organization/private packages. A cross-tenant proxy request is denied even with a valid package URL.
- **Upstream isolation:** the upstream npm registry must not be publicly reachable. The server read credential must be read-only and scoped to the private artifact target. A broad credential that can read every tenant is a cross-tenant risk.
- **Proxy request safety:** only GET/HEAD; strict npm path parsing; reject traversal, double encoding, null bytes, and absolute URLs; do not forward caller headers or tokens upstream.
- **Revocation:** proxy checks `skill_access` on every download, so revoking a grant blocks the next request. Already downloaded copies cannot be recalled.
- **Cache isolation:** no shared cache for authenticated/tenant-private responses or private tarballs; include the tenant in every non-public cache key.
- **CSRF:** Bearer tokens avoid cookie CSRF, but the site must have a strict Content Security Policy and avoid token exfiltration through XSS.
- **CORS:** exact origins only for authenticated routes.
- **Token storage:** follow the Logto SPA SDK guidance; document refresh-token storage and rotation.
- **Audit PII:** store email/display name only when needed; define retention and redaction.
- **Account deletion:** when a Logto user is deleted, anonymize audit actor fields, transfer or retire owned resources, and delete favorites and personal labels. Keep legally required audit records in anonymized form.
- **Tenant deletion:** block while it owns skills, namespaces, or grants; require transfer or explicit retirement. Logto organization deletion is not enough to erase tenant data.
- **Rate limits:** add per-user and per-tenant limits for publish, grants, favorites, and tag writes. Keep the existing global in-flight limiter.
- **Abuse:** favorites and tags are low-risk; publishing and grants are high-risk and should be audited and rate-limited.
- **Secrets:** Logto client secrets and M2M secrets only in the server environment or secret manager. The SPA and CLI are public clients with no secret.

## 13. Migration and rollout

Each phase is independently deployable and reversible. Feature flags default off.

| Phase | Deliverables | Acceptance criteria | Rollback |
| --- | --- | --- | --- |
| 0. Schema and identity plumbing | Migration runner; identity tables; `skill_access` backfill; Logto JWT verifier; `/api/v1/me`; audit columns. `AUTH_MODE=legacy`. | Existing tests pass. A Logto user can call `/me` in a test environment. No existing route changes behavior. | Disable Logto config; old tokens still work. |
| 1. Identity rollout | Site sign-in/logout; CLI device flow; `whoami`; audit actor fields populated. `AUTH_MODE=hybrid`. | A browser user and a CLI user can authenticate. Audit shows `actor_user_id`. Legacy tokens still work. | Set `AUTH_MODE=legacy` and hide the UI. |
| 2. Ownership and namespaces | Namespace claims (admin approval only); organization-only public ownership; personal private drafts; ownership transfer within a tenant; `NAMESPACE_ENFORCEMENT=warn` then `on`. | A publisher from tenant A cannot publish to tenant B's namespace. Public skills require an organization owner and a verified namespace. A personal draft cannot become public in place. Existing platform-owned skills remain publishable by legacy tokens. | Set enforcement back to `warn`/`off`. |
| 3. Visibility and grants | `skill_access`, tenant-scoped user grants, visibility API, list/detail/install authorization, `tenant_artifact_configs`, authenticated private artifact proxy, cache changes, tenant context plumbing. `VISIBILITY_ENABLED=true`. | Anonymous public routes never return private skills. Tenant A cannot read or download tenant B's organization/private skill even by exact name or proxy URL. Private install targets use the catalog proxy with `authorizationRequired=true`. A private publish without a configured artifact target fails. A granted user can read/install; revocation blocks the next download. | Turn off visibility; default all skills to public. |
| 4. Tags | System tags, tenant tags, assignment, filters, response fields. | Public responses show only public tags. Tenant tags are invisible outside the tenant. Filters compose with visibility and tenant scope. | Disable tags; keep tables. |
| 5. Favorites | Server favorites, tenant-aware merge endpoint, site star/Saved view, local fallback, i18n, tests. | A signed-in user sees the same favorites on two browsers. An anonymous user keeps local favorites. Switching tenants hides tenant-private favorites from the other tenant. A revoked private skill becomes unavailable without leaking metadata. | Disable server favorites; the site falls back to local storage. |
| 6. Hardening | SCIM 2.0 Users/Groups, optional Logto webhooks, admin views, metrics, data export/deletion, rate limits, docs. | Membership changes propagate through SCIM or webhooks within the agreed window. Account deletion removes personal data. | Disable SCIM/webhooks; use the legacy token path. |

Legacy compatibility:

- `AUTH_MODE=legacy` preserves the current token behavior.
- `AUTH_MODE=hybrid` accepts both legacy tokens and Logto JWTs.
- `AUTH_MODE=logto` accepts Logto JWTs for user routes and only explicitly scoped service tokens for automation.
- `LEGACY_TOKEN_FULL_ACCESS=false` means a legacy reader token sees only public skills once visibility is enabled. If an operator needs the old global read behavior during migration, set it to `true` for a limited window and audit the decision.
- Existing publisher tokens keep publishing platform-owned skills while `NAMESPACE_ENFORCEMENT` is `off` or `warn`. Once `on`, they need a service principal assigned to a tenant namespace or an owner.
- A legacy token has no tenant. It cannot use tenant-scoped routes after tenant isolation is enabled. It can only publish platform-owned skills through the legacy path or be replaced by a named service principal bound to a tenant.

## 14. Testing

### Unit

- JWT validation: valid token, wrong issuer, wrong audience, expired, `nbf` in the future, unknown key ID, `alg: none`, ID token rejected.
- Tenant claim: missing, mismatched, valid, multi-org selection.
- Access matrix: every principal/action/visibility combination, including cross-tenant denial.
- Namespace longest-prefix matching, admin-only verification, overlap rejection, personal draft prefix rejection.
- Tag visibility and tenant filters.
- Favorite deduplication, tenant-aware merge, tenant switch filtering, revocation, unavailable private skill.
- Audit field propagation for user, service, anonymous, and platform-admin principals, including target tenant.
- Personal draft promotion creates a new package identity and does not mutate the draft's releases.

### Integration

- Fake OIDC server with rotating JWKS and signed tokens.
- SQLite and PostgreSQL migration and query tests.
- Public registry routes never return private skills, even with an Authorization header.
- Tenant A cannot list, search, read, install, tag, audit, or favorite tenant B's organization/private skill by exact package name or ID.
- Authenticated list/detail/install authorization.
- Private install targets use `registryAccessMode=proxy`, `artifactAccess=catalog_proxy`, and `authorizationRequired=true`; the upstream npm URL is never exposed.
- The proxy permits a valid tenant token with install permission and rejects missing, expired, cross-tenant, and no-permission tokens.
- The proxy rejects malformed paths, traversal, double encoding, non-GET methods, and requests for a different tenant.
- Revoking a grant blocks the next proxy download without an npm credential change.
- Private publish fails when the tenant has no private artifact target configured.
- Cache key isolation: two users or two tenants in the same role do not share a tenant-private response or private tarball.
- Governance and release audit rows contain the human actor, active tenant, and resource tenant.
- Proxy download events record package, version, tenant, actor, and status without storing raw tokens.

### CLI

- Device-flow polling with success, `authorization_pending`, `slow_down`, `expired_token`, and `access_denied`.
- Credential file permissions and logout.
- Organization selection, multi-org switching, and a clear error when no organization is active.
- Private install sends the Skill Dock bearer token to the catalog proxy; it does not send an npm credential.
- A missing or expired token produces `401`; a tenant mismatch produces `403 tenant_mismatch`; a revoked grant produces `403 private_skill_forbidden`.
- `SKILLS_REGISTRY_TOKEN` remains a service credential and does not silently become a user session.

### Site

- Logged-out local favorites.
- Login merge.
- Star toggle on cards and drawer.
- Saved view and tag filter.
- Tenant switcher and active-tenant display.
- Switching tenants hides tenant-private favorites from the other tenant.
- Unavailable private favorite.
- Personal draft workspace is separate from tenant workspaces.
- i18n for English and Simplified Chinese.
- No account UI when the catalog is not configured.

## 15. Risks and open questions

### Risks

1. **Proxy bypass.** The catalog proxy is only useful if the upstream npm registry is not publicly reachable. If clients can reach npm directly, they can bypass proxy authorization. Network isolation is part of the security boundary.
2. **Proxy tenant bug.** A proxy request that fails to check the token's tenant, or parses the wrong package name, can leak another tenant's private bytes. Treat the proxy as a security-critical path with adversarial tests.
3. **Tenant boundary bugs.** A missing `organization_id` predicate in one query is a cross-tenant data leak. Every tenant-owned repository method must require the tenant ID; do not rely on handler discipline.
4. **Cache leakage.** A shared cache keyed only by role would leak private metadata. Public-only caching is the safest first step; include the tenant in every non-public key.
5. **Logto availability.** Login fails if Logto is down. Public reads should continue. Cache JWKS and document the degraded mode.
6. **Organization token limitations.** Organization tokens require the refresh-token flow. Test the SPA and CLI refresh paths, including token rotation and revocation.
7. **Legacy token expectations.** A reader token that used to see everything may see less after visibility is enabled. Communicate the change and provide a temporary `LEGACY_TOKEN_FULL_ACCESS` escape hatch.
8. **Legacy ownership.** Existing skills have no owner. The operator must decide whether to assign them to a tenant or leave them platform-owned before enabling enforcement.
9. **Personal draft promotion.** Promoting a draft creates a new package identity. Decide how much source/history is copied and whether the draft is archived or deleted.
10. **Tag sprawl.** Limit tag creation and require tenant admin rights for shared tags.
11. **PII in audit.** Audit emails and display names only if needed; define retention and deletion behavior.
12. **Public site split.** The Cloudflare site and the self-hosted catalog are separate deployments. Account features must degrade cleanly when no catalog is configured.
13. **Proxy availability and capacity.** Private installs depend on the catalog proxy. A proxy outage blocks private installs even when npm is healthy. Define timeouts, concurrency limits, rate limits, and a clear `503` contract; consider multiple replicas.
14. **Signed URL leakage.** If short-lived signed URLs are enabled, a leaked URL is a bearer capability until it expires. Bind it to package, version, tenant, and user; keep the TTL short; still check current access on redemption.

### Remaining open questions

1. **Tag taxonomy:** free-form tenant tags or a curated set? Recommendation: system tags curated; tenant tags free-form with limits.
2. **Tag-based policy:** do we need "only skills tagged `security` may be installed" in v1? Recommendation: no. Explicit grants first.
3. **Webhooks vs polling:** are Logto webhooks available in the target deployment? Recommendation: webhooks plus periodic reconciliation; polling alone is a fallback.
4. **Service token migration:** when do legacy environment tokens become named service accounts? Recommendation: keep them for CI, add named service tokens with expiry and last-used tracking in phase 6.
5. **Favorite limits:** per-user cap (for example 5,000) and rate limit. Recommendation: a cap prevents accidental abuse; surface a clear error.
6. **Platform-admin break-glass:** should cross-tenant access require a second approver or a time-bound elevation? Recommendation: time-bound elevation with audit in phase 6.
7. **Tenant creation:** does an organization admin self-create a tenant in Logto, or does a Skill Dock operator provision it? Recommendation: Logto remains the source of truth; Skill Dock exposes a link to the Logto flow and never creates shadow tenants.
8. **Cross-tenant sharing:** if a real use case appears, should it be a read-only projection or an explicit export/import? Recommendation: projection with its own audit and revocation, not direct cross-tenant rows.
9. **Signed URLs vs Bearer only:** do browser downloads need short-lived signed URLs, or is every private download performed by the CLI with a Bearer token? Recommendation: Bearer first; add signed URLs only when a browser download surface exists.
10. **Private upstream topology:** one shared private registry with scoped read credentials, or one registry per tenant? Who rotates the proxy's upstream credential? Recommendation: one private registry with tenant-scoped paths and a read-only credential in v1; per-tenant registries for high-assurance deployments.

## 16. SCIM provisioning

Logto's public documentation (checked 2026-11-23) exposes OIDC, organizations, webhooks, and JIT provisioning, but not a native SCIM 2.0 server or client. To reduce the custom reconciliation work in Phase 6, Skill Dock implements a standards-based **SCIM 2.0 inbound provisioning** surface. A SCIM client, directory sync service, or future Logto SCIM integration can use it directly.

### 16.1 Endpoints

Enabled only when `SKILL_REGISTRY_SCIM_TOKEN` is set.

| Route | Purpose |
| --- | --- |
| `GET /scim/v2/ServiceProviderConfig` | SCIM feature discovery. |
| `GET /scim/v2/ResourceTypes` | User and Group resource types. |
| `GET/POST /scim/v2/Users` | List or create users. |
| `GET/PUT/PATCH/DELETE /scim/v2/Users/:id` | Read, replace, patch, or delete a user. |
| `GET/POST /scim/v2/Groups` | List or create groups. |
| `GET/PUT/PATCH/DELETE /scim/v2/Groups/:id` | Read, replace, patch, or delete a group. |

Authentication is a bearer token in `Authorization: Bearer <SKILL_REGISTRY_SCIM_TOKEN>`. SCIM traffic does not use a Logto JWT. The token should be long, random, rotated on a schedule, and restricted at the network boundary.

### 16.2 Mapping

- **SCIM User** → local `users` row with issuer `scim`, subject = `externalId` or `userName`.
- **SCIM Group** → local organization (tenant). The group's `externalId` or display name becomes the tenant slug. If an organization with that slug exists, it is reused; otherwise a `scim:` external organization ID is created.
- **Group members** → `organization_members` rows. Members are users previously created through SCIM.
- **First Logto sign-in** links a SCIM user by email: the local user ID and memberships are preserved, and the issuer/subject are replaced with the Logto values.

### 16.3 What SCIM replaces

- For deployments with an external directory sync, SCIM replaces the per-tenant custom reconciliation loop in Phase 6.
- Logto webhooks (`Organization.Membership.Updated`) remain useful when Logto itself is the source of membership changes.
- The two paths can coexist. SCIM group membership is additive in the current baseline; role assignment and authoritative member removal are follow-up hardening items.

### 16.4 Limitations

- Group members are added, not authoritatively removed, in the first SCIM implementation.
- SCIM roles are not mapped to Skill Dock organization roles yet.
- Filtering is limited to `userName eq` and `emails.value eq`.
- Bulk operations, ETag/versioning, and SCIM password change are not supported.
- SCIM groups create tenant organizations; they are not a second team model inside a tenant.

## 17. Implementation map

This table is a starting point, not a file-by-file commit plan. It identifies the seams that should change together.

| Area | Primary files | Change |
| --- | --- | --- |
| Identity | New `internal/identity` package; [`internal/platform/auth/authorizer.go`](../internal/platform/auth/authorizer.go); [`internal/platform/http/middleware/auth.go`](../internal/platform/http/middleware/auth.go); [`cmd/skill-registry/main.go`](../cmd/skill-registry/main.go) | Add Logto OIDC discovery/JWKS validation, composite authentication, user/tenant projections, active-tenant validation, and configuration. |
| Access policy | New `internal/access` package; [`internal/platform/http/access/workflow.go`](../internal/platform/http/access/workflow.go); catalog and governance handlers | Add the tenant-aware decider and `VisibleFilter`; replace scattered role checks for new routes. Every decision requires a tenant when the resource is tenant-owned. |
| Catalog state | [`internal/catalog/domain/types.go`](../internal/catalog/domain/types.go); [`internal/catalog/domain/repository.go`](../internal/catalog/domain/repository.go); [`internal/catalog/infra/sql_state_store.go`](../internal/catalog/infra/sql_state_store.go); [`internal/catalog/infra/memory_repository.go`](../internal/catalog/infra/memory_repository.go); [`internal/catalog/application/read_service.go`](../internal/catalog/application/read_service.go); [`internal/catalog/application/write_service.go`](../internal/catalog/application/write_service.go) | Add ownership, visibility, tenant-scoped tags, grants, personal drafts, and per-tenant private artifact configuration without coupling them to aggregate revisions. Tenant-owned repository methods take a tenant ID. |
| HTTP surface | [`internal/catalog/http/routes.go`](../internal/catalog/http/routes.go); [`internal/catalog/http/handlers.go`](../internal/catalog/http/handlers.go); [`internal/backend/httpserver/server.go`](../internal/backend/httpserver/server.go); [`internal/backend/httpserver/public_registry.go`](../internal/backend/httpserver/public_registry.go); [`internal/backend/httpserver/npm_registry_proxy.go`](../internal/backend/httpserver/npm_registry_proxy.go) | Add `/me`, ownership, grant, tag, and favorite routes; make public reads public-only; add authenticated private artifact proxy with tenant/path checks; keep public proxy anonymous; include the tenant in cache keys. |
| Audit | [`internal/platform/persistence/audit_store.go`](../internal/platform/persistence/audit_store.go); [`internal/platform/audit/recorder.go`](../internal/platform/audit/recorder.go); [`internal/governance/http/workflow.go`](../internal/governance/http/workflow.go) | Add actor identity fields, active tenant, and resource tenant; propagate the human actor through mutations. |
| SCIM | New `internal/scim` package; [`internal/backend/httpserver/scim.go`](../internal/backend/httpserver/scim.go) | Add SCIM 2.0 Users/Groups, SCIM bearer auth, tenant mapping, and membership projection. |
| Site | [`apps/site/src/components/AppShell.tsx`](../apps/site/src/components/AppShell.tsx); [`apps/site/src/pages/SkillsDirectoryPage.tsx`](../apps/site/src/pages/SkillsDirectoryPage.tsx); [`apps/site/src/components/ExternalSkillDrawer.tsx`](../apps/site/src/components/ExternalSkillDrawer.tsx); `apps/site/src/lib/content.en.ts`; `apps/site/src/lib/content.zh.ts`; new auth/favorites modules | Add sign-in, organization/tenant switcher, star toggles, Saved view, tenant-aware local merge, personal draft workspace, and bilingual copy. |
| CLI | [`apps/skills-cli/src/cli.ts`](../apps/skills-cli/src/cli.ts); [`apps/skills-cli/src/catalog.ts`](../apps/skills-cli/src/catalog.ts); [`apps/skills-cli/src/registry-bundle.ts`](../apps/skills-cli/src/registry-bundle.ts); new auth/credential modules | Add `login`, `logout`, `whoami`, `org`, token refresh, secure credential storage, and private proxy downloads with the Skill Dock bearer token. Never send an npm credential to the catalog. |
| Deployment | `.env.example`; `docker-compose.yml`; `zarf/charts/skill-dock`; `docs/01-server-architecture.md`; `docs/03-deployment.md` | Add Logto settings, CORS origins, feature flags, tenant provisioning, authenticated private artifact proxy, upstream read credentials, TLS/rate limits, and migration guidance. |

## 18. Appendix: Logto facts used in this design

These are the Logto behaviors this design relies on. They should be re-checked against the target Logto version during implementation.

- Logto supports OpenID Connect discovery, JWKS, JWT access tokens for API resources, and opaque tokens for UserInfo.
- API resources are identified by an absolute URI, and permissions are scopes in the token.
- Organization-level API tokens include the `organization_id` claim and an `aud` for the API resource. The scopes are filtered by the user's organization roles.
- A Logto organization is the tenant boundary for this design. The token's `organization_id` is the active tenant, not a user preference header.
- Organization tokens are obtained through the refresh-token flow or client credentials; the authorization code flow does not return an organization token directly.
- ID tokens can include `organizations` and `organization_roles` claims when the corresponding scopes are requested. These help the client show an organization switcher; they do not authorize API requests.
- Logto supports device flow for native/CLI apps. Device-flow apps are public clients and do not use a client secret.
- Logto webhooks include `Organization.Membership.Updated` with added/removed user IDs, and user/organization data mutation events.
- Logto organization roles are defined in a tenant-wide organization template. Role assignments are per organization.
- Custom access-token claims are available but should be limited to display data, not authorization.

References:

- <https://docs.logto.io/authorization/organization-level-api-resources>
- <https://docs.logto.io/authorization/global-api-resources>
- <https://docs.logto.io/authorization/validate-access-tokens>
- <https://docs.logto.io/quick-starts/device-flow>
- <https://docs.logto.io/developers/webhooks/webhooks-events>
- <https://docs.logto.io/organizations>
- <https://docs.logto.io/authorization/organization-template>

## 19. Definition of done for the first release

The first release is complete when all of the following are true:

1. A user can sign in through Logto from the site and authenticate the CLI through device flow.
2. `GET /api/v1/me` returns the user, organizations, active tenant, roles, and scopes.
3. A user with multiple organizations can switch active tenants, and the API receives a different `organization_id` claim.
4. Publishing through a verified tenant namespace records the user, tenant, and client in audit.
5. A legacy publisher token cannot publish into a tenant's verified namespace, and a tenant token cannot publish into another tenant's namespace.
6. An anonymous caller cannot read or install a private skill, and a public route never returns one.
7. Tenant A cannot list, search, read, install, tag, audit, or favorite tenant B's organization/private skill, even by exact package name.
8. A user grant within the owning tenant works, and revocation takes effect on the next request. Cross-tenant grants return `403 cross_tenant_forbidden`.
9. A private install uses the catalog proxy with `authorizationRequired=true`. The proxy checks the token, tenant, package, version, and install permission before forwarding to npm.
10. The upstream npm URL and credential are never exposed. The upstream registry is not publicly reachable.
11. Private publishing fails unless the tenant has a configured private artifact target. Revoking a grant blocks the next proxy download.
12. Public skills are organization-owned and require a verified namespace. A personal draft cannot be made public in place.
13. Public tags and tenant tags are separated, and a tag filter composes with visibility and tenant scope.
14. A signed-in user sees the same favorites in two browsers; an anonymous user keeps local favorites; switching tenants hides tenant-private favorites from the other tenant.
15. The migration can be rolled back by turning off the feature flags without losing release data.
