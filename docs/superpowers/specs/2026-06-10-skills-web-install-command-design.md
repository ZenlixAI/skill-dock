# Skills Web Install Command Design

**Date:** 2026-06-10

## Goal

Adjust the skills web install command display so the UI suggests the same installation flow users actually run in terminals: `npx --registry ... skills add ...`, instead of `npm install ...`.

## Scope

- Update the web command generator used by the install panel.
- Preserve version-specific installation semantics on the version detail page.
- Keep the existing registry fallback placeholder behavior when registry metadata is absent.
- Update frontend tests that assert the rendered and copied install command.

## Design

### Command shape

The install command shown in the web UI will change from:

```bash
npm install <package>@<version> --registry=<registry>
```

to:

```bash
npx --registry <registry> skills add <package> --version <version>
```

Including `--version <version>` is required because the install panel appears on a version detail page and should continue to represent "install this exact version" instead of implicitly resolving `latest`.

### Fallback behavior

If `npmRegistry` is empty, the UI will continue to display the localized placeholder `待补充 npm 仓库地址`, and the generated command will embed that placeholder in the `--registry` position so operators can see what metadata is missing.

### Non-goals

- No backend API changes.
- No changes to the CLI install workflow itself.
- No changes to install policy labels, registry metadata display, or governance flows.

## Validation

- `InstallCommandPanel` tests cover copied and rendered command text.
- `VersionDetailPage` tests cover the version-detail install panel rendering.
