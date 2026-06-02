# Agent Skill Registry 与 skills.sh 兼容性边界说明

## 文档定位

本文是 V1 主规范之外的定位边界文档，专门回答两个问题：

1. 当前系统是否与 `skills.sh` 对齐
2. 当前系统是否兼容 `skills.sh` 提供的 `skills` 工具

结论是：**部分对齐，部分兼容，但不应宣称“完全兼容 skills.sh”**。

实现状态补充：

- 当前 CLI 主线是 `registry-first`
- 早期兼容入口（如 `experimental_install`、`experimental_sync`、`node_modules` 同步）已从公开 CLI 中移除

## 结论先行

当前系统对齐的是：

- `skill` 作为面向用户的核心对象
- `SKILL.md` 作为核心说明文件
- `skills` 命令语言心智
- `install` 必须真正落盘到 runtime 可发现目录

当前系统没有对齐的是：

- `skills.sh` 的 source-first 安装协议
- `owner/repo` 作为安装目标
- repo/source 作为公开身份
- 直接从 source 安装的分发模型

因此：

- 可以说“兼容 `skills.sh` 的部分文件格式、CLI 心智和安装语义”
- 不应说“兼容 `skills.sh` 的完整协议和工具行为”

## 为什么不能宣称完全兼容

`skills.sh` 更偏向 source-first 模型：用户可以把 source locator 当成安装输入，安装链路也更靠近源内容本身。

当前 `Agent Skill Registry` 的 V1 则是明确的 registry-first 模型：

1. 先把 skill 源内容收敛成 bundle
2. 再发布到 npm-compatible 仓
3. 再通过 Catalog 做 release record、governance 和 install target 解析
4. 最后才由 CLI 下载并落盘

这意味着两者在底层协议上不是同一套系统。

## 分层兼容性判断

本文主要讨论定位与协议差异。

关于 `Local Installer Adapter` 的本地语义，见：

- [Local Installer Adapter 详细设计](./2026-05-28-agent-skill-registry-local-installer-adapter.zh-CN.md)

### 1. 文件层

### 兼容

- `SKILL.md` 继续作为核心文件
- `plugin.json` 可以作为兼容附件保留

### 不兼容或不完全兼容

- `package.json` 在当前系统里是标准分发工件的一部分，不是可忽略附件
- 当前系统要求 bundle 进入 publish 之前必须具备合法 `package.json`

### 结论

文件层是 **部分兼容**，不是无条件兼容。

## 2. CLI 心智层

### 兼容

当前文档保留了以下核心命令心智：

- `skills add`
- `skills find`
- `skills list`
- `skills remove`
- `skills update`
- `skills init`
- `skills publish`
- `skills inspect`

用户依然是围绕 “查找 -> 安装 -> 更新 -> 删除 -> 发布” 这条主路径来理解系统。

### 不兼容

- CLI 的位置参数只接受 `package_name`
- 不接受 `owner/repo`
- 不支持 source locator 直接作为安装目标

### 结论

CLI 心智层是 **部分兼容**：命令名字和大体操作习惯兼容，但安装输入模型不兼容。

## 3. 安装语义层

### 刻意对齐

当前设计明确要求：

- `install` 不是“只下载到缓存”
- 而是“已经落盘到当前 runtime 的可发现目录”

这点是主动向 `skills.sh` 对齐的。

### 兼容点

- 安装成功的判断标准一致偏向“本地已可发现”
- 本地安装器可维护 discoverable state
- 若某个 CLI 实现多 scope，本地 scope 视图和 winner 只是本地策略，不属于 Registry Core 协议

### 差异点

- 当前系统在安装前必须先走 `install target` 解析
- 安装目标由 Catalog 决定，而不是直接由 source 输入决定

### 结论

安装结果语义 **较强兼容**，安装前解析协议 **不兼容**。

## 4. 身份模型层

### 当前系统

当前系统唯一公开身份是：

- `package_name`

例如：

- `@platform.skill/pdf-skill`

### 与 skills.sh 的差异

`skills.sh` 的用户心智里，source locator 往往就是重要身份线索，例如：

- repo
- source alias
- 目录路径

而当前系统明确不暴露：

- repo path
- source alias
- 第二套外部身份

### 结论

身份模型 **不兼容**。

## 5. 分发协议层

### 当前系统协议

当前系统的主链路是：

1. source -> packager
2. packager -> bundle
3. bundle -> npm publish
4. publish -> `release-record`
5. install -> `install-target`
6. CLI 下载 bundle 并落盘

### 与 skills.sh 的差异

这已经不是 source-first 协议，而是：

- bundle-first
- registry-first
- governance-aware

### 结论

分发协议层 **不兼容**。

## 6. 管控 / 治理层

### 当前系统新增能力

当前系统新增了 `skills.sh` 原生模型里没有明确定义的 control plane：

- `GET /api/v1/skills/{packageName}/install-target`
- `PUT /api/v1/skills/{packageName}/versions/{version}/release-record`
- `POST /api/v1/skills/{packageName}/governance-changes`
- `POST /api/v1/skills/{packageName}/versions/{version}/governance-changes`
- `installPolicy`
- `versionStatus`
- `retired / deprecated / yanked`

这些能力说明当前系统不是在复刻 `skills.sh`，而是在构建一个 Registry / Catalog control plane。

### 结论

管理面 API **不是兼容层**，而是 **新体系**。

## 兼容矩阵

| 层级 | 判断 |
| --- | --- |
| 文件层 | 部分兼容 |
| CLI 命令名与操作心智 | 部分兼容 |
| 安装结果语义 | 较强兼容 |
| 安装输入模型 | 不兼容 |
| 身份模型 | 不兼容 |
| 分发协议 | 不兼容 |
| 管控 / 治理 API | 不兼容，属于新增体系 |

## 对外应该怎么表述

推荐表述：

- 与 `skills.sh` **保持部分兼容**
- 兼容 `SKILL.md`、CLI 基本心智与安装落盘语义
- 不兼容 `skills.sh` 的 source-first 安装协议与 repo-based identity

不推荐表述：

- 完全兼容 `skills.sh`
- 完全兼容 `skills.sh skills` 工具
- `skills.sh` 的无缝替代品

## 如果未来想做到更强兼容，还缺什么

至少还需要：

- `owner/repo` 或 source locator 作为合法安装输入
- source-first install flow
- source identity 到 `package_name` 的稳定映射层
- 不经过 Catalog 也可直接从 source 安装的协议定义
- source export / import 的双向工具链

只有补齐这些，才有资格进一步讨论“强兼容”或“完全兼容”。

## 推荐结论

V1 的正式定位应是：

- 借鉴 `skills.sh` 的用户心智和安装落盘语义
- 保留 `SKILL.md` 兼容性
- 在分发、身份、治理和安装解析上采用新的 Registry / Catalog 模型

一句话总结：

当前系统不是 `skills.sh` 的协议兼容实现，而是一个 **保留部分使用体验、但采用新分发与治理模型的 skill registry 系统**。
