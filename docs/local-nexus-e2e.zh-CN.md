# 本地 Kind + Nexus E2E

当前仓库的本地真实 E2E 只保留一条路径：

- `skills-web`
- `catalog`
- `postgres`
- `nexus`

全部跑在 `kind` 里，宿主机只负责：

- 浏览器访问
- 执行 `skills` CLI
- 执行 `npm publish`

## 固定本地入口

- Web：<http://127.0.0.1:8080>
- Nexus UI / API：<http://127.0.0.1:8081>
- Catalog API：<http://127.0.0.1:18080>
- Nexus npm repository：<http://127.0.0.1:8081/repository/skills-npm>
- Nexus `.npmrc`：`/private/tmp/skill-registry-nexus/.npmrc`

## 启动方式

```bash
cd <repo-root>/skill_registry
make dev-run
```

`make dev-run` 会完成：

1. 构建 `catalog` / `skills-web` 本地镜像
2. 创建 `kind` 集群
3. 把镜像 load 进集群
4. apply `skills-web + catalog + postgres + nexus`
5. 初始化集群内 Nexus 仓，并在宿主机写出 `.npmrc`

如果你的环境拉 Docker Hub 不稳定，可以先改基础镜像来源：

```bash
make dev-run \
  DEV_POSTGRES_IMAGE=registry.example.com/library/postgres:17 \
  DEV_NEXUS_IMAGE=registry.example.com/sonatype/nexus3:latest
```

这两个镜像现在会在 `dev-run` 里先由宿主机 `docker pull`，再 `kind load docker-image` 进集群。

如果你只想重跑 Nexus 初始化：

```bash
make dev-bootstrap
```

严格鉴权模式：

```bash
make dev-run STRICT=1
```

## 路径 A：publish / find / add

```bash
make dev-run
make cli ARGS="publish /abs/path/to/skill" NPM_BACKEND=nexus
make cli ARGS="find your-skill"
make cli ARGS="add @company.skill/your-skill-name -y" NPM_BACKEND=nexus
```

预期：

- Nexus `skills-npm` 出现新 package/version
- Catalog 可查询到 release record
- 安装目录落到 `./.agents/skills/<skill-basename>`
- `skills-lock.json` 写入 `packageName`、`version`、`registry`、`integrity`、`skillManifestDigest`

## 路径 B：import-local

```bash
make dev-run
make cli \
  ARGS="import-local /abs/path/to/legacy-skill --package-name @company.skill/legacy-skill --version 0.1.0" \
  NPM_BACKEND=nexus
```

说明：

- `SKILL_PATH` 只在 CLI 本地使用
- CLI 会先归档源码，再把内容上传给 Catalog
- Catalog 不直接依赖调用方的绝对路径

## 批量导入

```bash
make dev-run
make batch-import MANIFEST=/abs/path/to/import.json NPM_BACKEND=nexus
```

manifest 里的本地 `path` 会先在调用方本地转成源码归档，再走 `source.type=upload_archive`。

兼容说明：

- `make cli-publish` / `make cli-find` / `make cli-add` / `make cli-import-local` 还可以继续用
- 但现在内部都只是对 `make cli ARGS="..."` 的薄封装
- `make cli` 会按命令自动补默认 token；支持 Catalog 的命令会自动补 `--registry`
- `import-local + NPM_BACKEND=nexus` 会自动补 `--npm-registry`
- 新增 CLI 子命令时，优先直接走 `make cli`

## 建议验收顺序

1. 打开 <http://127.0.0.1:8080>，确认 web 正常
2. 打开 <http://127.0.0.1:8081>，确认 Nexus 可访问且 `skills-npm` 已存在
3. `curl http://127.0.0.1:18080/healthz`，确认 Catalog 返回 200
4. 跑一次 `cli-publish`
5. 跑一次 `cli-find`
6. 跑一次 `cli-add`
7. 跑一次 `cli-import-local`
8. 如有批量场景，再跑 `batch-import`

## 重置本地环境

```bash
# 删除 kind 集群、本地 CLI/Nexus 临时状态，然后重新拉起
make dev-reset
```

如果你只想停掉环境并清理本地临时文件，不自动重启：

```bash
make dev-down
make clean-dev
```
