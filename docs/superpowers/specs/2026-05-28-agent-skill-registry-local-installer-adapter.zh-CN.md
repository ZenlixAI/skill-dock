# Local Installer Adapter 详细设计

## 文档定位

本文是 V1 主规范之外的消费侧补充文档，描述 skill 的搜索、安装目标解析，以及一个可选的 `Local Installer Adapter` 在本地落盘时可采用的最小语义。

文件名沿用早期讨论阶段的命名；当前文档不再把本地作用域视为 Registry Core 的必备远端模型。

补充说明：

- 当前 V1 正式主线仍是 `registry-first`
- 当前实现已移除 `node_modules` 同步等兼容入口；旧 source-based lock entry 只作为历史读取兼容元数据存在
- 本文以下描述的“推荐本地安装流程”默认指 registry-resolved 安装路径

## 目标

- 搜索结果以 skill 为中心
- 安装流程先解析、后下载
- 让 install target 解析与本地安装 adapter 解耦
- 明确本地安装与 lockfile 的边界

## 搜索模型

本节只描述发现与安装语义。

具体 HTTP 读接口、字段命名和响应结构见：

- [Catalog API 详细设计](./2026-05-28-agent-skill-registry-catalog-api-detailed-design.zh-CN.md)
- [API 与 CLI 契约](./2026-05-28-agent-skill-registry-api-cli-contracts.zh-CN.md)

搜索结果按 `skill` 维度返回，而不是按 `version` 展开。

主展示字段：

- `display_name`

稳定引用字段：

- `package_name`

支持过滤：

- `q`
- `package_name`
- `source_type`
- `include_retired`

默认不返回 retired skill，除非显式要求。

## install target 解析

安装不会直接去 npm 仓猜版本，而是先向 Catalog 请求 install target。

说明：

- 文档中的领域名统一使用 `package_name`
- HTTP JSON 字段沿用 API 风格，使用 `packageName`

V1 只支持：

- 空版本：默认 latest 解析
- 精确版本：显式安装

V1 不支持 semver range。

### 默认解析规则

1. 只从 `published` 版本中选
2. 取最高 semver
3. 跳过 `deprecated`
4. 跳过 `yanked`
5. retired skill 直接失败

### 显式版本规则

- `published` 可安装
- `deprecated` 可显式安装
- `yanked` 不参与正常安装解析
- 不存在版本返回结构化失败

完整 request / response schema 与错误码见：

- [Catalog API 详细设计](./2026-05-28-agent-skill-registry-catalog-api-detailed-design.zh-CN.md)
- [Catalog API OpenAPI](./2026-05-28-agent-skill-registry-catalog-api.openapi.yaml)

## Registry Core 与本地安装的边界

`install target` 解析属于 Registry Core。

“下载、解包、放到哪里、如何让 runtime 发现它” 属于 `Local Installer Adapter`。

V1 不要求所有消费端都实现同一套本地 scope 模型；只要消费端能把解析后的 bundle 落到自己的 runtime 可发现位置，就不影响 Registry Core 的成立。

## 一个推荐的本地安装流程

1. 用户执行 `skills add <package>`
2. CLI 调用 install target 解析接口
3. Catalog 返回 install target
4. `Local Installer Adapter` 从内部 npm 仓下载 bundle
5. `Local Installer Adapter` 解包并安装到某个本地可发现目录
6. 如该安装器实现了本地状态管理，则更新本地 discoverable state

安装成功不只是“下载完成”，还必须确保本地 discovery 可见。

这一定义应与 `skills.sh` 的用户心智保持一致：`install` 指 skill 已经落盘到当前 runtime 的可发现目录，而不是仅存在缓存或 tarball 下载记录中。

旧 source-based lock replay 不再作为公开 CLI 能力存在；若本地状态里仍残留这些字段，应只把它们理解为历史兼容元数据，而不是 V1 安装协议的一部分。

### 可选的本地作用域模型

以下 scope 语义是对实现 `skills` CLI 的推荐本地策略，不是 Registry Core 的远端领域对象。

若实现多 scope，本地安装器可支持：

- `global`
- `project`
- `agent`

推荐默认 scope：

- `project`

若实现多 scope，每个 scope 中，同一 `package_name` 最多只能安装一个版本。

这里的 `project` 仅表示一个本地安装位置标签，不表达 Registry / Catalog 里的项目依赖声明。

### scope precedence

如果某个本地安装器允许多个 scope 同时存在，同一 package 的 winner 可固定为：

`agent > project > global`

不实现多 scope 的安装器可以完全跳过这套规则。

## CLI 行为

### skills add

- 解析安装目标
- 下载 bundle
- 安装到本地安装器约定的可发现目录

### skills list

- 若实现多 scope，则展示各 scope 中已装版本
- 若实现多 scope，则标记当前 winner

### skills remove

- 仅删除本地安装副本
- 不改变 Catalog 生命周期

### skills update

- 重新解析当前安装 skill 的默认可安装版本
- 若无可解析版本或被治理阻止，必须显式失败

## Installed View

若 CLI 实现本地安装视图，建议至少显示：

- `package_name`
- 已安装版本
- scope
- 是否当前 winner

## 当前项目级 `skills-lock.json`

当前实现已经引入项目级 lockfile：`<project>/skills-lock.json`。

它不是 Registry Core 的远端领域对象，而是本地安装器的项目态快照，服务于：

- `list` / `remove` / `update` 的项目级行为
- 已安装 skill 的去重与 discoverable-state 归一
- 尽量减少分支合并冲突

补充说明：

- 本节讨论的是项目级 `skills-lock.json`
- 全局状态仍使用单独的 `~/.agents/.skill-lock.json` 或 `$XDG_STATE_HOME/skills/.skill-lock.json`
- 两者职责不同，不应混用

### 文件位置

- 项目级：`<cwd>/skills-lock.json`
- 默认与项目 scope 安装目录 `./.agents/skills/` 配套出现

### 顶层结构

当前 schema version 为 `1`：

```json
{
  "version": 1,
  "skills": {
    "@platform.skill/pdf-skill": {
      "sourceType": "registry",
      "computedHash": "2d4f...",
      "packageName": "@platform.skill/pdf-skill",
      "version": "1.2.0",
      "registry": "https://catalog.example.com",
      "integrity": "sha512-...",
      "skillManifestDigest": "sha256-..."
    }
  }
}
```

约束：

- `version`：lockfile schema version
- `skills`：按 skill key 建立的 map；当前 registry-first 主线下，推荐直接使用 `package_name`

### 单个 entry 字段

必需字段：

- `sourceType`
- `computedHash`

registry-first 主线下的正式字段：

- `packageName`
- `version`
- `registry`
- `integrity`
- `skillManifestDigest`

兼容旧 source-based 安装来源时，仍允许历史顶层字段存在：

- `source`
- `ref`
- `skillPath`

这些字段的语义已经降级为历史兼容元数据：

- 新写入的 registry entry 不再主动写这些字段
- 读取旧 entry 时，会在内部归一化为 `legacy` 元数据
- `update` 等主流程不得把这些字段当作正式协议输入

### `computedHash`

`computedHash` 是项目级 lockfile 的核心字段。

定义：

- 对已安装 skill 目录下所有文件递归计算 SHA-256
- hash 输入包含相对路径和文件内容
- 遍历前按相对路径排序，以保证 deterministic output
- 目录内的 `.git` 与 `node_modules` 不参与计算

设计目的：

- 判断本地安装目录是否与 lock entry 描述一致
- 避免依赖远端 GitHub tree SHA
- 让项目级安装快照完全由磁盘真实内容决定

### 写入规则

项目级 `skills-lock.json` 当前实现遵循以下写入约束：

- 不写时间戳
- `skills` key 按字母序排序
- 文件以 pretty JSON + trailing newline 写出

这样做的目的不是追求“完整审计”，而是：

- 降低多分支同时新增 skill 时的 merge conflict 概率
- 让 git diff 更稳定
- 使 lockfile 更接近可提交的项目元数据，而不是易抖动的运行时状态

### 读取与容错规则

当前实现的读取策略偏保守：

- 文件不存在：返回空结构
- JSON 损坏：返回空结构
- 出现 merge conflict marker 等无效内容：返回空结构
- 旧版本 `version`：迁移到当前 schema version，并保留已有 entry

这意味着 `skills-lock.json` 更偏“可重建的项目态缓存”，而不是不可丢失的唯一事实源。

### 与项目安装目录的关系

项目级安装的 canonical 目录是：

- `./.agents/skills/<sanitized-skill-name>`

`skills-lock.json` 不直接取代目录扫描；它与目录扫描互补：

- lockfile 提供 package 级元数据和已解析版本信息
- 目录提供真实 discoverable state

当两者都存在时，CLI 优先把 lockfile 视为项目安装来源的结构化说明。

## 本地状态持久化的收口原则

V1 仍然不引入 npm 式 project dependency graph。

当前本地状态持久化的收口原则是：

- 项目级 lockfile 只表达已安装结果，不表达依赖求解图
- 新写入 entry 只保留 registry-first 所需正式字段
- 历史 source-based 字段只作为兼容读取输入继续保留
- 本地安装器可以据此支持 `list` / `remove` / `update`，但不应把它上升为 Registry Core 领域模型

这里的 `discoverable state` 不是 Registry Core 字段，而是“该安装副本是否已经处于当前 runtime 可发现目录并可被发现逻辑识别”的本地状态。

## 测试建议

### 单元测试

- latest 解析算法
- exact version 解析
- 若实现多 scope，则测试 scope precedence
- 本地 update / remove 逻辑

### 集成测试

- CLI + Catalog install target 解析
- CLI + npm 仓下载
- 若实现多 scope，则覆盖多 scope 安装场景
