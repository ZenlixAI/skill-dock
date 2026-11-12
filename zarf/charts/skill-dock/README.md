# Skill Dock Helm chart

This chart deploys Skill Dock on Kubernetes:

- the Catalog API and its npm proxy;
- the nginx-based web site;
- an embedded [Verdaccio](https://verdaccio.org/) npm registry;
- an embedded PostgreSQL database;
- optional external PostgreSQL and external npm registry;
- standard Kubernetes Ingress or Traefik `IngressRoute`.

The default install is self-contained: Catalog, the site, Verdaccio, and
PostgreSQL all run in the cluster. Use the external options when you already
operate those services.

## Requirements

- Kubernetes 1.25 or later
- Helm 3.12 or later
- A default StorageClass for the embedded PostgreSQL and Verdaccio volumes
- Container images available to the cluster

The default images are:

- `ghcr.io/<owner>/skill-dock-catalog`
- `ghcr.io/<owner>/skill-dock-site`
- `verdaccio/verdaccio:6.11.1`
- `postgres:17`

The repository's `Images` workflow publishes the Skill Dock catalog and site
images. Override the image values for private registries or mirrored images.
Air-gapped clusters should also mirror `verdaccio/verdaccio`, `postgres`, and the
`busybox` test image.

## Quick start

```bash
helm upgrade --install skill-dock ./zarf/charts/skill-dock \
  --namespace skill-dock --create-namespace
```

The default install uses embedded PostgreSQL and embedded Verdaccio. Ingress is
disabled by default, so use a port-forward for the first check:

```bash
kubectl -n skill-dock port-forward svc/skill-dock-site 8080:8080
# open http://127.0.0.1:8080
```

Get the Catalog tokens:

```bash
kubectl -n skill-dock get secret skill-dock-catalog-secrets \
  -o jsonpath='{.data.reader-token}' | base64 -d
```

Get the Verdaccio publisher credentials:

```bash
kubectl -n skill-dock get secret skill-dock-verdaccio-secrets \
  -o jsonpath='{.data.publisher-user}' | base64 -d
kubectl -n skill-dock get secret skill-dock-verdaccio-secrets \
  -o jsonpath='{.data.publisher-password}' | base64 -d
```

Get the embedded PostgreSQL password:

```bash
kubectl -n skill-dock get secret skill-dock-postgres-secrets \
  -o jsonpath='{.data.password}' | base64 -d
```

## Accessing the site and Catalog

Catalog is available inside the cluster at:

```text
http://skill-dock-catalog:8080
```

The site proxies `/api` and `/repository` to Catalog. For CLI clients, point
`SKILLS_REGISTRY_URL` at the site URL, or directly at the Catalog URL when you
expose Catalog separately.

### Ingress

```yaml
ingress:
  enabled: true
  type: ingress          # ingress | ingressRoute
  className: traefik
  site:
    host: skills.example.com
    tls:
      enabled: true
      secretName: skill-dock-tls
```

`ingress.site` exposes the site. `ingress.catalog` optionally exposes Catalog
directly. `ingress.verdaccio` optionally exposes the embedded registry. Ingress
is disabled by default.

For Traefik:

```yaml
ingress:
  enabled: true
  type: ingressRoute
  site:
    host: skills.example.com
    entrypoint: web
    tls:
      enabled: true
      secretName: skill-dock-tls
```

Per-ingress annotations override `ingress.annotations`.

Keep the Verdaccio ingress path at `/` unless you also configure Verdaccio's
`url_prefix` in `verdaccio.config` and set `artifactRepository.publicUrl` to the
resulting external URL.

## Embedded Verdaccio

Verdaccio is deployed when `artifactRepository.url` is empty and
`verdaccio.enabled=true` (the default). The chart:

- creates a hosted npm registry;
- creates a generated `publisher` account;
- allows anonymous reads by default (`verdaccio.enableAnonymousRead`);
- restricts publishes to authenticated users;
- stores package data on a PersistentVolume.

Catalog publishes to Verdaccio through the `publisher` account. The generated
password is stored in `skill-dock-verdaccio-secrets` and is reused on upgrade.
Set `verdaccio.auth.publisher.password` explicitly if you need to control it.

### Add an upstream npm source

By default the embedded registry is self-contained: it stores and serves the
packages published through Catalog and does not proxy an external npm registry.
To proxy an upstream registry, configure `verdaccio.uplinks`:

```yaml
verdaccio:
  uplinks:
    npmjs: https://registry.npmmirror.com
```

The default package rules use `verdaccio.proxyUplink` (default `npmjs`). If you
rename the uplink, set `proxyUplink` to the same key.

To remove the default `npmjs` key entirely (for example, in an air-gapped
cluster), set the map to `null` in your values file:

```yaml
verdaccio:
  uplinks: null
```

The registry still stores and serves packages published through Catalog.

### Advanced Verdaccio configuration

Set `verdaccio.config` to replace the generated `config.yaml` entirely:

```yaml
verdaccio:
  config: |
    storage: /verdaccio/storage
    listen: 0.0.0.0:4873
    auth:
      htpasswd:
        file: /verdaccio/conf/htpasswd
        max_users: -1
    packages:
      '@*/*':
        access: $all
        publish: $authenticated
      '**':
        access: $all
        publish: $authenticated
```

The htpasswd file is still mounted at `/verdaccio/conf/htpasswd`.

### Publishing directly to Verdaccio

Catalog is the normal publish path. For a direct npm publish, use the generated
publisher credentials:

```bash
npm config set registry http://127.0.0.1:4873/
npm login --registry http://127.0.0.1:4873/
npm publish --registry http://127.0.0.1:4873/
```

Use a port-forward to reach Verdaccio when its ingress is disabled:

```bash
kubectl -n skill-dock port-forward svc/skill-dock-verdaccio 4873:4873
```

## External npm registry

Set `artifactRepository.url` to use an existing npm-compatible registry instead
of Verdaccio. When this value is set, the embedded Verdaccio is not deployed,
even if `verdaccio.enabled=true`.

```yaml
artifactRepository:
  url: https://artifacts.example.com/repository/skills-npm/
  publicUrl: https://artifacts.example.com/repository/skills-npm/
  readAuthorization: Bearer <read-token>
  publishAuthorization: Bearer <publish-token>
```

`url` is used by Catalog pods. `publicUrl` is returned to clients and defaults to
`url`. For GitOps, put the authorization headers in an existing Secret:

```yaml
artifactRepository:
  url: https://artifacts.example.com/repository/skills-npm/
  publicUrl: https://artifacts.example.com/repository/skills-npm/
  existingSecret: skill-dock-artifacts
  existingSecretKeys:
    readAuthorization: read-authorization
    publishAuthorization: publish-authorization
```

The artifact repository must support the npm registry protocol. Catalog remains
the authority for release records; do not publish bundles directly to the
repository unless you also create the release record.

### Public install URLs

With `catalog.installTargetMode=proxy` (the default), install targets are served
through Catalog's `/repository/skills-npm` mount. Clients only need to reach the
site or Catalog URL. Set `catalog.installTargetMode=direct` when clients can
reach the artifact repository directly.

In embedded mode the public URL defaults to the site ingress, Catalog ingress, or
Verdaccio ingress, in that order. If no ingress is enabled, the public URL falls
back to the internal Service URL. Set `artifactRepository.publicUrl` to the URL
clients actually use when they fetch npm metadata or tarball URLs directly.

## PostgreSQL

PostgreSQL is embedded when:

- `postgres.enabled=true` (default), and
- `database.url` and `database.existingSecret` are empty.

The chart generates the database password on first install and stores it in
`skill-dock-postgres-secrets`. The password is reused on upgrade. Changing
`postgres.auth.password` after the database volume has been initialized does not
change the existing PostgreSQL password; update it inside PostgreSQL or recreate
the volume.

### External PostgreSQL

Set `database.url` to use an existing database. The embedded PostgreSQL is
skipped automatically.

```yaml
postgres:
  enabled: false
database:
  url: postgres://skill_registry:password@postgres.example.com:5432/skill_registry?sslmode=require
```

Or reference a Secret:

```yaml
postgres:
  enabled: false
database:
  existingSecret: skill-dock-database
  existingSecretKey: dsn
```

### SQLite

For a single-replica evaluation install without PostgreSQL:

```yaml
catalog:
  replicaCount: 1
  persistence:
    enabled: true
database:
  url: file:/var/lib/skill-registry/skill-registry.db
```

Do not share a SQLite file between multiple Catalog replicas.

## Persistence and backup

| Component | Default | Notes |
| --- | --- | --- |
| PostgreSQL | 8Gi StatefulSet volume when embedded | Back up with your database tooling |
| Verdaccio | 10Gi StatefulSet volume when embedded | Back up the registry storage |
| Catalog workspace | `emptyDir` | Request-scoped; enable `catalog.persistence` for retained or larger temporary storage |

PostgreSQL and the artifact repository are the authoritative state. The Catalog
workspace is not.

The Catalog container defaults to UID/GID 10001, matching the official image.
Override `catalog.securityContext` and `catalog.podSecurityContext` for custom
images.

Helm does not delete StatefulSet volume claim templates on upgrade or uninstall.
When you switch from embedded PostgreSQL or Verdaccio to an external service,
the old PVCs remain until you remove them manually.

## skills.sh discovery

skills.sh search is enabled by default (`catalog.skillsSh.enabled: true`). The
site's `/registry` page uses the Catalog's `/api/skills-sh/*` aliases. Set
`catalog.skillsSh.enabled: false` to remove the external directory entirely.

Restricted clusters can use an egress proxy or a reachable mirror:

```yaml
catalog:
  extraEnv:
    - name: HTTPS_PROXY
      value: http://proxy.example.com:7890
  skillsSh:
    url: https://skills.sh
```

The Catalog's Go HTTP client uses the standard `HTTP_PROXY`/`HTTPS_PROXY`
environment variables.

## Existing Secrets

For GitOps, the catalog can use an existing Secret:

```yaml
catalog:
  existingSecret: skill-dock-catalog
```

That Secret must contain these keys:

- `database-url`
- `reader-token`
- `authoring-token`
- `publisher-token`
- `artifact-read-url`
- `artifact-read-authorization`
- `artifact-publish-authorization`
- `public-install-url`
- `auth-tokens-json`

The site can use an existing Secret with `catalog-authorization-header`:

```yaml
site:
  existingSecret: skill-dock-site
```

When `catalog.existingSecret` is used together with embedded Verdaccio or
PostgreSQL, the existing Secret must contain values that match the credentials
generated in the component Secrets.

Generated credentials use Helm's `lookup` and are reused on `helm upgrade`.
GitOps renderers that render without cluster access should supply explicit
tokens/passwords, or use the `existingSecret` values above. Otherwise each render
can generate new credentials.

## Values reference

| Value | Default | Description |
| --- | --- | --- |
| `nameOverride` | `""` | Override the chart name |
| `fullnameOverride` | `""` | Override the full resource prefix |
| `namespace.create` | `false` | Create the release namespace |
| `imagePullSecrets` | `[]` | Pull secrets for all pods |
| `serviceAccount.create` | `false` | Create a ServiceAccount |
| `database.url` | `""` | External database DSN |
| `database.existingSecret` | `""` | Secret containing the DSN |
| `database.existingSecretKey` | `skill_registry_database_url` | DSN key in the external Secret |
| `postgres.enabled` | `true` | Deploy embedded PostgreSQL when no external DSN is set |
| `postgres.auth.database` | `skill_registry` | Database name |
| `postgres.auth.user` | `postgres` | Database user |
| `postgres.auth.password` | `""` | Password; generated when empty |
| `postgres.persistence.size` | `8Gi` | PostgreSQL volume size |
| `catalog.replicaCount` | `1` | Catalog replicas |
| `catalog.image.repository` | `ghcr.io/zenlixai/skill-dock-catalog` | Catalog image |
| `catalog.image.tag` | `latest` | Catalog image tag |
| `catalog.service.port` | `8080` | Catalog service port |
| `catalog.auth.*` | generated | Reader, authoring, and publisher tokens |
| `catalog.seedSampleData` | `false` | Seed sample catalog records |
| `catalog.installTargetMode` | `proxy` | `proxy` or `direct` tarball access |
| `catalog.skillsSh.enabled` | `true` | Enable skills.sh discovery |
| `catalog.sharedWorkspaceRoot` | `/var/lib/skill-registry/packaging-workspaces` | Packaging workspace root |
| `catalog.waitForDatabase.enabled` | `true` | Wait for embedded PostgreSQL before starting Catalog |
| `catalog.persistence.enabled` | `false` | Mount a persistent workspace |
| `site.enabled` | `true` | Deploy the web site |
| `site.replicaCount` | `1` | Site replicas |
| `site.image.repository` | `ghcr.io/zenlixai/skill-dock-site` | Site image |
| `site.service.port` | `8080` | Site service port |
| `site.catalogAuthorizationHeader` | `""` | Optional fixed Catalog read token |
| `artifactRepository.url` | `""` | External npm registry URL; empty selects embedded Verdaccio |
| `artifactRepository.publicUrl` | `""` | Public registry URL returned to clients |
| `artifactRepository.readAuthorization` | `""` | Read authorization header for external mode |
| `artifactRepository.publishAuthorization` | `""` | Publish authorization header for external mode |
| `artifactRepository.existingSecret` | `""` | Secret containing external authorization headers |
| `verdaccio.enabled` | `true` | Deploy embedded Verdaccio when no external registry URL is set |
| `verdaccio.image.tag` | `6.11.1` | Verdaccio image tag |
| `verdaccio.auth.publisher.user` | `publisher` | Catalog publish user |
| `verdaccio.auth.publisher.password` | `""` | Publisher password; generated when empty |
| `verdaccio.enableAnonymousRead` | `true` | Allow anonymous reads |
| `verdaccio.uplinks` | `{}` | Optional upstream npm registries |
| `verdaccio.proxyUplink` | `npmjs` | Uplink used by the default package rules |
| `verdaccio.web.enabled` | `false` | Enable the Verdaccio web UI |
| `verdaccio.config` | `""` | Raw `config.yaml` override |
| `verdaccio.persistence.size` | `10Gi` | Verdaccio volume size |
| `ingress.enabled` | `false` | Enable ingress resources |
| `ingress.type` | `ingress` | `ingress` or `ingressRoute` |
| `ingress.className` | `""` | Ingress class; empty uses the cluster default |
| `ingress.site.*` | example host | Site ingress |
| `ingress.catalog.*` | disabled | Direct Catalog ingress |
| `ingress.verdaccio.*` | disabled | Direct Verdaccio ingress |
| `tests.enabled` | `true` | Render the Helm test pod |

## Migrating from the legacy skill-registry chart

This chart intentionally uses new resource names and values. It is not an
in-place upgrade of the previous internal `skill-registry` chart.

- New resources are release-scoped, for example `skill-dock-catalog`,
  `skill-dock-site`, `skill-dock-verdaccio`, and `skill-dock-postgres`.
- Nexus is replaced by Verdaccio when using the embedded registry.
- To keep your existing Nexus, set `artifactRepository.url` to its npm
  repository URL. The chart will use the external Nexus and skip Verdaccio.
- To migrate package data to Verdaccio, republish packages through Catalog or
  copy the npm tarballs into the new registry.
- Reuse the existing PostgreSQL database by setting `database.url`; no data
  migration is needed if the schema is already current.

## Operational checks

```bash
helm lint ./zarf/charts/skill-dock
helm template skill-dock ./zarf/charts/skill-dock >/dev/null
helm test skill-dock -n skill-dock
```

The repository's `Images` workflow publishes the catalog and site images for the
default branch and version tags. Pin `catalog.image.tag` and `site.image.tag` to
immutable release tags in production.
