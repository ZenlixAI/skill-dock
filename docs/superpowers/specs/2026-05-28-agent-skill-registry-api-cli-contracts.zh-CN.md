# API 与 CLI 契约

## 文档定位

本文是 V1 主规范之外的对外契约文档，定义 CLI 命令语义、Catalog HTTP API、可选 packaging API 以及文件契约。

其中：

- Catalog Registry API 的实现级细节见 [Catalog API 详细设计](./2026-05-28-agent-skill-registry-catalog-api-detailed-design.zh-CN.md)
- 本文保留 Registry Core 的对外契约、消费侧 CLI 语义、Packaging API 和跨接口边界约束

如果只关心本地安装 adapter 的推荐语义，见：

- [Local Installer Adapter 详细设计](./2026-05-28-agent-skill-registry-local-installer-adapter.zh-CN.md)

## 命名约定

- 文档中的领域概念统一写作 `package_name`
- HTTP JSON 字段使用 API 风格，写作 `packageName`
- 两者表达的是同一个公开分发身份，只是呈现层命名风格不同

## CLI 命令

以下命令属于消费侧 CLI 能力。

其中：

- `find` / `publish` / `inspect` 更直接面向 Registry Core
- `add` / `list` / `update` / `remove` 依赖一个本地 `Local Installer Adapter`

- `skills add`
- `skills find`
- `skills list`
- `skills remove`
- `skills update`
- `skills init`
- `skills publish`
- `skills inspect`

通用约束：

- CLI 位置参数只接受 `package_name`
- 不接受 `owner/repo`
- `--version` 只接受精确版本
- `--registry` 指向 Catalog entry point，而不是让用户直接管理 npm 仓

实现状态补充：

- 当前 V1 只保留 registry-resolved 安装作为正式公开契约
- CLI 不再保留 `experimental_install`、`experimental_sync` 或 `node_modules` 发现入口

### skills init

- 本地 authoring scaffold
- 初始化 `SKILL.md`
- 初始化 `package.json`
- 初始化基础目录结构

不要求调用 Catalog API；如需模板或命名策略，可额外调用平台 authoring API。

### skills inspect

- 调用或复用 packager 校验能力
- 输出本地目录检查结果

### skills publish

- 调用 `prepare + validate + pack`
- 调用 `npm publish`
- 调用 `PUT /api/v1/skills/{packageName}/versions/{version}/release-record`

说明：

- `skills publish` 不应把 `npm publish` 的响应体当作 Catalog 最终落库依据
- 更稳妥的流程是：publish 成功后，再由发布方或 Catalog 查询 registry 版本元数据

### skills add

- 调用 install target 解析接口
- 下载 bundle
- 安装到本地安装器约定的可发现目录

### skills list

- 若实现多 scope，则列出各 scope 安装状态
- 若实现多 scope，则标记 winner

### skills update

- 重新解析当前安装 package 的默认可安装版本

### skills remove

- 删除本地副本

## Catalog HTTP API

### 读接口

- `GET /api/v1/skills`
- `GET /api/v1/skills/curated`
- `GET /api/v1/skills/{packageName}`
- `GET /api/v1/skills/{packageName}/versions`
- `GET /api/v1/skills/{packageName}/versions/{version}`

`GET /api/v1/skills` 支持：

- `q`
- `package_name`
- `source_type`
- `include_retired`

兼容别名：

- `GET /api/v1/skills/search`
  与 `GET /api/v1/skills` 同语义，仅兼容保留

## 决策与写接口

### GET /api/v1/skills/{packageName}/install-target

请求示例：

`GET /api/v1/skills/%40platform.skill%2Fpdf-skill/install-target?version=1.2.0`

成功返回示例：

```json
{
  "packageName": "@platform.skill/pdf-skill",
  "version": "1.2.0",
  "npmRegistry": "https://npm.internal.example.com",
  "integrity": "sha512-...",
  "tarballUrl": "https://npm.internal.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz",
  "skillManifestDigest": "sha256:...",
  "versionStatus": "published"
}
```

说明：

- 该接口是安装决策接口，而不是版本列表接口
- CLI 不应通过 `GET /api/v1/skills/{packageName}/versions` 自行重建 latest 解析或治理判断
- 成功响应中的 `versionStatus` 默认 latest 解析通常为 `published`，显式安装 deprecated version 时可为 `deprecated`
- 下载、解包和本地落盘由 `Local Installer Adapter` 负责，不属于 Catalog 远端状态

失败返回使用统一错误模型：

```json
{
  "error": {
    "code": "version_yanked",
    "message": "Requested version is yanked.",
    "details": {
      "packageName": "@platform.skill/pdf-skill",
      "version": "1.2.0",
      "replacementRecommendation": "1.2.1"
    },
    "requestId": "req_01hx..."
  }
}
```

稳定错误码：

- `skill_not_found`
- `skill_retired`
- `version_not_found`
- `version_yanked`
- `not_resolvable_by_policy`

### PUT /api/v1/skills/{packageName}/versions/{version}/release-record

用途：

- 为已存在于 npm 仓的 bundle 创建 Catalog 记录
- 不上传 tarball 文件流
- 不依赖 `npm publish` 响应体本身作为唯一元数据来源

请求路径示例：

`PUT /api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0/release-record`

服务端最少应补充读取的 registry 源数据：

- `dist.tarball`
- `dist.integrity`
- `dist.shasum`
- 可选 `dist-tags`
- 可选 `time[version]`

说明：

- `packageName + version` 来自 path parameter，是 release record 的逻辑主键
- tarball 地址与完整性信息来自 npm-compatible 仓
- `skillManifestDigest`、`components`、`installPolicy` 仍由 Catalog 自己生成或判定

### POST /api/v1/skills/{packageName}/governance-changes

用于 skill 级治理事件。

请求示例：

```json
{
  "action": "retire",
  "reason": "merged into successor skill"
}
```

### POST /api/v1/skills/{packageName}/versions/{version}/governance-changes

用于 version 级治理事件。

请求示例：

```json
{
  "action": "yank",
  "reason": "security issue"
}
```

## Packaging API

V1 不要求 `Skill Packager / Importer` 一定通过 HTTP 暴露。但如果平台要服务化 packaging，建议它属于 authoring / packaging API，而不是 Catalog API。

V1 推荐同步 API，内部仍按 `prepare -> validate -> pack` 三阶段推进，但不默认抽象成异步 job。

建议接口：

- `POST /api/v1/skill-packaging/prepare`
- `POST /api/v1/skill-packaging/validate`
- `POST /api/v1/skill-packaging/pack`

### POST /api/v1/skill-packaging/prepare

用途：

- 生成规范化工作目录和最小元数据

请求示例：

```json
{
  "source": {
    "type": "git",
    "location": "https://github.com/org/repo?path=skills/pdf",
    "ref": "main"
  },
  "target": {
    "packageName": "@platform.skill/pdf-skill",
    "version": "1.2.0"
  },
  "metadata": {
    "displayName": "PDF Skill",
    "description": "PDF processing workflow",
    "sourceType": "imported"
  },
  "options": {
    "publishRegistry": "https://npm.internal.example.com"
  }
}
```

成功返回示例：

```json
{
  "workspacePath": "/tmp/skill-packager/pdf-skill",
  "packageName": "@platform.skill/pdf-skill",
  "version": "1.2.0",
  "warnings": [],
  "warningDetails": [],
  "importReport": {
    "sourceType": "imported",
    "strictMode": false,
    "normalized": false,
    "generatedPackageJson": false,
    "generatedCapabilityPlaceholder": false,
    "rewrotePackageIdentity": false,
    "generatedPaths": [],
    "warningCodes": []
  },
  "errors": []
}
```

### POST /api/v1/skill-packaging/validate

用途：

- 对已准备好的工作目录做结构化校验

响应示例：

```json
{
  "ok": true,
  "warnings": [],
  "warningDetails": [],
  "errors": []
}
```

### POST /api/v1/skill-packaging/pack

用途：

- 生成最终 bundle 产物元数据

成功响应示例：

```json
{
  "workspacePath": "/tmp/skill-packager/pdf-skill",
  "bundlePath": "/tmp/skill-packager/pdf-skill-1.2.0.tgz",
  "packageName": "@platform.skill/pdf-skill",
  "version": "1.2.0",
  "manifestDigest": "sha256:...",
  "components": ["commands", "scripts"],
  "warnings": [],
  "warningDetails": [],
  "errors": []
}
```

补充约束：

- `warnings` 保持稳定机读 code，供脚本、批量导入和审计使用
- `warningDetails` 提供人类可读展示内容，供 CLI 和未来平台 UI 直接渲染
- `importReport` 提供结构化导入状态，供 CLI、batch import 和未来平台 authoring flow 直接消费

### Packaging API 边界

该 API 只负责：

- prepare
- validate
- pack

它不负责：

- `npm publish`
- `PUT /api/v1/skills/{packageName}/versions/{version}/release-record`
- install target 解析
- 本地安装落盘

### 同步 API 适用前提

- 打包路径是轻量、秒级操作
- 不要求前端轮询进度
- 不要求取消 / 重试 / 长任务编排

### 异步 job API 作为后续扩展

若后续接入大型外部源拉取、扫描、签名、审核或进度展示，可扩展为异步接口：

- `POST /api/v1/skill-packaging-jobs`
- `GET /api/v1/skill-packaging-jobs/{id}`
- `GET /api/v1/skill-packaging-jobs/{id}/artifacts`
- `POST /api/v1/skill-packaging-jobs/{id}/cancel`

## 文件契约

### SKILL.md

- 作者说明与最小元数据
- 最小字段：`name`、`description`

### package.json

- npm 分发身份与版本
- 关键字段：`name`、`version`
- 可选 `publishConfig.registry`

### plugin.json

- 兼容附件
- 不属于 Catalog 一等领域对象

## skills.sh 兼容边界

完整说明见：

- [与 skills.sh 的兼容性边界说明](./2026-05-28-agent-skill-registry-skills-sh-compatibility.zh-CN.md)

兼容：

- `SKILL.md`
- 命令语言心智
- 只读发现接口风格

不兼容：

- source-first 安装协议
- `owner/repo` 作为安装目标
- repo/source 作为公开身份

## 版本兼容策略

### HTTP API

- 通过新增字段前向兼容
- 不随意重命名已有字段

### CLI

- 新能力优先通过新增 flags 或子命令表达
- 既有命令语义保持稳定

### 文件格式

- `SKILL.md` 保持最小稳定字段
- bundle 目录允许可选文件增量扩展
