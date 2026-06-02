# External Skill Import Normalization 详细设计

## 文档定位

本文是对 [Skill Packager / Importer 详细设计](./2026-05-28-agent-skill-registry-skill-packager-importer.zh-CN.md) 的实现收口补充，专门定义“外部 existing skill 导入”在 `prepare` 阶段的归一化语义。

它解决的不是新的发布模型，而是把现有设计里已经声明的“外部导入”真正落到实现上。

## 问题陈述

现有设计文档已经定义：

- `prepare` 负责拉取或生成工作目录
- `prepare` 负责生成或重写 `package.json`
- `prepare` 负责整理能力目录

但当前实现只做了两件事：

1. 把 `local` / `git` 源复制到临时 workspace
2. 按 target 重写 `package.json`

这导致一个明显缺口：

- 只有根级 `SKILL.md` 的 existing skill
- 没有 `package.json`
- 没有 `commands/` / `docs/` / `plugin.json` 等 capability 内容

即使这类 skill 本身语义完整，也会在 `validate` 阶段因为 `missing_capability_content` 失败，无法走完“外部导入 -> pack -> publish -> release-record”链路。

这与既有设计中“外部导入 existing skill 源目录”应被支持的目标不一致。

## 目标

- 让 `local` / `git` 外部 existing skill 能在不污染原始目录的前提下导入
- 把归一化逻辑统一收口到 `prepare`
- 保持 `validate` / `pack` 的 bundle 结构要求不变
- 保留严格模式，允许调用方显式要求“只校验、不补齐”
- 让 CLI `publish`、批量导入、未来平台导入共用同一套导入兼容层

## 非目标

- 不改变 install target、release-record、治理模型
- 不支持没有根级 `SKILL.md` 的目录自动推导为 skill
- 不把“已安装态目录”原样视为 publishable bundle
- 不自动生成真实 commands / scripts / hooks 逻辑
- 不回写原始导入目录或外部 git 仓

## 核心决策

### 1. 归一化落在 `prepare`

新增语义：

- `prepare` 不再只是“复制源目录”
- 对 `local` / `git` 源，`prepare` 可以在临时 workspace 内执行最小归一化

原因：

- 最符合原 packager/importer 设计职责
- CLI `publish`、批量导入、平台导入都能直接复用
- 避免把兼容逻辑散落到 batch-import 或某个单独命令里

### 2. 默认自动补齐，保留严格模式

默认行为：

- 对 `local` / `git` 外部源启用归一化

严格模式：

- 新增 `options.strict`
- `options.strict=true` 时，`prepare` 只复制源内容并重写 `package.json`
- 严格模式下不生成 capability placeholder，不补导入兼容文件

说明：

- `platform_authoring` 不走这套 imported normalization 语义
- 它仍按现有 authoring 输入模型工作

### 3. 只修改临时 workspace

归一化永远只发生在 `prepare` 产生的 workspace 中：

- 不修改 `source.location` 指向的本地目录
- 不修改外部 git 仓源内容
- 不修改用户已安装的 `~/.agents/skills/...` 目录

这样可以保证：

- 导入是可重复、可审计的
- 原始源保持只读心智
- 不会把消费态目录悄悄“升级成源码目录”

### 4. 最小 capability placeholder 选用 `docs/README.md`

当外部源满足：

- 有根级 `SKILL.md`
- 但没有 capability 目录或兼容文件

系统自动补：

```text
docs/
  README.md
```

理由：

- 最轻量
- 不伪造可执行能力
- 满足现有 bundle 结构要求
- 对导入结果最容易解释

不会自动补：

- `commands/`
- `scripts/`
- `hooks/`
- 伪造的 `plugin.json`
- 伪造的 `.mcp.json`

## 归一化规则

### 输入前提

对 `local` / `git` 外部导入，仍要求：

- 根级 `SKILL.md` 必须存在
- `SKILL.md` frontmatter 至少有 `name`
- `SKILL.md` frontmatter 至少有 `description`

如果缺失这些前提，导入仍然失败。

### package.json 规则

`prepare` 在 workspace 中总是生成或重写 `package.json`，以内部 target 为准。

最小输出字段：

- `name`
- `version`
- `description`
- 可选 `publishConfig.registry`
- `skill.sourceType`
- `skill.normalizedFromImport`

建议输出示例：

```json
{
  "name": "@platform.skill/zoom-out",
  "version": "1.0.0",
  "description": "Tell the agent to zoom out and give broader context or a higher-level perspective.",
  "publishConfig": {
    "registry": "http://127.0.0.1:4873"
  },
  "skill": {
    "sourceType": "imported",
    "normalizedFromImport": true
  }
}
```

约束：

- `name` 始终以 `target.packageName` 为准
- `version` 始终以 `target.version` 为准
- `description` 优先取 request metadata
- `skill.sourceType` 优先取 request metadata，否则使用默认 source type
- `normalizedFromImport=true` 只在实际发生 imported normalization 时写入

如果源目录本来就有合法 `package.json`，但 identity 与 target 冲突，仍按 target 重写。

### capability placeholder 规则

若 workspace 中不存在以下任一内容：

- `commands/`
- `agents/`
- `hooks/`
- `scripts/`
- `docs/`
- `plugin.json`
- `.mcp.json`

则 `prepare` 自动创建：

```text
docs/README.md
```

文件内容要求：

- 明确说明这是 imported-normalized placeholder
- 明确说明原始 skill 未提供显式 capability 目录
- 不伪装为真实执行说明

建议内容：

```md
# Imported Skill Placeholder

This file was generated during packaging normalization because the imported skill
did not include any capability directories or compatibility manifests.
```

### warnings 规则

`prepare` 在发生自动补齐时返回 warning，而不是静默处理。

新增稳定 warning 值：

- `normalized_missing_package_json`
- `normalized_missing_capability_content`

可选扩展 warning：

- `normalized_rewrote_package_json_identity`

warning 语义：

- 调用成功，但 workspace 不是原始源的原样镜像
- 方便 CLI、batch-import、平台 UI 做提示和审计

## API 变更

### POST /api/v1/skill-packaging/prepare

对应 Go 请求结构需要扩展为：

```go
type Options struct {
    PublishRegistry string `json:"publishRegistry,omitempty"`
    Strict          bool   `json:"strict,omitempty"`
}
```

请求 `options` 新增：

```json
{
  "strict": true
}
```

默认值：

- `false`

行为：

- `strict=false`：允许 imported normalization
- `strict=true`：关闭 imported normalization

成功响应继续返回：

- `workspacePath`
- `packageName`
- `version`
- `warnings`
- `errors`

其中 `warnings` 可能包含新增 normalization warning。

### validate / pack

`validate` / `pack` 的结构规则不变：

- 仍然要求 `package.json`
- 仍然要求至少一种 capability 内容

变化点在于：

- 这些结构现在可以由 `prepare` 的 imported normalization 补齐
- 而不是要求外部源必须天然满足 publishable bundle 结构

## 各入口语义

### skills publish

默认：

- 允许 imported normalization
- 如果本地目录只有 `SKILL.md`，但元数据足够，发布可以成功

本轮结论：

- 不增加新的 CLI flag
- CLI 默认走 `strict=false`
- 严格模式先通过直接调用 packaging API 或批量导入脚本扩展使用

### batch import

默认：

- 允许 imported normalization
- 用户不需要为纯 `SKILL.md` existing skill 手工补 `package.json`
- 用户也不需要手工添加 `docs/README.md`

如果调用方需要“只导入本来就满足 publishable bundle 契约的目录”，应显式传 strict。

### 平台导入

未来平台侧如果接 `git` / `local mirror` 导入，同样复用这套语义：

- imported normalization 属于 packager 层
- 不在平台 UI 或业务服务层重复实现

## 错误与边界

以下情况仍应失败：

- 缺根级 `SKILL.md`
- `SKILL.md` frontmatter 缺 `name`
- `SKILL.md` frontmatter 缺 `description`
- 存在嵌套 `SKILL.md`
- 存在符号链接、大小写冲突、路径不安全
- `target.packageName` 非法
- `target.version` 非法
- `strict=true` 且缺 capability 内容

也就是说：

- 自动补齐解决的是“分发工件最小结构缺口”
- 不是放宽对 skill identity 和安全边界的要求

## 实现范围

### Go 服务

- 在 `business/packaging/service.go` 的 `Prepare` 流程里补 imported normalization
- 为 `PrepareRequest.Options` 增加 `Strict`
- 抽出 workspace component 扫描与 normalization 辅助函数

### CLI / 脚本

- CLI packaging session 保持与服务端语义一致
- batch-import 不再要求已有 skill 手工补 `package.json` 或 capability placeholder
- 必要时在 CLI 输出中透出 `prepare.warnings`

### 测试

至少覆盖：

1. `local` source 只有 `SKILL.md` 时，默认 prepare + validate 成功
2. 缺 `package.json` 时自动生成
3. 缺 capability 内容时自动生成 `docs/README.md`
4. `prepare.warnings` 正确返回 normalization warning
5. `strict=true` 时同样输入仍失败
6. 原始源目录未被修改

## 与现有主规范的一致性

这次设计不是改写原规范，而是把原规范中已经存在但未完整实现的部分补齐：

- “生成或重写 `package.json`”
- “整理能力目录”
- “支持外部导入 existing skill 源目录”

当前缺的不是新的产品方向，而是把 imported normalization 从文档义务变成真实运行时行为。
