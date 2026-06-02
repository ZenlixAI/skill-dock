# Agent Skill Registry Catalog API 详细设计

## 文档定位

本文是 V1 主规范之外的服务端实现级文档，把 `Skill Catalog API` 收敛到可实现级别，覆盖：

- endpoint 列表
- request / response schema
- 分页与过滤
- 错误模型
- 幂等语义
- 权限矩阵

本文只讨论 `Catalog Registry API`。`Authoring / Packaging API` 继续留在 [API 与 CLI 契约](./2026-05-28-agent-skill-registry-api-cli-contracts.zh-CN.md) 和 [Skill Packager / Importer 详细设计](./2026-05-28-agent-skill-registry-skill-packager-importer.zh-CN.md) 中。

如果需要机器可读接口定义，见 [Catalog API OpenAPI](./2026-05-28-agent-skill-registry-catalog-api.openapi.yaml)。

## 设计目标

- 让 Catalog API 可直接指导后端实现
- 明确读接口与决策/写接口的边界
- 明确 npm-compatible 仓元数据与 Catalog 派生元数据的分层
- 让 CLI、平台后端和未来 UI 都能共享同一套接口契约

## 非目标

- 不定义 agent runtime 如何执行 skill
- 不定义 `Local Installer Adapter` 的本地 scope、winner 或 discoverable state 语义
- 不定义 npm-compatible 仓自身 API
- 不定义 OpenAPI 文件生成细节
- 不引入 project-level dependency model

## 基本约定

### Base Path

所有路径均以 `/api/v1` 为前缀。

示例：

- `GET /api/v1/skills`
- `GET /api/v1/skills/{packageName}/install-target`

### 命名约定

- 文档领域名使用 `package_name`
- HTTP JSON 字段使用 `packageName`
- path parameter 中的 scoped package 必须做 URL encode

示例：

- `@platform.skill/pdf-skill`
- path 中写作 `%40platform.skill%2Fpdf-skill`

### 通用格式

- request / response body 使用 `application/json`
- 时间字段统一使用 RFC 3339 UTC 字符串
- `version` 统一使用精确 semver 字符串
- V1 不接受 semver range

### 鉴权边界

V1 只定义角色授权，不把认证机制写死。

可接受的认证承载方式包括：

- Bearer token
- 平台 session
- 内部服务间身份

但无论使用哪种认证方式，授权语义统一只认：

- `reader`
- `publisher`

## 统一错误模型

所有非 2xx 响应统一返回：

```json
{
  "error": {
    "code": "skill_not_found",
    "message": "Skill not found.",
    "details": {
      "packageName": "@platform.skill/pdf-skill"
    },
    "requestId": "req_01hx..."
  }
}
```

字段说明：

- `code`
  稳定机读错误码
- `message`
  面向日志和调试的人类可读描述
- `details`
  结构化上下文，字段按 endpoint 场景扩展
- `requestId`
  用于日志追踪

### 通用错误码

- `invalid_request`
- `unauthorized`
- `forbidden`
- `skill_not_found`
- `version_not_found`
- `skill_retired`
- `version_yanked`
- `not_resolvable_by_policy`
- `invalid_state_transition`
- `registry_version_not_found`
- `registry_unavailable`
- `bundle_validation_failed`
- `immutable_release_conflict`
- `internal_error`

## 分页与列表约定

V1 所有列表接口统一使用 cursor 分页。

请求参数：

- `pageSize`
  默认 `20`
- `cursor`
  可选

约束：

- `pageSize` 最大 `100`
- 未传 `cursor` 表示第一页

统一响应形态：

```json
{
  "items": [],
  "pageInfo": {
    "nextCursor": "cursor_abc",
    "hasNextPage": true
  }
}
```

## 资源模型

### SkillSummary

```json
{
  "packageName": "@platform.skill/pdf-skill",
  "displayName": "PDF Skill",
  "description": "PDF processing workflow",
  "sourceType": "imported",
  "status": "active",
  "latestVersion": "1.2.0",
  "latestVersionStatus": "published",
  "updatedAt": "2026-05-28T08:30:00Z"
}
```

### SkillDetail

```json
{
  "packageName": "@platform.skill/pdf-skill",
  "displayName": "PDF Skill",
  "description": "PDF processing workflow",
  "sourceType": "imported",
  "status": "active",
  "latestVersion": "1.2.0",
  "latestVersionStatus": "published",
  "createdAt": "2026-05-20T02:00:00Z",
  "updatedAt": "2026-05-28T08:30:00Z"
}
```

### SkillVersionSummary

```json
{
  "packageName": "@platform.skill/pdf-skill",
  "version": "1.2.0",
  "versionStatus": "published",
  "installPolicy": "resolvable",
  "publishedAt": "2026-05-28T08:20:00Z"
}
```

### SkillVersionDetail

```json
{
  "packageName": "@platform.skill/pdf-skill",
  "version": "1.2.0",
  "versionStatus": "published",
  "installPolicy": "resolvable",
  "registry": {
    "npmRegistry": "https://npm.internal.example.com",
    "tarballUrl": "https://npm.internal.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz",
    "integrity": "sha512-...",
    "shasum": "abc123",
    "publishedAt": "2026-05-28T08:20:00Z",
    "distTags": ["latest"]
  },
  "bundle": {
    "skillManifestDigest": "sha256:...",
    "components": ["commands", "scripts"]
  }
}
```

### GovernanceRecord

```json
{
  "id": "gov_01hx...",
  "targetType": "version",
  "packageName": "@platform.skill/pdf-skill",
  "version": "1.2.0",
  "action": "yank",
  "reason": "security issue",
  "replacementRecommendation": "1.2.1",
  "changedBy": "user_123",
  "changedAt": "2026-05-28T09:00:00Z"
}
```

### AuditEventRecord

```json
{
  "id": "aud_01hx...",
  "requestId": "req_01hx...",
  "traceId": "trace_01hx...",
  "action": "release.record",
  "actor": "svc-publisher",
  "role": "publisher",
  "authMechanism": "configured_token",
  "authDecision": "allowed",
  "endpoint": "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0/release-record",
  "method": "PUT",
  "packageName": "@platform.skill/pdf-skill",
  "version": "1.2.0",
  "outcome": "succeeded",
  "statusCode": 201,
  "details": {
    "created": true
  },
  "createdAt": "2026-05-28T09:10:00Z"
}
```

## 读接口

### GET /api/v1/skills

用途：

- 统一 skill 列表与搜索入口

请求参数：

- `q`
- `package_name`
- `source_type`
- `include_retired`
- `pageSize`
- `cursor`
- `sort`

`sort` 取值：

- `relevance`
- `updated_at_desc`
- `package_name_asc`

规则：

- 当 `q` 存在时，默认 `sort=relevance`
- 当 `q` 不存在时，默认 `sort=updated_at_desc`
- `include_retired` 默认 `false`

成功响应：

```json
{
  "items": [
    {
      "packageName": "@platform.skill/pdf-skill",
      "displayName": "PDF Skill",
      "description": "PDF processing workflow",
      "sourceType": "imported",
      "status": "active",
      "latestVersion": "1.2.0",
      "latestVersionStatus": "published",
      "updatedAt": "2026-05-28T08:30:00Z"
    }
  ],
  "pageInfo": {
    "nextCursor": null,
    "hasNextPage": false
  }
}
```

状态码：

- `200` 成功
- `400` query 参数非法
- `401` 未认证
- `403` 无权限

### GET /api/v1/skills/search

用途：

- 兼容别名

规则：

- V1 推荐把它实现为 `GET /api/v1/skills` 的同语义别名
- 新客户端优先使用 `GET /api/v1/skills`
- 它不引入新的搜索能力或新的返回结构

### GET /api/v1/skills/curated

用途：

- 返回平台精选或推荐列表

请求参数：

- `pageSize`
- `cursor`

成功响应：

```json
{
  "items": [
    {
      "packageName": "@platform.skill/pdf-skill",
      "displayName": "PDF Skill",
      "description": "PDF processing workflow",
      "sourceType": "imported",
      "status": "active",
      "latestVersion": "1.2.0",
      "latestVersionStatus": "published",
      "updatedAt": "2026-05-28T08:30:00Z",
      "curationReason": "featured"
    }
  ],
  "pageInfo": {
    "nextCursor": null,
    "hasNextPage": false
  }
}
```

### GET /api/v1/skills/{packageName}

用途：

- 获取单个 skill 详情

成功响应：

```json
{
  "packageName": "@platform.skill/pdf-skill",
  "displayName": "PDF Skill",
  "description": "PDF processing workflow",
  "sourceType": "imported",
  "status": "active",
  "latestVersion": "1.2.0",
  "latestVersionStatus": "published",
  "createdAt": "2026-05-20T02:00:00Z",
  "updatedAt": "2026-05-28T08:30:00Z"
}
```

状态码：

- `200` 成功
- `404` `skill_not_found`

### GET /api/v1/skills/{packageName}/versions

用途：

- 获取某个 skill 的版本列表

请求参数：

- `include_deprecated`
- `include_yanked`
- `pageSize`
- `cursor`

默认规则：

- `include_deprecated=true`
- `include_yanked=false`

成功响应：

```json
{
  "items": [
    {
      "packageName": "@platform.skill/pdf-skill",
      "version": "1.2.0",
      "versionStatus": "published",
      "installPolicy": "resolvable",
      "publishedAt": "2026-05-28T08:20:00Z"
    }
  ],
  "pageInfo": {
    "nextCursor": null,
    "hasNextPage": false
  }
}
```

状态码：

- `200` 成功
- `404` `skill_not_found`

### GET /api/v1/skills/{packageName}/versions/{version}

用途：

- 获取某个 skill version 的详细元数据

成功响应：

```json
{
  "packageName": "@platform.skill/pdf-skill",
  "version": "1.2.0",
  "versionStatus": "published",
  "installPolicy": "resolvable",
  "registry": {
    "npmRegistry": "https://npm.internal.example.com",
    "tarballUrl": "https://npm.internal.example.com/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz",
    "integrity": "sha512-...",
    "shasum": "abc123",
    "publishedAt": "2026-05-28T08:20:00Z",
    "distTags": ["latest"]
  },
  "bundle": {
    "skillManifestDigest": "sha256:...",
    "components": ["commands", "scripts"]
  }
}
```

状态码：

- `200` 成功
- `404` `skill_not_found`
- `404` `version_not_found`

### GET /api/v1/governance-changes

用途：

- 查询治理审计历史

请求参数：

- `package_name`
- 可选 `version`
- 可选 `target_type`
- `pageSize`
- `cursor`

成功响应：

```json
{
  "items": [
    {
      "id": "gov_01hx...",
      "targetType": "version",
      "packageName": "@platform.skill/pdf-skill",
      "version": "1.2.0",
      "action": "yank",
      "reason": "security issue",
      "replacementRecommendation": "1.2.1",
      "changedBy": "user_123",
      "changedAt": "2026-05-28T09:00:00Z"
    }
  ],
  "pageInfo": {
    "nextCursor": null,
    "hasNextPage": false
  }
}
```

状态码：

- `200` 成功
- `400` 请求过滤条件非法

### GET /api/v1/audit-events

用途：

- 查询运行时审计事件
- 支持按调用动作、主体、鉴权决策和坐标过滤

请求参数：

- 可选 `action`
- 可选 `actor`
- 可选 `auth_decision`
- `package_name`
- 可选 `version`
- 可选 `outcome`
- `pageSize`
- `cursor`

字段规则：

- `auth_decision` 取值为 `allowed` 或 `denied`
- 对 package-scoped principal，仍可要求显式提供 `package_name`

成功响应：

```json
{
  "items": [
    {
      "id": "aud_01hx...",
      "requestId": "req_01hx...",
      "traceId": "trace_01hx...",
      "action": "release.record",
      "actor": "svc-publisher",
      "role": "publisher",
      "authMechanism": "configured_token",
      "authDecision": "allowed",
      "endpoint": "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0/release-record",
      "method": "PUT",
      "packageName": "@platform.skill/pdf-skill",
      "version": "1.2.0",
      "outcome": "succeeded",
      "statusCode": 201,
      "details": {
        "created": true
      },
      "createdAt": "2026-05-28T09:10:00Z"
    }
  ],
  "pageInfo": {
    "nextCursor": null,
    "hasNextPage": false
  }
}
```

状态码：

- `200` 成功
- `400` 请求过滤条件非法
- `401` 未认证
- `403` 已认证但无权读取

## 决策与写接口

### GET /api/v1/skills/{packageName}/install-target

用途：

- 把安装请求解析成可下载、可校验的 install target

请求参数：

- path `packageName`
- 可选 query `version`

字段规则：

- `packageName` 必填
- `version` 可选
- 未传 `version` 表示默认 latest 解析
- 若传 `version`，必须是精确 semver

成功响应：

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

状态码：

- `200` 成功
- `404` `skill_not_found`
- `404` `version_not_found`
- `409` `skill_retired`
- `409` `version_yanked`
- `409` `not_resolvable_by_policy`
- `422` `invalid_request`

说明：

- 该接口是“解析”而不是“安装”
- 它不在服务端落任何安装状态
- 它是安装决策的主接口；客户端不应通过版本列表接口自行重建 latest 解析、生命周期过滤或 install policy 判断
- 成功响应返回 `versionStatus`，而不是 `deprecated` / `yanked` 布尔组合，以保持与 version 资源模型一致；显式安装 deprecated version 时可返回 `deprecated`

失败响应沿用统一错误模型；若存在替代推荐，应放入 `error.details.replacementRecommendation`。

### PUT /api/v1/skills/{packageName}/versions/{version}/release-record

用途：

- 为已经存在于 npm-compatible 仓的 tarball 创建 Catalog version 记录

V1 规则：

- `packageName + version` 是逻辑主键
- 默认从系统配置的内部 npm-compatible 仓读取元数据
- 不上传 tarball 文件流

成功响应：

首次登记成功返回 `201`：

```json
{
  "packageName": "@platform.skill/pdf-skill",
  "version": "1.2.0",
  "created": true,
  "idempotentReplay": false,
  "versionStatus": "published",
  "installPolicy": "resolvable"
}
```

重复登记且内容一致返回 `200`：

```json
{
  "packageName": "@platform.skill/pdf-skill",
  "version": "1.2.0",
  "created": false,
  "idempotentReplay": true,
  "versionStatus": "published",
  "installPolicy": "resolvable"
}
```

状态码：

- `200` 幂等重放成功
- `201` 首次登记成功
- `404` `registry_version_not_found`
- `409` `immutable_release_conflict`
- `422` `bundle_validation_failed`
- `503` `registry_unavailable`

冲突判定：

- 若同一 `packageName + version` 已存在，且 registry 元数据与 bundle 扫描结果一致，则按幂等成功处理
- 若同一 `packageName + version` 已存在，但内容不一致，则硬拒绝

### POST /api/v1/skills/{packageName}/governance-changes

用途：

- 对 skill 执行治理动作，并写入审计记录

请求体：

```json
{
  "action": "retire",
  "reason": "merged into successor skill"
}
```

字段规则：

- path `packageName` 必填
- `action` 取值：`retire`
- `reason` 必填

### POST /api/v1/skills/{packageName}/versions/{version}/governance-changes

用途：

- 对 skill version 执行治理动作，并写入审计记录

请求体：

```json
{
  "action": "yank",
  "reason": "security issue",
  "replacementRecommendation": "1.2.1"
}
```

字段规则：

- path `packageName` 必填
- path `version` 必填
- `action` 取值：`deprecate`、`yank`
- `reason` 必填

成功响应：

```json
{
  "record": {
    "id": "gov_01hx...",
    "targetType": "version",
    "packageName": "@platform.skill/pdf-skill",
    "version": "1.2.0",
    "action": "yank",
    "reason": "security issue",
    "replacementRecommendation": "1.2.1",
    "changedBy": "user_123",
    "changedAt": "2026-05-28T09:00:00Z"
  },
  "resultingState": {
    "versionStatus": "yanked"
  }
}
```

状态码：

- `201` 治理动作成功
- `404` `skill_not_found`
- `404` `version_not_found`
- `409` `invalid_state_transition`
- `422` `invalid_request`

幂等语义：

- 该接口默认不是幂等重放接口
- 对已经处于目标状态的对象再次提交同类动作，返回 `409 invalid_state_transition`
- 如果未来需要抗网络重试，可再引入 `Idempotency-Key`

## 状态机与接口映射

### skill 级

- `active -> retired`

### version 级

- `published -> deprecated`
- `published -> yanked`
- `deprecated -> yanked`

非法流转示例：

- `retired -> active`
- `yanked -> published`
- 对已经 `yanked` 的 version 再次 `yank`

## 权限矩阵

| Endpoint | reader | publisher |
| --- | --- | --- |
| `GET /api/v1/skills` | allowed | allowed |
| `GET /api/v1/skills/search` | allowed | allowed |
| `GET /api/v1/skills/curated` | allowed | allowed |
| `GET /api/v1/skills/{packageName}` | allowed | allowed |
| `GET /api/v1/skills/{packageName}/versions` | allowed | allowed |
| `GET /api/v1/skills/{packageName}/versions/{version}` | allowed | allowed |
| `GET /api/v1/governance-changes` | allowed | allowed |
| `GET /api/v1/skills/{packageName}/install-target` | allowed | allowed |
| `PUT /api/v1/skills/{packageName}/versions/{version}/release-record` | forbidden | allowed |
| `POST /api/v1/skills/{packageName}/governance-changes` | forbidden | allowed |
| `POST /api/v1/skills/{packageName}/versions/{version}/governance-changes` | forbidden | allowed |

## 真相源分层

下列字段以 npm-compatible 仓为源：

- `tarballUrl`
- `integrity`
- `shasum`
- `publishedAt`
- `distTags`

下列字段以 Catalog 为源：

- `displayName`
- `description`
- `status`
- `versionStatus`
- `installPolicy`
- `skillManifestDigest`
- `components`
- 治理审计记录

## 与 CLI 的对应关系

- `skills find`
  读 `GET /api/v1/skills`
- `skills inspect`
  不走 Catalog，走本地 packager
- `skills publish`
  先 `npm publish`，后 `PUT /api/v1/skills/{packageName}/versions/{version}/release-record`
- `skills add`
  先 `GET /api/v1/skills/{packageName}/install-target`，再本地下载并落盘
- `skills update`
  对当前 `package_name` 重新发起 `GET /api/v1/skills/{packageName}/install-target`
- 运维 / 平台审计
  读 `GET /api/v1/audit-events`

## V1 刻意不做的 API

- 不提供直接上传 tarball 到 Catalog 的接口
- 不提供基于 `owner/repo` 的 source-first 安装接口
- 不提供 project dependency manifest / lockfile 相关接口
- 不把 npm-compatible 仓代理成一套 Catalog 子接口

## 测试建议

### 契约测试

- 统一错误包结构
- cursor 分页稳定性
- path 中 scoped package 的 URL encode 行为

### 行为测试

- retired skill 的 install target 解析拒绝
- deprecated version 的显式安装允许
- yanked version 的默认解析拒绝
- release-record 幂等成功与冲突分支
- governance 非法状态流转拒绝
