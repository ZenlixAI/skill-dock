# Agent Skill Registry Web UI 设计

## 文档定位

本文定义 `Registry Core MVP` 的首版 Web UI。

目标不是重新设计后端，而是把现有能力：

- `Catalog API`
- `Skill Packaging API`
- `warningDetails + importReport`
- `governance / audit`

组织成一套真正可供用户使用的 Web 产品界面。

配套实现见：

- `apps/skills-web/`

## 设计目标

- 让 `reader` 可以通过 Web 发现、理解、评估并复制安装信息
- 让 `publisher` 可以通过 Web 完成导入、校验、打包、发布和治理
- 让平台运营方可以通过 Web 查看治理状态、审计事件和导入结果
- 让前端只消费服务端契约，不在前端重复实现 normalization / governance 判断

## 非目标

- 不设计 agent runtime 内执行 UI
- 不设计本地安装目录、winner、scope 决策界面
- 不要求 V1 引入新的后端聚合能力才能上线
- 不把同步 packaging API 强行包装成复杂工作流引擎

## 约束

### 现有后端边界

- Catalog 真相源：
  - `GET /api/v1/skills`
  - `GET /api/v1/skills/curated`
  - `GET /api/v1/skills/{packageName}`
  - `GET /api/v1/skills/{packageName}/versions`
  - `GET /api/v1/skills/{packageName}/versions/{version}`
  - `GET /api/v1/skills/{packageName}/install-target`
  - `GET /api/v1/governance-changes`
  - `GET /api/v1/audit-events`
- 发布 / 治理写接口：
  - `PUT /api/v1/skills/{packageName}/versions/{version}/release-record`
  - `POST /api/v1/skills/{packageName}/governance-changes`
  - `POST /api/v1/skills/{packageName}/versions/{version}/governance-changes`
- 打包导入接口：
  - `POST /api/v1/skill-packaging/prepare`
  - `POST /api/v1/skill-packaging/validate`
  - `POST /api/v1/skill-packaging/pack`

### 现有权限边界

- `reader`
  - 读 Catalog / versions / governance / audit / install target
- `publisher`
  - 包含 reader 能力
  - 可执行 release-record / governance change / packaging

### 现有导入能力

打包响应已经返回：

- `warnings`
- `warningDetails`
- `importReport`

因此前端必须：

- 直接展示 `warningDetails`
- 用 `importReport` 呈现“这次导入被系统改写了什么”
- 不自己判断是否生成了 `package.json` 或 placeholder

## 用户与任务

### Reader

主要任务：

- 搜索 skill
- 看 skill 描述和最新版本状态
- 看某版本是否可安装
- 复制 install target / registry 信息

### Publisher

主要任务：

- 从 `git / local / platform_authoring` 导入
- 检查 warnings / import report
- validate / pack / publish
- 为已发布版本登记 release record
- 对 skill / version 做 retire / deprecate / yank

### Operator / Auditor

主要任务：

- 看 governance change 时间线
- 查 audit events
- 追溯某次发布 / 导入是怎么发生的

## 信息架构

首版建议统一为一个产品壳：

1. `Overview`
2. `Catalog`
3. `Import & Package`
4. `Releases`
5. `Governance`
6. `Audit`

其中真正必须优先交付的是：

- `Catalog`
- `Skill Detail`
- `Import & Package`
- `Release Review`

## 路由建议

### P0

- `/`
  - 轻量 overview
- `/catalog`
  - 搜索 / curated / filter
- `/skills/:packageName`
  - skill detail
- `/skills/:packageName/versions/:version`
  - version detail
- `/import`
  - 导入与打包工作台
- `/releases/review`
  - pack 完成后的发布确认页

### P1

- `/governance`
  - governance list / filter
- `/audit`
  - audit event list / filter

## 页面设计

### 1. Overview

定位：

- 给第一次进入系统的用户一个低认知负担入口
- 不做伪大屏，不依赖新增聚合 API

内容：

- `Curated skills` 快捷入口
- `Recent governance changes` 列表
- `Recent audit events` 列表
- `Start importing` 主 CTA

实现建议：

- `Curated` 用 `GET /api/v1/skills/curated`
- `Recent governance` 用 `GET /api/v1/governance-changes?pageSize=5`
- `Recent audit` 用 `GET /api/v1/audit-events?pageSize=5`

### 2. Catalog

定位：

- 主发现页

布局：

- 顶部搜索条
- 左侧过滤区
  - source type
  - skill status
  - version status
- 中央结果列表
- 右侧上下文预览卡

关键元素：

- `packageName`
- `displayName`
- `description`
- `sourceType`
- `latestVersion`
- `latestVersionStatus`
- `updatedAt`

交互：

- 默认页签：`All`
- 次页签：`Curated`
- 支持 cursor 翻页，不做页码

接口：

- `GET /api/v1/skills`
- `GET /api/v1/skills/curated`

### 3. Skill Detail

定位：

- 回答“这个 skill 是什么、现在是否活跃、有哪些版本、怎么安装”

布局：

- 顶部 hero
  - display name
  - package name
  - source type
  - status chip
- 中部双栏
  - 左：description、version list
  - 右：install target preview、governance summary

交互：

- 默认突出 latest published version
- 点击某版本进入 version detail
- 可直接复制 install target 示例

接口：

- `GET /api/v1/skills/{packageName}`
- `GET /api/v1/skills/{packageName}/versions`
- `GET /api/v1/skills/{packageName}/install-target`

### 4. Version Detail

定位：

- 回答“这个精确版本的 registry/bundle 状态是什么”

内容：

- version status
- install policy
- npm registry / tarball / integrity / shasum
- bundle manifest digest
- components
- publish / governance timeline

Publisher 额外动作：

- `Create release record`
- `Deprecate`
- `Yank`

接口：

- `GET /api/v1/skills/{packageName}/versions/{version}`
- `POST /api/v1/skills/{packageName}/versions/{version}/governance-changes`
- `PUT /api/v1/skills/{packageName}/versions/{version}/release-record`

### 5. Import & Package

定位：

- 这是首版 Web UI 的核心工作台
- 必须显式体现 `prepare -> validate -> pack`

建议做成 4 步 wizard：

1. `Source`
2. `Target`
3. `Validation`
4. `Pack / Publish`

#### Step 1: Source

支持三类输入：

- `git`
- `local`
- `platform_authoring`

表单字段：

- source type
- location
- ref
- optional inline files for `platform_authoring`

#### Step 2: Target

字段：

- package name
- version
- display name
- description
- source type
- publish registry
- `strict` 开关

#### Step 3: Validation

这是差异化体验重点。

页面必须同时展示：

- `warnings`
- `warningDetails`
- `importReport`
- validate result
- components

推荐视觉结构：

- 顶部状态条：`Prepared / Validated / Failed`
- 左侧 `Import summary`
  - normalized / direct
  - generated paths
  - strict mode
  - identity rewrite
- 右侧 `Warnings`
  - title
  - severity
  - message
  - suggested action

设计原则：

- warning 不是红字堆砌
- `warningDetails` 用卡片
- `importReport` 用结构化摘要
- 失败错误与 warnings 视觉上必须分层

#### Step 4: Pack / Publish

内容：

- bundle path
- manifest digest
- components
- release record readiness

动作：

- `Pack`
- `Copy publish command`
- `Create release record`

注意：

- 当前系统仍然是 `npm publish` 后再 `release-record`
- 因此 UI 需要明确分开：
  - `Pack result`
  - `Registry publish done`
  - `Release record done`

不能假装这是单步完成。

### 6. Governance

定位：

- 让运营 / publisher 看 state transition

内容：

- skill-level changes
- version-level changes
- action / reason / actor / time

交互：

- filter by target type
- filter by package name
- filter by action

接口：

- `GET /api/v1/governance-changes`

### 7. Audit

定位：

- 让平台侧做追溯

内容：

- requestId
- actor
- role
- endpoint
- packageName / version
- outcome
- status code
- createdAt

交互：

- endpoint filter
- actor filter
- outcome filter

接口：

- `GET /api/v1/audit-events`

## 核心组件

### Status Chip

统一承载：

- `active / retired`
- `published / deprecated / yanked`
- `resolvable / blocked`
- `normalized / direct`

### Import Report Card

字段：

- source type
- strict mode
- normalized
- generated package.json
- generated placeholder
- rewrote package identity
- generated paths

### Warning Stack

每条 warning 卡片包含：

- severity badge
- title
- message
- suggested action
- code 折叠显示

默认面向人类说明，不默认突出机器码。

### Install Snippet Panel

显示：

- install target
- registry
- tarball
- integrity
- CLI snippet

Reader 的核心需求不是“理解底层对象模型”，而是“我现在怎么装”。

## 交互与状态

### Loading

- Catalog / versions / governance / audit 使用 skeleton rows
- Import wizard 的 `prepare / validate / pack` 使用 step-level loading

### Empty

- Catalog 空状态给出：
  - clear filters
  - jump to import
- Governance / audit 空状态不做插图，直接给过滤建议

### Error

- API error 与 domain warning 分开
- API error 使用顶部 inline error banner
- warnings 留在结果区

## 权限体验

### Reader

- 可见 Catalog / Skill / Version / Governance / Audit
- `Import & Package` 入口显示只读说明或隐藏
- 写动作按钮禁用并解释权限不足

### Publisher

- 全量可见
- 可执行 release/governance/import

## 视觉方向

建议产品不是纯“开发工具终端皮肤”，也不是“营销站”。

应采用：

- 高信息密度
- 清晰层级
- 强状态视觉
- 轻量但不冰冷的品牌感

避免：

- 大面积空洞卡片网格
- 紫色泛滥的默认 SaaS 风格
- 把 warning/error/normal 信息混成一种样式

## 首版交付建议

### P0

- `/catalog`
- `/skills/:packageName`
- `/skills/:packageName/versions/:version`
- `/import`
- `release record` 确认页

### P1

- `/governance`
- `/audit`
- 更完整的 overview

### P2

- 如果未来引入 async packaging job
  - 增加 job progress
  - retry / cancel
  - artifact drawer

## 与当前实现的贴合结论

这套 UI 是按当前 repo 真实能力设计的：

- 不依赖 runtime/operator/customer-side 运行时概念
- 主心智仍然是 `registry-first`
- 把 `importReport` 和 `warningDetails` 作为 UI 差异化能力中心
- 把 `release-record` 明确成独立步骤，而不是伪装成黑盒“一键发布”

如果要直接开始做真实前端，我建议第一批页面按这个顺序实现：

1. `Catalog`
2. `Skill Detail + Version Detail`
3. `Import & Package Wizard`
4. `Governance / Audit`
