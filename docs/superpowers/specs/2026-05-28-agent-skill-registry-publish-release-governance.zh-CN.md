# 发布、登记与治理详细设计

## 文档定位

本文是 V1 主规范之外的发布与治理补充文档，描述 bundle 如何从“已被 packager 生成”进入“published release”，并定义治理状态机与审计规则。

## 核心原则

- 制品存储成功不等于业务发布成功
- Catalog 是治理真相源
- `package_name + version` 不可变

## 发布链路

1. Packager 产出合法 bundle
2. 发布方将 bundle `npm publish` 到内部 npm-compatible 仓
3. 发布方调用 `PUT /api/v1/skills/{packageName}/versions/{version}/release-record`
4. Catalog 拉取并复检 bundle
5. Catalog 写入 skill、version 与治理记录

## npm publish 语义

失败场景：

- registry 不可达
- 认证失败
- 同版本覆盖冲突
- 上传中断

这类失败发生在 Catalog 之前，Catalog 不应感知该版本。

### 关于 publish 返回值的边界

V1 不应把 Catalog 设计成依赖 `npm publish` 命令响应体本身。更稳妥的做法是：

1. `package_name` 和 `version` 直接来自待发布 bundle 内的 `package.json`
2. `npm publish` 成功后，再从 npm-compatible 仓查询该包的版本元数据
3. Catalog 只消费“发布后可查询到的 registry 权威元数据”

原因：

- 不同 npm-compatible 仓对 publish 响应体的细节承诺可能不一致
- `name` / `version` 是发布输入的一部分，不需要依赖响应体回填
- `integrity`、`tarball` 等更适合从仓端元数据读取，而不是依赖 CLI 当场返回值

## Registry 元数据读取模型

`release-record` 之前或期间，发布方或 Catalog 至少要能从 npm-compatible 仓读取该版本的包元数据。V1 建议把 registry 返回字段分成两类。

### 应依赖的稳定字段

- `name`
  对应 `package_name`
- `version`
  对应内部版本号
- `dist.tarball`
  tarball 下载地址
- `dist.integrity`
  完整性校验值
- `dist.shasum`
  兼容性摘要
- `dist-tags`
  可选，用于了解默认 tag，如 `latest`
- `time[version]`
  该版本发布时间
- `npm_registry`
  发布到哪个内部 npm-compatible 仓

### 可选读取但不应当作核心真相源的字段

- `description`
- `keywords`
- `repository`
- `author`
- `_npmUser`
- `_npmVersion`
- `_nodeVersion`

这些字段可用于展示、审计辅助或来源分析，但不应用于决定 identity、治理状态或 install policy。

## Catalog 从 Registry 读取的最小源数据

对当前系统来说，Catalog 至少应从 registry 元数据中稳定获取：

- `package_name`
- `version`
- `npm_registry`
- `integrity`
- `shasum`
- `tarball_url`
- `published_at`
- 可选 `dist_tag`

这些字段是“registry 源数据”，不等于 Catalog 自己的治理字段。

## release-record 语义

`PUT /api/v1/skills/{packageName}/versions/{version}/release-record` 的语义不是上传文件，而是“为已存在于 npm 仓的制品创建 Catalog 记录”。

服务端应：

1. 根据 `package_name + version` 定位制品
2. 读取 registry 版本元数据
3. 拉取 bundle
4. 扫描 `SKILL.md`
5. 校验 `package.json`
6. 校验 digest / integrity / single-skill 约束
7. 写入 skill 记录、version 记录和审计记录

### 真相源分层

release-record 过程中，不同字段的真相源应明确分层：

- `package_name`
  来自 `package.json.name`
- `version`
  来自 `package.json.version`
- `integrity` / `shasum` / `tarball_url` / `published_at`
  来自 npm-compatible 仓元数据
- `skill_manifest_digest`
  由 Catalog 服务端扫描 bundle 计算
- `components`
  由 Catalog 服务端扫描 bundle 推断
- `install_policy`
  由 Catalog 治理逻辑决定
- `version_status`
  由 Catalog 生命周期状态机决定

## Published Release 与 Orphaned Artifact

### Published Release

同时满足：

- bundle 已在 npm 仓
- release record 登记成功

### Orphaned Artifact

只满足：

- bundle 已在 npm 仓
- Catalog 未登记成功

语义：

- 不参与搜索
- 不参与安装解析
- 不进入正常版本历史集合

## 幂等与冲突

- 同一 `package_name + version` 重复登记且内容一致：幂等成功
- 同一 `package_name + version` 已存在但内容不一致：硬拒绝

## 数据模型

### skill 主记录

- `package_name`
- `display_name`
- `description`
- `source_type`
- `status`

### version 记录

- `package_name`
- `version`
- `version_status`
- `integrity`
- `shasum`
- `tarball_url`
- `published_at`
- `npm_registry`
- `skill_manifest_digest`
- `install_policy`

### governance 记录

- `target_type`
- `package_name`
- 可选 `version`
- `action`
- `reason`
- `changed_by`
- `changed_at`
- 可选 `replacement_recommendation`

## 生命周期状态机

### skill 级

- `active`
- `retired`

### version 级

- `published`
- `deprecated`
- `yanked`

语义：

- `published` 参与默认 latest
- `deprecated` 只允许显式安装
- `yanked` 退出正常安装流
- `retired` 作用于 skill 级

推荐流转：

| 目标 | 当前状态 | 可流转到 |
| --- | --- | --- |
| skill | `active` | `retired` |
| version | `published` | `deprecated`、`yanked` |
| version | `deprecated` | `yanked` |

## install policy 与导入准入

对外部导入版本，`source_type` 只描述来源，不决定是否可安装。

建议状态：

- `catalog_only`
- `resolvable`
- `blocked`

## governance change

V1 将治理写接口按资源挂载，而不是收敛到单一顶层 RPC：

- `POST /api/v1/skills/{packageName}/governance-changes`
- `POST /api/v1/skills/{packageName}/versions/{version}/governance-changes`

skill 级输入字段：

- path `packageName`
- `action`
- `reason`

version 级输入字段：

- path `packageName`
- path `version`
- `action`
- `reason`
- 可选 `replacementRecommendation`

要求：

- 每次治理动作必须写审计记录
- 必须记录操作者和时间
- 不允许 silent status flip

常见动作：

- `deprecate`
- `yank`
- `retire`

## 失败语义

### publish 失败

- Catalog 不感知

### release-record 失败

- 结果是 orphaned artifact

### governance change 失败

- 当前治理状态不变

## 权限

V1 最小权限模型：

- `reader`
- `publisher`

约束：

- `publisher` 才能发布和治理
- `reader` 可搜索、查看和解析安装

## 测试建议

### 单元测试

- 幂等登记
- 冲突内容拒绝
- 状态机流转

### 集成测试

- Catalog + npm 仓
- release record 成功与失败分支
- governance change 对 install target 解析的影响
