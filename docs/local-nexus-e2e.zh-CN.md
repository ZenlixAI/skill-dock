# 本地 Kind + Nexus E2E

这份文档描述当前仓库唯一保留的本地真实 E2E 路径：

- `skills-web`
- `catalog`
- `postgres`
- `nexus`

全部运行在 `kind` 中。Makefile 只负责顶层环境拉起与部署，以及 CLI 的测试 / type-check / 包发布；终端交互类 `skills` 子命令仍直接走独立 CLI。

## 固定本地入口

- Web: <http://127.0.0.1:8080>
- Nexus UI / API: <http://127.0.0.1:8081>
- Catalog API: <http://127.0.0.1:18080>
- Nexus npm repository: <http://127.0.0.1:8081/repository/skills-npm/>
- Nexus `.npmrc`: `/private/tmp/skill-registry-nexus/.npmrc`

## 启动方式

```bash
cd <repo-root>/skill_registry
make dev
```

`make dev` 会完成：

1. 构建本地 `catalog` / `skills-web` 镜像
2. 确保 `kind` 集群存在
3. 把镜像 load 进集群
4. apply 开发清单
5. 初始化集群内 Nexus 仓，并在宿主机写出 `.npmrc`
6. 等待 postgres、nexus、catalog、skills-web Ready

如果基础镜像需要替换来源：

```bash
make dev \
  DEV_POSTGRES_IMAGE=registry.example.com/library/postgres:17 \
  DEV_NEXUS_IMAGE=registry.example.com/sonatype/nexus3:latest
```

严格鉴权模式：

```bash
make dev STRICT=1
```

如果镜像已构建，只想把最新内容部署到已有 kind 环境：

```bash
make build
make deploy
```

## 使用独立 `skills` CLI

本地 E2E 现在直接走独立 CLI，不再通过 Makefile 包装 CLI 子命令。

先准备环境变量：

```bash
export SKILLS_REGISTRY_URL=http://127.0.0.1:18080
export SKILLS_REGISTRY_TOKEN=publisher-secret
export NPM_CONFIG_USERCONFIG=/private/tmp/skill-registry-nexus/.npmrc
export npm_config_registry=http://127.0.0.1:8081/repository/skills-npm/
```

只读查询可以把 `SKILLS_REGISTRY_TOKEN` 改成 `reader-secret`。

### 路径 A: publish / find / add

```bash
skills publish /abs/path/to/skill
skills find your-skill
skills add @company.skill/your-skill-name -y
```

预期：

- Nexus `skills-npm` 出现新 package/version
- Catalog 可查询到 release record
- 安装目录落到 `./.agents/skills/<skill-basename>`
- `skills-lock.json` 写入安装状态

### 路径 B: import-local

```bash
skills import-local /abs/path/to/legacy-skill \
  --package-name @company.skill/legacy-skill \
  --version 0.1.0 \
  --publish-registry http://127.0.0.1:8081/repository/skills-npm/
```

说明：

- `SKILL_PATH` 只在 CLI 本地使用
- CLI 会先归档源码，再把内容上传给 Catalog
- Catalog 不直接依赖调用方的绝对路径

如需从 git 源一把导入发布，使用：

```bash
skills import-git owner/repo --skill pdf
```

### 批量导入

```bash
uv run python zarf/tools/batch-import/batch_import_skills.py \
  --manifest /abs/path/to/import.json \
  --catalog-url http://127.0.0.1:18080 \
  --npm-registry http://127.0.0.1:8081/repository/skills-npm/
```

manifest 里的本地 `path` 会先在调用方本地转成源码归档，再走 `source.type=upload_archive`。

## 建议验收顺序

1. 打开 <http://127.0.0.1:8080>，确认 web 正常
2. 打开 <http://127.0.0.1:8081>，确认 Nexus 可访问且 `skills-npm` 已存在
3. `curl http://127.0.0.1:18080/healthz`，确认 Catalog 返回 200
4. 跑一次 `skills publish /abs/path/to/skill`
5. 跑一次 `skills find your-skill`
6. 跑一次 `skills add @company.skill/your-skill-name -y`
7. 跑一次 `skills import-local ... --publish-registry http://127.0.0.1:8081/repository/skills-npm/`
8. 如有批量场景，再跑批量导入脚本

## 重置本地环境

Makefile 不再提供 `dev-reset` / `dev-down`。需要完整重建时直接：

```bash
kind delete cluster --name skill-registry
make clean-dev
make dev
```
