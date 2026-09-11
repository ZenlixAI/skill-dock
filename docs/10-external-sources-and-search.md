# External sources and discovery

The catalog is the default search domain. The only configured external search adapter is `skills.sh`.

```bash
skills find <query> --include-external
skills find <query> --source skills.sh
skills find <query> --source all
```

External results carry `sourceType: external`, `sourceKind: skills.sh`, origin data, and `installable: false`. They are descriptive; a package is installable through the registry only when the catalog server can resolve its hosted release.

The provider URL defaults to `https://skills.sh` and is configured with `SKILL_REGISTRY_SKILLS_SH_URL`. The server forwards the query, page size, and optional `registry`, `scope`, and `origin` filters to that provider.

The CLI also works directly with local and git sources:

```bash
skills add ./path
skills add owner/repo
skills add <git-url>
```

`import-local`, `import-git`, and `import-npm` submit collected source to the server for publication. Source installation and catalog installation remain separate workflows.
