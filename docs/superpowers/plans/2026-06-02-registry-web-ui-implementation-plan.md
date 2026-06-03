# Registry Web UI Implementation Plan

## Goal

在当前 `Registry Core MVP` 之上补一套首版 Web UI，覆盖：

- Catalog 浏览
- Skill / Version 详情
- Import & Package 工作流
- Governance / Audit 可视化

并明确下一步如何从 mock UI 接到真实 API。

## Constraints

- 当前仓库没有既有前端工程，需要新建独立 `apps/skills-web`
- 不改写现有 Go Catalog、packaging、CLI 主链
- 首版先用 typed mock client，后续再切换到真实 HTTP client
- 不在前端重复实现 normalization / governance 规则，只消费后端结果

## Deliverables

### 1. Web app scaffold

目录：

- `apps/skills-web/`

内容：

- Vite + React + TypeScript
- 基础路由
- 可运行样式系统
- typed mock API client

### 2. Screen skeletons

- `/`
  - `OverviewPage`
- `/catalog`
  - `CatalogPage`
- `/skills/:packageName`
  - `SkillDetailPage`
- `/skills/:packageName/versions/:version`
  - `VersionDetailPage`
- `/import`
  - `ImportPage`
- `/governance`
  - `GovernancePage`
- `/audit`
  - `AuditPage`

### 3. Shared components

- `AppShell`
- `SurfaceCard`
- `StatusPill`
- `SectionHeading`
- `WarningStack`
- `ImportReportCard`
- `TimelineList`

### 4. Typed API seam

- `src/api/types.ts`
- `src/api/mockClient.ts`

真实接入时只需要新增：

- `src/api/httpClient.ts`

并把页面层从 mock client 切换到真实 client。

## Page-by-page implementation

### OverviewPage

目标：

- 给第一次进入的用户一个总览入口

依赖数据：

- curated skills
- recent governance changes
- recent audit events

对应 API：

- `GET /api/v1/skills/curated`
- `GET /api/v1/governance-changes?pageSize=5`
- `GET /api/v1/audit-events?pageSize=5`

### CatalogPage

目标：

- 搜索 skill
- 看 package、description、sourceType、latestVersion、status

依赖数据：

- skill list

对应 API：

- `GET /api/v1/skills`

实现重点：

- 支持搜索输入
- 支持 sourceType / status filter
- 支持 cursor pagination UI 占位

### SkillDetailPage

目标：

- 展示 skill 基本信息和 versions 列表

依赖数据：

- skill detail
- version summaries
- latest install target preview

对应 API：

- `GET /api/v1/skills/{packageName}`
- `GET /api/v1/skills/{packageName}/versions`
- `GET /api/v1/skills/{packageName}/install-target`

### VersionDetailPage

目标：

- 展示具体版本的 registry/bundle/install 状态

依赖数据：

- version detail

对应 API：

- `GET /api/v1/skills/{packageName}/versions/{version}`

Publisher action placeholders：

- `Create release record`
- `Deprecate`
- `Yank`

### ImportPage

目标：

- 用 4-step wizard 表达 `prepare -> validate -> pack -> publish handoff`

依赖数据：

- prepare response
- validate response
- pack response

对应 API：

- `POST /api/v1/skill-packaging/prepare`
- `POST /api/v1/skill-packaging/validate`
- `POST /api/v1/skill-packaging/pack`

关键可视化：

- `warningDetails`
- `importReport`
- validate status
- components
- bundle metadata

### GovernancePage

目标：

- 展示 governance timeline

依赖数据：

- governance changes

对应 API：

- `GET /api/v1/governance-changes`

### AuditPage

目标：

- 展示操作审计

依赖数据：

- audit events

对应 API：

- `GET /api/v1/audit-events`

## Component granularity

### AppShell

负责：

- 顶部导航
- 全局背景
- 页面外壳

### StatusPill

负责统一展示：

- skill status
- version status
- install policy
- normalized/direct import

### WarningStack

输入：

- `WarningDetail[]`

输出：

- warning cards with title/message/suggested action

### ImportReportCard

输入：

- `ImportReport`

输出：

- structured summary of:
  - normalized
  - strict mode
  - generated paths
  - rewritten identity

### TimelineList

输入：

- governance or audit items

输出：

- vertical timeline / feed

## Data layer plan

### Phase 1

- use `mockClient`
- each page independently fetches mock async data

### Phase 2

- add `httpClient`
- map response schemas 1:1 to current backend contracts
- preserve page props / component contracts

### Phase 3

- add auth/session boundary
- gate publisher-only actions

## Suggested issue split

1. `apps/skills-web scaffold`
2. `shared UI system and shell`
3. `catalog + skill detail pages`
4. `version detail page`
5. `import wizard with warningDetails/importReport`
6. `governance + audit pages`
7. `mock client -> real API client integration`
8. `publisher auth gating and action submission`

## Validation

首版完成后至少验证：

- `npm run type-check`
- `npm run build`

后续接真实 API 后再补：

- route smoke tests
- API contract tests
- permission gating checks
