# External sources and discovery

The catalog is the default search domain. The only configured external search adapter is `skills.sh`.

```bash
skills find <query> --include-external
skills find <query> --source skills.sh
skills find <query> --source all
```

External results carry `sourceType: external`, `sourceKind: skills.sh`, origin data, and `installable: false`. They are descriptive; a package is installable through the registry only when the catalog server can resolve its hosted release. The CLI can still install an external skill from its source repository with `skills add <owner/repo> --skill <name>`.

The provider URL defaults to `https://skills.sh` and is configured with `SKILL_REGISTRY_SKILLS_SH_URL`. Search forwards `q` and `limit`; the CLI's `registry`, `scope`, and `origin` filters are not part of the skills.sh search contract and are ignored by this adapter. The adapter uses the public skills.sh API:

- `GET /api/search?q=<query>&limit=<n>` for keyword search (`q` must contain at least 2 characters);
- `GET /api/skills/<view>/<page>` for browse views (`all-time`, `trending`, `hot`);
- `GET /<owner>/<repo>/<skill>` for the detail page used by the Registry drawer. The server parses the HTML and the RSC payload, extracts the description, installs, weekly history, install command, topics, summary, SKILL.md content, and related skills, and sanitizes the HTML before returning it.

Set `SKILLS_SH_ENABLED=false` on the server to disable the server-side provider. The server does not construct the adapter and the public capabilities endpoint reports `externalSearch.enabled: false`. The Cloudflare Worker has its own `SKILLS_SH_ENABLED` variable in `apps/site/wrangler.jsonc`; when false, the site's directory page renders a disabled state and the Worker returns 404 for data routes.

The shipped `/registry` page is a skills.sh-only directory served by a Cloudflare Worker. The Worker reverse-proxies `/api/skills-sh/*` to the public skills.sh API, parses and sanitizes detail pages on the edge, and caches responses. The Go catalog mounts the same `/api/skills-sh/capabilities`, `/search`, `/browse`, and `/skill` routes for self-hosted/Docker deployments, and also exposes the `/api/v1/public/registry/*` API described in `13-api-endpoint-spec.md`.

The CLI also works directly with local and git sources:

```bash
skills add ./path
skills add owner/repo
skills add <git-url>
```

`import-local`, `import-git`, and `import-npm` submit collected source to the server for publication. Source installation and catalog installation remain separate workflows.
