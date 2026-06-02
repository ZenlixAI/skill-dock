# Skill Packager / Importer 详细设计

## 文档定位

本文是 V1 主规范之外的 bundle 生产侧补充文档，定义 `Skill Packager / Importer` 的职责、输入输出、bundle 生成规则、校验器和错误模型。

## 设计目标

- 统一 CLI scaffold、平台代创建、外部导入三条路径的产物
- 把 `package.json` 生成逻辑集中到一处
- 让 bundle 在进入 publish 前就完成结构与安全校验

## 非目标

- 不负责 install target 解析
- 不负责 published release 登记
- 不负责运行时脚本执行

## 输入模式

### 本地 scaffold

- `skills init`
- 作者本地维护目录

### 平台创建

- 平台 UI 表单
- 结构化 authoring payload

### 外部导入

- GitHub repo 某个目录
- 外部 git 仓目录
- 已存在的本地 skill 源目录

## 逻辑接口

- `prepare(input) -> normalized_workspace`
- `validate(workspace) -> validation_result`
- `pack(workspace) -> bundle_artifact`

### prepare

职责：

- 拉取或生成工作目录
- 规范化 `SKILL.md`
- 生成或重写 `package.json`
- 整理能力目录

输入示例：

```json
{
  "source": {
    "type": "git|local|platform_authoring",
    "location": "https://github.com/org/repo?path=skills/pdf",
    "ref": "main"
  },
  "packageName": "@platform.skill/pdf-skill",
  "version": "1.2.0",
  "displayName": "PDF Skill",
  "description": "PDF processing workflow",
  "sourceType": "imported",
  "publishRegistry": "https://npm.internal.example.com"
}
```

### validate

职责：

- 结构校验
- 命名校验
- 安全校验
- `single-skill bundle` 校验

### pack

职责：

- 生成最终 `.tgz`
- 计算 manifest digest

输出示例：

```json
{
  "workspacePath": "/tmp/skill-packager/pdf-skill",
  "packageName": "@platform.skill/pdf-skill",
  "version": "1.2.0",
  "bundlePath": "/tmp/skill-packager/pdf-skill-1.2.0.tgz",
  "manifestDigest": "sha256:...",
  "components": ["commands", "scripts"],
  "warnings": [],
  "errors": []
}
```

## Bundle 目录规范

```text
SKILL.md
package.json
plugin.json
commands/
agents/
hooks/
.mcp.json
scripts/
docs/
```

约束：

- 必须且只能有一个根级 `SKILL.md`
- 必须有 `package.json`
- 必须至少包含一种能力目录或能力配置
- 不允许嵌套 `SKILL.md`
- 不允许路径穿越、绝对路径、符号链接、大小写冲突路径

## SKILL.md 规范化

最小要求：

- `name`
- `description`

规则：

- 平台创建场景可生成最小 `SKILL.md`
- 外部存在合法根级 `SKILL.md` 时优先复用
- 外部不存在合法 `SKILL.md` 时导入失败
- 不从 README 或 repo 名自动推导主描述

## package.json 生成规则

### 责任边界

`package.json` 属于分发工件，不属于 Catalog 状态文件。它可以由：

- `skills init`
- 平台后端
- 外部导入重组
- 迁移流程

生成。

### 最小字段

- `name`
- `version`
- 可选 `publishConfig.registry`

### 生成算法

1. `name` 始终以内部 `package_name` 为准
2. `version` 以本次内部发布版本为准
3. `publishConfig.registry` 由内部 npm 仓配置注入
4. 外部 `package.json` 仅作候选输入
5. 与内部身份冲突的字段必须重写

### 关于 private

- publishable bundle 不应设置 `private: true`
- 若要限制只发内部仓，使用 `publishConfig.registry`、`.npmrc` 或 CLI registry 配置

## 外部导入映射

### 身份映射

- 外部 repo 名不是最终公开身份
- 外部 npm 包名也不是最终公开身份
- 导入操作者必须显式确定内部 `package_name`

### 版本映射

- 可复用 `upstream_version`
- 也可指定内部 `version`
- 若发生重组或重打包，应使用新的内部 `package_name`

### 能力目录映射

- `commands/`、`agents/`、`hooks/`、`scripts/` 可在通过安全校验后原样纳入
- `plugin.json` 作为兼容附件纳入
- 多 skill 布局必须拆分后分别导入

## 校验器

### 结构校验

- 是否存在根级 `SKILL.md`
- 是否存在多个 `SKILL.md`
- 是否存在嵌套 `SKILL.md`
- 是否存在至少一种能力目录或配置

### 命名校验

- `package_name` 是否合法
- 是否满足 `@scope.skill/name`
- `version` 是否为精确版本号

### 安全校验

- 路径穿越
- 绝对路径
- 符号链接
- 大小写冲突

## 错误模型

- `missing_skill_md`
- `multiple_skill_md`
- `nested_skill_md`
- `invalid_package_name`
- `invalid_version`
- `unsafe_path_entry`
- `missing_capability_content`
- `package_version_conflict`

## 与其他组件的接口

### 与 CLI

- `skills init`: 主要使用 `prepare`
- `skills inspect`: 使用 `prepare + validate`
- `skills publish`: 使用 `prepare + validate + pack`

### 与平台后端

若服务化，建议属于 authoring / packaging API，而不是 Catalog API，例如：

- `POST /api/v1/skill-packaging/prepare`
- `POST /api/v1/skill-packaging/validate`
- `POST /api/v1/skill-packaging/pack`

V1 推荐同步 API，而不是异步 job API。原因是当前打包路径预期应保持轻量、秒级完成，同步请求更利于实现和联调。

### 同步 API 请求模型

建议 `pack` 请求示例：

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

### 同步 API 响应模型

建议 `pack` 成功响应示例：

```json
{
  "packageName": "@platform.skill/pdf-skill",
  "version": "1.2.0",
  "workspacePath": "/tmp/skill-packager/pdf-skill",
  "bundlePath": "/tmp/skill-packager/pdf-skill-1.2.0.tgz",
  "manifestDigest": "sha256:...",
  "components": ["commands", "scripts"],
  "warnings": [],
  "errors": []
}
```

同步响应建议至少包含：

- `workspacePath`
- `bundlePath`
- `packageName`
- `version`
- `manifestDigest`
- `components`

### prepare / validate / pack 语义

- `prepare`
  返回规范化工作目录和最小元数据结果
- `validate`
  返回结构化校验结果
- `pack`
  返回最终 bundle 产物元数据

### 错误返回

同步错误返回建议：

- `warnings`
- `errors`
- 可选部分结果，例如已生成的工作目录摘要

### 幂等与权限

建议：

- 同步接口支持调用方自带 `idempotencyKey`
- 平台端可按 source + target 去重短时间内的重复请求
- 调用 packaging API 需要 `publisher` 级别或 authoring 权限

### 异步 job API 作为后续扩展

若后续出现以下情况，再演进为异步 job API 更合理：

- 需要远程拉取大型外部源
- 需要引入扫描、签名、审核等长耗时步骤
- 平台 UI 需要进度展示、取消和重试

到那时可扩展为：

- `POST /api/v1/skill-packaging-jobs`
- `GET /api/v1/skill-packaging-jobs/{id}`
- `GET /api/v1/skill-packaging-jobs/{id}/artifacts`
- `POST /api/v1/skill-packaging-jobs/{id}/cancel`

### 与发布链路

Packager 的成功输出是：

- bundle 工作目录
或
- `.tgz` 归档

后续仍需：

1. `npm publish`
2. `PUT /api/v1/skills/{packageName}/versions/{version}/release-record`

## 测试建议

### 单元测试

- `SKILL.md` 解析
- `package.json` 生成
- package name 校验
- version 校验
- 路径安全校验

### 集成测试

- git / local / platform_authoring 三类输入
- 导入已有外部 `package.json` 的目录
- 产出 `.tgz` 可被后续 publish
