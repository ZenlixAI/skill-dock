# Skill Registry Product Intro Deck

## Run locally

```bash
pnpm install
pnpm run dev
```

## Build

```bash
pnpm run build
```

## Export PDF

`playwright-chromium` is already declared as a dev dependency for PDF export. Install dependencies, then run:

```bash
pnpm run export
```

## Verify

Expected commands:

- `pnpm run dev` starts a local Slidev server and renders a 9-slide deck.
- `pnpm run build` generates `dist/`.
- If `playwright-chromium` is installed and usable in the environment, `pnpm run export` generates a PDF in the project directory.

Recorded outcome in this execution environment:

- `build` was verified successfully with `pnpm run build`.
- `dev` is currently unverified and environment-blocked here because Slidev could not bind a local port.
- `export` remains unverified and dependency-gated.

Once deck content is added, the presentation source should live in `slides.md`.
Shared presentation styling should live in `styles.css`.
Reusable diagram components should live in `components/`.
