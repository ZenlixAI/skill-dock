# 批量初始化已有 Skills

这个脚本用于把一批已经存在的 skill 源目录或 git 源，一次性导入到当前 Catalog。

它复用现有生产链路：

1. `POST /api/v1/skill-packaging/prepare`
2. `POST /api/v1/skill-packaging/validate`
3. `POST /api/v1/skill-packaging/pack`
4. 本地 `npm publish <bundle>`
5. `PUT /api/v1/skills/{packageName}/versions/{version}/release-record`

脚本入口：

```bash
uv run python tools/batch-import/batch_import_skills.py --help
```

示例 manifest：

```bash
tools/batch-import/example.json
```

## 前提

- 已启动 Catalog，并且当前凭据有 packaging / release-record 权限
- 当前机器已安装 `uv`
- 当前机器可以执行 `npm publish`
- npm 凭据已通过 `.npmrc`、token 或企业内部登录态准备好

重要限制：

- 当 entry 使用本地 `path` 时，脚本会调用 `source.type=local`
- 这要求 Catalog 服务器能看到同一份文件系统路径
- 如果 Catalog 不在同一台机器，改用 manifest 里的 `source.type=git`

默认导入行为：

- 在 manifest / 扫描结果已经提供导入所需 target 信息后，默认路径不再要求外部 `local` / `git` 源目录预先手写 bundle 用的 `package.json` 或 capability 占位内容
- `prepare` 默认会在临时 workspace 内重写面向 bundle 的 `package.json`，使其匹配本次导入的 target / metadata，并在需要时合成缺失的 capability 占位内容
- 这些重写和合成都只发生在打包会话的临时目录，不会回写原始源目录

如果你不想接受这种规范化，而是要显式拒绝还不满足 publishable bundle 契约的源目录，请传 `--strict`。

## 方式一：显式 manifest

```bash
export SKILLS_REGISTRY_URL=http://127.0.0.1:18080
export SKILLS_REGISTRY_TOKEN=publisher-secret

uv run python tools/batch-import/batch_import_skills.py \
  --manifest tools/batch-import/example.json \
  --npm-registry http://127.0.0.1:8081/repository/skills-npm \
  --continue-on-error
```

如果你已经按 [docs/local-verdaccio-e2e.zh-CN.md](/Users/ultraman/workspace/skill_registry/docs/local-verdaccio-e2e.zh-CN.md) 启动了本地 Verdaccio，也可以直接用：

```bash
uv run python tools/batch-import/batch_import_skills.py \
  --manifest tools/batch-import/example.json \
  --npm-registry http://127.0.0.1:4873 \
  --continue-on-error
```

或直接：

```bash
make batch-import-verdaccio MANIFEST=/abs/path/to/import.json
```

严格模式示例：

```bash
uv run python tools/batch-import/batch_import_skills.py \
  --manifest tools/batch-import/example.json \
  --npm-registry http://127.0.0.1:4873 \
  --strict
```

或：

```bash
make batch-import-verdaccio MANIFEST=/abs/path/to/import.json STRICT=1
```

manifest 最小格式：

```json
{
  "skills": [
    {
      "path": "/abs/path/to/pdf-skill",
      "packageName": "@platform.skill/pdf-skill",
      "version": "1.2.0",
      "displayName": "PDF Skill",
      "description": "PDF processing workflow"
    }
  ]
}
```

支持字段：

- `path`: 本地 skill 目录
- `source`: 非本地源；当前支持 `git`、`platform_authoring`
- `packageName`
- `version`
- `displayName`
- `description`
- `sourceType`
- `publishRegistry`
- `npmCwd`: `npm publish` 的执行目录，默认本地 skill 目录；用于让 `.npmrc` 生效

## 方式二：扫描目录快速导入

如果一批 skill 已经放在某个目录下，而且每个 skill 至少有根级 `SKILL.md`，可以直接扫目录：

```bash
uv run python tools/batch-import/batch_import_skills.py \
  --root-dir /abs/path/to/skills \
  --recursive \
  --default-version 0.1.0 \
  --npm-registry http://127.0.0.1:8081/repository/skills-npm
```

扫描模式下：

- `displayName` / `description` 从 `SKILL.md` frontmatter 读取
- 若本地已有 `package.json`，优先使用其中的 `name` / `version`
- 若没有 `package.json`，脚本会推导
  - `packageName`: `@platform.skill/<目录名slug>`
  - `version`: `--default-version`，默认 `0.1.0`
- 默认模式下，后续 packaging `prepare` 仍会在临时 workspace 内重写 bundle 用的 `package.json`，并在需要时合成缺失的 capability 占位内容
- 若加 `--strict`，则不会接受需要这类规范化的源目录；prepare / validate 会直接暴露缺失项

导入前建议先看计划：

```bash
uv run python tools/batch-import/batch_import_skills.py --root-dir /abs/path/to/skills --dry-run
```

## 和真实 local npm registry 配合

脚本不依赖某个特定 npm 实现，只要满足下面两点即可：

1. `npm publish` 能成功推送 bundle
2. Catalog `release-record` 能从同一个 registry 读到 package metadata

因此可以接：

- Nexus
- Verdaccio
- Artifactory
- GitLab Package Registry
- 其他 npm-compatible registry

通常要保证两处一致：

- 脚本侧 `--npm-registry`
- Catalog 启动时的 `SKILL_REGISTRY_NPM_REGISTRY_URL`

如果两边不一致，`release-record` 可能会因为解析到的 registry metadata 不匹配而失败。

补充：

- 本地 Verdaccio 默认地址是 `http://127.0.0.1:4873`
- `http://127.0.0.1:8081/repository/skills-npm` 这类地址更适合 Nexus 一类仓库示例
- 如果你想先显式准备 Python 环境，可以先执行 `make python-sync`
