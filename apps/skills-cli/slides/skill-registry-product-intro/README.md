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

- `pnpm run dev` should start a local Slidev server and render a 9-slide deck.
- `pnpm run build` should generate `dist/`.
- If `playwright-chromium` is installed and usable in the environment, `pnpm run export` should generate a PDF in the project directory.

Once deck content is added, the presentation source should live in `slides.md`.
Shared presentation styling should live in `styles.css`.
Reusable diagram components should live in `components/`.
