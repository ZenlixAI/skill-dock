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

If you need PDF export, install a Chromium runtime first:

```bash
pnpm add -D playwright-chromium
```

Then run:

```bash
pnpm run export
```

Once deck content is added, the presentation source should live in `slides.md`.
Shared presentation styling should live in `styles.css`.
Reusable diagram components should live in `components/`.
