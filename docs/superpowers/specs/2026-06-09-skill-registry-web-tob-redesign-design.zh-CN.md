# Skill Registry Web ToB 重设计

## 文档定位

本文定义当前 `apps/skills-web` 的一次面向 ToB 场景的产品化重设计。

目标不是重写前端，也不是改变现有后端契约，而是在已有 `Catalog / Packaging / Governance / Audit` 能力之上，把当前更像“内部工作台”的界面，升级成一个真正可交付的企业级 `registry portal + operations console`。

本次设计结论来自以下已确认的方向：

- 产品主身份：`开发者门户 > 云平台控制台`
- 首页转化优先级：`搜索并进入 skill 详情 > curated / recommended > 安装信息`
- 目录卡片优先级：`用途与适用场景 > 接入成本 > owner / team`
- 详情页主 CTA：`Install / Copy command`
- 后台能力组织：`Console 次级工作区 > 权限驱动显式展开`
- 视觉主基调：`深色开发者门户 > 混合前后台 > 浅色企业门户`
- 总体方案：`B 为主，吸收 A 的首页感`

## 设计目标

- 让开发团队把这个产品视为默认的 skill 发现与安装入口
- 让 `Catalog -> Skill Detail -> Install` 成为最短、最清晰的主路径
- 让 `authoring / import / governance / audit` 从“和目录并列的同级导航噪音”变成明确的 `Console` 工作区
- 让界面在视觉、状态、权限、空白态和异常态上达到真实 ToB 产品的完成度
- 在不改变后端 API 契约的前提下，显著提升可读性、可信度和产品层级

## 非目标

- 不迁移技术栈，不把项目改成 Tailwind、Next.js 或其它方案
- 不重做后端数据模型，不要求新增聚合 API 才能落地
- 不在本次设计中重写导入、发布、治理的业务逻辑
- 不把 Portal 做成营销官网
- 不把 Console 做成大屏式运营驾驶舱

## 现状诊断

当前前端已经具备基础页面和数据流，但整体气质、信息架构和状态表达仍偏向“内部工具壳层”：

- 视觉基调偏暖色编辑台，不符合企业级 registry 的可信、冷静、开发者导向气质
- 首页更像分散的信息面板，不像“发现入口”
- 主导航把目录、工作台、导入、治理、审计并列暴露，前台与后台分层不清
- 目录页过早暴露元数据和状态摘要，未优先回答“这个 skill 有什么用”
- 详情页还没有把 `Install / Copy command` 建成最强行动区
- 治理与审计仍主要以时间线表达，缺少“影响面摘要 -> 进入详情”的层次
- 产品完整性缺口仍在：
  - 缺少 `skip to content`
  - 缺少法律链接
  - 缺少明确的 `404` 页面
  - 缺少更完整的 meta 信息
  - 缺少更产品化的空白态、错误态、只读态

## 约束

### 技术约束

- 继续使用 `React 19 + Vite + 自定义 CSS`
- 保持现有路由和页面职责，不做全量重写
- 页面层的设计重排不能破坏现有测试覆盖的关键行为

### 契约约束

前端继续直接消费当前后端契约，包括但不限于：

- `GET /api/v1/skills`
- `GET /api/v1/skills/curated`
- `GET /api/v1/skills/{packageName}`
- `GET /api/v1/skills/{packageName}/versions`
- `GET /api/v1/skills/{packageName}/versions/{version}`
- `GET /api/v1/skills/{packageName}/install-target`
- `GET /api/v1/governance-changes`
- `GET /api/v1/audit-events`
- `POST /api/v1/skill-packaging/*`
- skill / version governance mutation endpoints

### 权限约束

- 普通开发者默认看到 Portal 主路径
- `Console` 对有权限用户显式可达，对无权限用户弱化或隐藏
- 只读态必须可理解，不能只靠接口报错反推权限

## 用户与主任务

### 开发者用户

主要任务：

- 搜索 skill
- 理解 skill 作用与适用场景
- 评估兼容性、版本状态和接入成本
- 复制安装命令或 registry 接入信息

### 平台运营 / 发布 / 治理用户

主要任务：

- 导入或打包 skill
- 审核发布与 release 信息
- 做 retire / deprecate / yank
- 查看治理记录与审计事件

本次重设计优先优化第一类用户的进入与转化路径，同时保证第二类用户有明确、可信的后台入口。

## 设计总纲

### 总体结构

产品采用双层结构：

1. `Portal`
2. `Console`

其中：

- `Portal` 是默认入口，服务发现、评估、安装
- `Console` 是次级工作区，服务导入、发布、治理、审计

这不是两个独立产品，而是一个产品中的前台与后台。

### 命名与导航原则

- 主导航优先暴露开发者路径
- 后台能力通过单一 `Console` 入口收纳
- 当前页高亮必须清晰
- 导航不能把“发现”和“操作”混成一层

## 信息架构

### Portal

- `/`
  - `Overview`
- `/catalog`
  - `Catalog`
- `/skills/:packageName`
  - `Skill Detail`
- `/skills/:packageName/versions/:version`
  - `Version Detail`

### Console

- `/workbench`
  - `Console Home`
- `/workbench/authoring`
  - `Authoring`
- `/workbench/import`
  - `Import`
- `/workbench/governance`
  - `Governance`
- `/workbench/audit`
  - `Audit`

## 视觉语言

### 基调

视觉方向采用深色开发者门户，而不是暖色工作台，也不是泛 AI 风格霓虹控制台。

核心特征：

- 深石墨、冷灰、钢蓝为主
- 单一强调色，不混用多组高饱和渐变
- 强搜索、强结构、弱装饰
- 突出可信、稳定、工程感

### 颜色策略

- 基础底色：深石墨 / 冷黑蓝
- 面板层：低反差深色面板
- 强调色：单一钢蓝或冷青蓝
- 警示色：
  - `warn` 用琥珀
  - `bad` 用低饱和红
  - 只在状态与风险处局部使用

避免：

- 米色、纸张感、暖棕阴影
- 紫蓝 AI 渐变
- 同时使用多个强调色

### 字体与排版

- 替换当前偏温和的组合，改为更干净、工程感更强的无衬线体系
- 大标题收紧字距，建立更强层级
- 正文控制阅读宽度
- 版本号、命令、时间、计数使用等宽或启用 `tabular-nums`
- 降低全站大写标签比例，用 sentence case 为主

### 动效原则

- 页面切换：轻量淡入和位移
- 结果进入：分批次轻量 reveal
- 卡片与按钮：明确 hover / pressed / focus
- 不做大幅炫技动画，不影响 registry 工具属性

## 页面设计

### 1. Overview

定位：

- 开发者默认入口
- 以搜索和发现为第一目标
- 用少量可信信号建立“这是正式 registry”的感知

布局：

- 顶部强搜索 hero
- 非对称内容区：
  - 左：`recommended / curated skills`
  - 右：`trusted signals / recently updated / status digest`
- 下方：结果预览带或最近新增 / 最近更新流
- 低干扰 `Console` 入口

内容原则：

- 先让用户开始找 skill
- 再给推荐和可信摘要
- 不做平均分栏的大卡片看板

### 2. Catalog

定位：

- 真正的 registry 浏览页
- 回答“有什么、适不适合我、现在能不能装”

布局：

- 顶部固定搜索和高价值筛选
- 中间主列为结果流
- 右侧为上下文侧栏，可承接：
  - 筛选摘要
  - 推荐集合
  - 版本或兼容性说明

结果卡信息优先级：

1. 用途描述与适用场景
2. 安装方式、兼容 agent、版本状态
3. owner / team / 最近更新时间

结果卡应包含：

- display name
- package name
- 一句话描述
- 标签或场景
- source type
- latest version
- latest version status
- 兼容性摘要
- 次级 ownership 信息

避免：

- 让治理元数据主导列表阅读
- 过度平均的三栏卡片阵列
- 把筛选面板做成主视觉焦点

### 3. Skill Detail

定位：

- 全产品最关键的转化页
- 首要任务是让用户可以安全、快速地安装

首屏结构：

- 左列：
  - skill 价值说明
  - 适用场景
  - 标签
  - 状态与可信摘要
- 右列：
  - 固定安装行动区

安装行动区必须优先承载：

- `Install`
- 复制命令
- registry / install target
- 最新版本
- 兼容 agent
- 必要 warning
- deprecated / retired 提示

页面其余区域承载：

- 版本列表
- 变更与治理摘要
- 所有者或来源信息
- 相关 skill / 替代建议

原则：

- 用户不需要先完整阅读页面，才能拿到安装动作
- 评估信息围绕安装区展开，而不是让安装动作埋在评估信息后面

### 4. Version Detail

定位：

- 面向具体交付物的版本级确认页
- 回答“这个版本是否可装、为什么可装、和其它版本差在哪”

优先信息：

- version
- release status
- install target
- tarball / digest / integrity
- 版本级 warning
- 版本级治理状态
- 与最新版本的关系

### 5. Console Home

定位：

- 后台工作区入口
- 汇总“我可以在这里执行哪些操作”

内容：

- 导入 / 打包入口
- 发布 / authoring 入口
- 治理 / 审计入口
- 权限态摘要

不追求：

- 大屏监控
- 把后台首页做成指标驾驶舱

### 6. Governance

定位：

- 从单纯时间线升级为“影响面优先”的治理控制台

表达层次：

1. 影响摘要
2. 当前筛选上下文
3. 分层事件流
4. 进入具体 skill / version 详情的入口

### 7. Audit

定位：

- 可追溯，但不喧宾夺主

设计要求：

- 保留检索与筛选能力
- 先回答“发生了什么类型的操作”
- 再进入逐条事件阅读

## 状态设计

### Loading

- 用骨架屏替代泛化的简单文案
- Portal 与 Console 的骨架应接近真实页面结构

### Empty

- 搜索空结果要给下一步建议
- 没有治理记录、没有审计记录、没有版本时，空白态文案需说明含义与下一步

### Error

- 区分：
  - 网络失败
  - 后端错误
  - 权限受限
  - 资源不存在
- 能保留旧结果时，必须明确告诉用户当前展示的是上一次成功结果

### Read-only / Permission

- 不能只在按钮点击时报错
- 应在页面层直接表达：
  - 当前会话可读
  - 当前会话可执行
  - 当前动作需要特定 package scope

### Not found

- 补充真实 `404` 页面
- 提供返回目录、返回首页或回到上一层的路径

## 真实 ToB 完成度补项

本次设计必须一并补上以下产品级缺口：

- `skip to content`
- 明确的当前导航态
- `404` 页面
- `<title>` / `description` / 社交分享 meta
- 页脚中的 `Privacy` / `Terms`
- 更完整的空白态、错误态、只读态
- 更清晰的 back navigation

这些项不是额外优化，而是“真实 registry 产品”成立的最低要求。

## 实现边界

### 保持不变

- 现有 API 契约
- 现有前端技术栈
- 现有业务逻辑主路径

### 允许调整

- `AppShell`
- 全局样式变量与主题体系
- Portal 与 Console 导航结构
- 首页、目录、详情页和后台页的布局重排
- 状态组件的视觉与文案
- 产品级补完项

## 分阶段落地

### P0: Portal 重建

优先改造：

- `AppShell`
- `Overview`
- `Catalog`
- `Skill Detail`
- `Version Detail`
- 全局 design tokens
- 关键状态组件

目标：

- 先让开发者主路径成立

### P1: Console 归位

后续改造：

- `Console Home`
- `Workbench / Authoring / Import / Governance / Audit` 的入口关系和视觉统一

目标：

- 让后台从“散落页面”变成“清晰工作区”

## 验收标准

### 视觉验收

- Portal 与 Console 有清晰层次差异，但属于同一产品
- 首页第一屏明确服务搜索与发现
- 深色基调稳定、克制，不出现 AI 风格噪音

### 交互验收

- 搜索、筛选、跳转、分页、安装信息展示不退化
- 详情页安装动作在首屏清晰可达
- 权限与只读态在进入页面时即可理解

### 产品验收

- 用户能自然区分 Portal 与 Console
- 开发者用户从首页进入 skill detail 的路径显著更短
- 页面具备真实 ToB 产品所需的基础完整度

## 风险与注意事项

- 若视觉重做过强而信息架构不变，只会得到“换皮工作台”，无法达成目标
- 若过度强调治理与审计，会削弱开发者主路径
- 若首页做成营销页，会背离 registry 的可信和工具属性
- 若忽略权限态与异常态，ToB 完成度仍然不足

## 结论

本次重设计采用：

- `Portal + Console` 双层结构
- `B 为主，吸收 A 的首页感`
- 深色开发者门户视觉
- `Catalog -> Skill Detail -> Install` 为核心主路径

先用 `P0` 建立开发者入口和安装转化，再用 `P1` 让后台工作区归位。这是在当前代码与后端边界内，风险最低、收益最高的升级路径。
