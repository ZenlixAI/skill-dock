# Skill Registry Product Intro Deck Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a polished 9-slide Slidev deck that introduces `skill registry` to the internal product team in 10 minutes and keeps the narrative focused on product understanding.

**Architecture:** Create a self-contained Slidev project under the CLI app so the deck can be iterated, previewed, built, and optionally exported without mixing presentation assets into the CLI runtime. Keep the deck mostly Markdown-driven, add one stylesheet for product-facing visuals, and use a small number of Vue slide components only where diagrams would be clumsy in raw Markdown.

**Tech Stack:** Slidev, Vue 3 single-file components, Markdown, CSS, pnpm

---

### Task 1: Scaffold a Self-Contained Slidev Deck Project

**Files:**
- Create: `slides/skill-registry-product-intro/package.json`
- Create: `slides/skill-registry-product-intro/README.md`
- Create: `slides/skill-registry-product-intro/.gitignore`

- [ ] **Step 1: Create the deck package manifest**

Create `slides/skill-registry-product-intro/package.json` with a minimal local package that can run, build, and export the deck without touching the CLI package scripts:

```json
{
  "name": "skill-registry-product-intro-deck",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "slidev --open false",
    "build": "slidev build",
    "export": "slidev export"
  },
  "devDependencies": {
    "@slidev/cli": "^0.52.0",
    "@slidev/theme-default": "^0.25.0",
    "vue": "^3.5.0"
  }
}
```

- [ ] **Step 2: Document how to run the deck**

Create `slides/skill-registry-product-intro/README.md` so another engineer can open and export the deck without reading the plan:

````md
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

```bash
pnpm add -D playwright-chromium
pnpm run export
```

The deck source lives in `slides.md`.
Shared presentation styling lives in `styles.css`.
Reusable diagram components live in `components/`.
````

- [ ] **Step 3: Ignore generated Slidev output**

Create `slides/skill-registry-product-intro/.gitignore`:

```gitignore
node_modules
dist
.slidev
```

- [ ] **Step 4: Install dependencies**

Run: `pnpm install`

Expected: lockfile created or updated inside `slides/skill-registry-product-intro` and install completes without unresolved package errors.

- [ ] **Step 5: Commit**

```bash
git add slides/skill-registry-product-intro/package.json slides/skill-registry-product-intro/README.md slides/skill-registry-product-intro/.gitignore
git commit -m "chore: scaffold skill registry intro deck"
```

### Task 2: Add Shared Visual Building Blocks for the Deck

**Files:**
- Create: `slides/skill-registry-product-intro/styles.css`
- Create: `slides/skill-registry-product-intro/components/LifecycleFlow.vue`
- Create: `slides/skill-registry-product-intro/components/ProductLoop.vue`

- [ ] **Step 1: Add the deck stylesheet**

Create `slides/skill-registry-product-intro/styles.css` with a product-oriented visual system that gives the deck a distinct look without requiring a custom Slidev theme package:

```css
:root {
  --sr-ink: #14213d;
  --sr-accent: #fca311;
  --sr-accent-soft: #ffe8b6;
  --sr-surface: #f8fafc;
  --sr-surface-strong: #ffffff;
  --sr-border: rgba(20, 33, 61, 0.12);
  --sr-muted: #546277;
  --sr-success: #1f9d74;
}

.slidev-layout {
  background:
    radial-gradient(circle at top left, rgba(252, 163, 17, 0.18), transparent 28%),
    linear-gradient(180deg, #fffdf8 0%, #f7fafc 100%);
  color: var(--sr-ink);
}

h1,
h2,
h3 {
  color: var(--sr-ink);
  letter-spacing: -0.03em;
}

.eyebrow {
  color: var(--sr-accent);
  font-size: 0.85rem;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.hero-panel,
.content-card,
.role-card,
.summary-card {
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid var(--sr-border);
  border-radius: 24px;
  box-shadow: 0 20px 60px rgba(20, 33, 61, 0.08);
}

.hero-panel {
  padding: 1.4rem 1.6rem;
}

.content-grid {
  display: grid;
  gap: 0.9rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.content-card {
  padding: 1rem 1.1rem;
}

.role-strip {
  display: grid;
  gap: 0.9rem;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.role-card {
  padding: 1rem;
}

.pillars {
  display: grid;
  gap: 1rem;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.pillar {
  background: linear-gradient(180deg, #ffffff 0%, #fff6df 100%);
  border: 1px solid rgba(252, 163, 17, 0.28);
  border-radius: 24px;
  padding: 1rem;
}

.bullets {
  margin-top: 0.6rem;
  color: var(--sr-muted);
  font-size: 0.95rem;
}

.boundary-list {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.boundary-item {
  border-left: 4px solid var(--sr-accent);
  background: rgba(255, 255, 255, 0.88);
  border-radius: 18px;
  padding: 0.85rem 1rem;
}

.quote-band {
  font-size: 1.5rem;
  line-height: 1.4;
  font-weight: 600;
}

.summary-grid {
  display: grid;
  gap: 0.8rem;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.summary-card {
  padding: 1rem;
}
```

- [ ] **Step 2: Add the lifecycle diagram component**

Create `slides/skill-registry-product-intro/components/LifecycleFlow.vue` to render the end-to-end skill lifecycle slide without relying on Mermaid styling quirks:

```vue
<script setup lang="ts">
const steps = [
  'Create',
  'Inspect',
  'Publish',
  'Release',
  'Search',
  'Install',
  'Update',
  'Govern',
];
</script>

<template>
  <div class="lifecycle-flow">
    <div v-for="step in steps" :key="step" class="lifecycle-step">
      <span>{{ step }}</span>
    </div>
  </div>
</template>

<style scoped>
.lifecycle-flow {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.lifecycle-step {
  align-items: center;
  background: linear-gradient(180deg, #ffffff 0%, #fff5dc 100%);
  border: 1px solid rgba(252, 163, 17, 0.28);
  border-radius: 999px;
  color: #14213d;
  display: flex;
  font-size: 0.92rem;
  font-weight: 700;
  justify-content: center;
  min-height: 3.25rem;
  position: relative;
}

.lifecycle-step::after {
  color: rgba(20, 33, 61, 0.3);
  content: '→';
  position: absolute;
  right: -0.52rem;
  top: 50%;
  transform: translateY(-50%);
}

.lifecycle-step:nth-child(4)::after,
.lifecycle-step:last-child::after {
  content: '';
}
</style>
```

- [ ] **Step 3: Add the product surface relationship component**

Create `slides/skill-registry-product-intro/components/ProductLoop.vue` to visualize how `CLI`, `Catalog`, `Web`, and the `npm-compatible registry` work together:

```vue
<template>
  <div class="loop-shell">
    <div class="loop-center">Skill Registry</div>
    <div class="loop-item top">Web</div>
    <div class="loop-item left">CLI</div>
    <div class="loop-item right">Catalog</div>
    <div class="loop-item bottom">npm-compatible registry</div>
  </div>
</template>

<style scoped>
.loop-shell {
  display: grid;
  grid-template-areas:
    ". top ."
    "left center right"
    ". bottom .";
  grid-template-columns: 1fr 1.2fr 1fr;
  grid-template-rows: auto 1fr auto;
  gap: 1rem;
  min-height: 18rem;
  place-items: center;
}

.loop-center,
.loop-item {
  align-items: center;
  background: rgba(255, 255, 255, 0.94);
  border: 1px solid rgba(20, 33, 61, 0.12);
  border-radius: 24px;
  box-shadow: 0 18px 45px rgba(20, 33, 61, 0.08);
  color: #14213d;
  display: flex;
  font-weight: 700;
  justify-content: center;
  padding: 1rem 1.1rem;
  text-align: center;
}

.loop-center {
  background: linear-gradient(180deg, #fca311 0%, #ffbf52 100%);
  grid-area: center;
}

.top { grid-area: top; }
.left { grid-area: left; }
.right { grid-area: right; }
.bottom { grid-area: bottom; }
</style>
```

- [ ] **Step 4: Run a build-shaped smoke check**

Run: `pnpm run build`

Expected: the build may fail at this point because `slides.md` does not exist yet; confirm the error is specifically about missing entry slides rather than dependency resolution or Vue parse failures.

- [ ] **Step 5: Commit**

```bash
git add slides/skill-registry-product-intro/styles.css slides/skill-registry-product-intro/components/LifecycleFlow.vue slides/skill-registry-product-intro/components/ProductLoop.vue
git commit -m "feat: add deck visual building blocks"
```

### Task 3: Author the 9-Slide Product Narrative

**Files:**
- Create: `slides/skill-registry-product-intro/slides.md`
- Modify: `slides/skill-registry-product-intro/styles.css`

- [ ] **Step 1: Create deck headmatter and cover slide**

Create `slides/skill-registry-product-intro/slides.md` with the shared headmatter and the opening slide wired to the custom stylesheet:

```md
---
theme: default
title: Skill Registry 产品介绍
titleTemplate: '%s'
info: 面向 AI Agent 技能的标准化生产与分发基础设施
class: text-left
drawings:
  persist: false
transition: slide-left
mdc: true
layout: cover
---

<style src="./styles.css"></style>

<div class="eyebrow">Internal Product Introduction</div>
# Skill Registry 产品介绍

<div class="hero-panel">
  <div class="quote-band">面向 AI Agent 技能的标准化生产与分发基础设施</div>
  <p class="bullets">目标：让内部产品团队在 10 分钟内统一认知它是什么、解决什么问题、边界在哪里。</p>
</div>

<!--
开场先定性：这不是单一工具演示，而是对 skill registry 产品边界的统一说明。
-->
```

- [ ] **Step 2: Add the problem, definition, and persona slides**

Append slides 2 to 4 to `slides.md`:

```md
---
layout: default
---

<div class="eyebrow">Why Now</div>
## 为什么要有它

<div class="hero-panel">
  <p class="quote-band">今天的问题，不是 skill 能不能写出来，而是写出来之后，怎么被可靠地交付、复用和治理。</p>
</div>

<div class="content-grid">
  <div class="content-card">
    <h3>来源分散</h3>
    <p class="bullets">skills 散落在仓库、文档和个人环境里，缺少统一入口。</p>
  </div>
  <div class="content-card">
    <h3>交付不统一</h3>
    <p class="bullets">不同团队靠手工复制、脚本、私有约定分发技能。</p>
  </div>
  <div class="content-card">
    <h3>版本不可控</h3>
    <p class="bullets">谁发布、发布了什么版本、谁在用，追踪成本高。</p>
  </div>
  <div class="content-card">
    <h3>企业交付复杂</h3>
    <p class="bullets">内网、权限、审计、稳定发布都让 skill 规模化落地变难。</p>
  </div>
</div>

---
layout: default
---

<div class="eyebrow">Definition</div>
## 它到底是什么

<div class="hero-panel">
  <p class="quote-band">Skill Registry 不是单点工具，而是一条围绕 skill 的完整产品链路。</p>
  <p class="bullets">它把定义、校验、发布、搜索、安装、更新连接成一套标准化流程。</p>
</div>

<div class="pillars">
  <div class="pillar">
    <h3>标准化生产</h3>
    <p class="bullets">让 skill 有统一的包形态、校验和发布规则。</p>
  </div>
  <div class="pillar">
    <h3>标准化分发</h3>
    <p class="bullets">让 skill 可搜索、可安装、可更新，而不是靠口口相传。</p>
  </div>
  <div class="pillar">
    <h3>标准化治理</h3>
    <p class="bullets">让版本、权限、记录和可追溯性成为平台能力。</p>
  </div>
</div>

---
layout: default
---

<div class="eyebrow">Audience</div>
## 它服务谁

<div class="role-strip">
  <div class="role-card">
    <h3>Skill 作者</h3>
    <p class="bullets">创建、整理、打包和发布技能。</p>
  </div>
  <div class="role-card">
    <h3>平台运营方</h3>
    <p class="bullets">维护规则、权限、发布流程和分发入口。</p>
  </div>
  <div class="role-card">
    <h3>使用团队</h3>
    <p class="bullets">发现、安装、更新并在业务场景中使用 skill。</p>
  </div>
</div>
```

- [ ] **Step 3: Add the value, lifecycle, surfaces, boundary, and summary slides**

Append slides 5 to 9 to `slides.md`:

```md
---
layout: default
---

<div class="eyebrow">Value</div>
## 它解决的核心问题

<div class="pillars">
  <div class="pillar">
    <h3>标准化发布</h3>
    <p class="bullets">把 skill 从“能写”变成“能稳定发布”。</p>
  </div>
  <div class="pillar">
    <h3>可发现可安装</h3>
    <p class="bullets">让技能像软件包一样可以搜索、选择和安装。</p>
  </div>
  <div class="pillar">
    <h3>可更新可治理</h3>
    <p class="bullets">让版本、升级、移除和记录进入统一体系。</p>
  </div>
</div>

---
layout: default
---

<div class="eyebrow">Lifecycle</div>
## 能力全景

<div class="hero-panel">
  <p class="bullets">围绕 skill 全生命周期提供支撑，而不是只覆盖某一个操作点。</p>
</div>

<LifecycleFlow />

---
layout: default
---

<div class="eyebrow">Product Shape</div>
## 产品形态

<div class="content-grid" style="grid-template-columns: 1.1fr 0.9fr;">
  <div class="content-card">
    <ProductLoop />
  </div>
  <div class="content-card">
    <h3>四个关键表面</h3>
    <ul class="bullets">
      <li><strong>CLI</strong>：操作闭环入口</li>
      <li><strong>Catalog</strong>：元数据、校验、发布流程编排</li>
      <li><strong>npm-compatible registry</strong>：包存储与分发</li>
      <li><strong>Web</strong>：浏览、门户与面向人的入口</li>
    </ul>
  </div>
</div>

---
layout: default
---

<div class="eyebrow">Boundary</div>
## 边界与非目标

<div class="hero-panel">
  <p class="quote-band">它解决的是技能分发基础设施问题，不是所有 Agent 问题。</p>
</div>

<div class="boundary-list">
  <div class="boundary-item"><strong>不是</strong> 大模型平台</div>
  <div class="boundary-item"><strong>不是</strong> Agent 运行时本体</div>
  <div class="boundary-item"><strong>不是</strong> 通用工作流编排器</div>
  <div class="boundary-item"><strong>不是</strong> 所有知识资产的统一管理系统</div>
</div>

---
layout: center
class: text-center
---

<div class="eyebrow">Takeaway</div>
## 让 AI Agent 技能像软件包一样被管理

<div class="summary-grid">
  <div class="summary-card">
    <h3>它解决什么</h3>
    <p class="bullets">标准化 skill 交付，不只是一次性安装。</p>
  </div>
  <div class="summary-card">
    <h3>它覆盖什么</h3>
    <p class="bullets">从制作、发布到搜索、安装、更新和治理的全链路。</p>
  </div>
  <div class="summary-card">
    <h3>它不是什么</h3>
    <p class="bullets">不是模型平台，不是运行时，不是通用编排器。</p>
  </div>
</div>

<!--
结尾回收三点：它是什么、覆盖什么、边界是什么。
-->
```

- [ ] **Step 4: Tune stylesheet for any layout gaps discovered while authoring**

Modify `slides/skill-registry-product-intro/styles.css` only if the content needs spacing or density tweaks. Prefer small, targeted additions such as:

```css
ul.bullets {
  padding-left: 1.1rem;
}

.slidev-layout h2 {
  margin-bottom: 0.8rem;
}
```

- [ ] **Step 5: Run a real build verification**

Run: `pnpm run build`

Expected: PASS and a generated `dist/` directory under `slides/skill-registry-product-intro`.

- [ ] **Step 6: Commit**

```bash
git add slides/skill-registry-product-intro/slides.md slides/skill-registry-product-intro/styles.css
git commit -m "feat: author skill registry intro deck"
```

### Task 4: Final Verification and Delivery Notes

**Files:**
- Modify: `slides/skill-registry-product-intro/README.md`

- [ ] **Step 1: Add final verification and export notes**

Update `slides/skill-registry-product-intro/README.md` to include explicit verification expectations after the deck has been built once:

```md
## Verify

- `pnpm run dev` should open a local Slidev server and render a 9-slide deck.
- `pnpm run build` should generate `dist/`.
- If `playwright-chromium` is installed, `pnpm run export` should generate a PDF in the project directory.
```

- [ ] **Step 2: Run the deck in dev mode for a smoke check**

Run: `pnpm run dev`

Expected: Slidev starts successfully and prints a local URL, typically `http://localhost:3030`.

- [ ] **Step 3: Stop the dev server and optionally test PDF export**

If PDF output is required, run:

```bash
pnpm add -D playwright-chromium
pnpm run export
```

Expected: PASS with a generated PDF. If export is skipped, record that dev and build were verified and export remains dependency-gated.

- [ ] **Step 4: Commit**

```bash
git add slides/skill-registry-product-intro/README.md
git commit -m "docs: add deck verification notes"
```

## Self-Review

### Spec coverage

- Product-team audience and 10-minute scope: covered by Tasks 1 and 3 through the deck structure and concise slide copy.
- 9-slide narrative: fully implemented in Task 3.
- Product-first visual direction: covered by Task 2 and the style tuning step in Task 3.
- Lifecycle and product-surface visuals: covered by `LifecycleFlow.vue` and `ProductLoop.vue` in Task 2.
- Verification and optional export: covered by Task 4.

No spec sections are currently uncovered.

### Placeholder scan

- No `TODO`, `TBD`, or deferred implementation markers remain.
- Verification commands and expected outcomes are explicit.
- All created file paths are concrete.

### Type consistency

- `LifecycleFlow.vue` and `ProductLoop.vue` match the component tags used in `slides.md`.
- `styles.css` is referenced exactly once via `<style src="./styles.css"></style>`.
- The Slidev package scripts match the commands referenced throughout the plan.
