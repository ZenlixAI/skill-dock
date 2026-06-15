# skill_registry

`skill_registry` 的本地运行路径现在只保留一条 production-oriented 流程：

- `kind`
- `catalog`
- `skills-web`
- `postgres`
- `nexus`

不再提供细粒度的 CLI / kind 生命周期 Make 包装。Makefile 只保留顶层工作流入口。

## 常用命令

```bash
make help
make dev
make deploy
make build
make verify
make test-go
make test-web
make build-web
make clean-dev
```

含义：

- `make dev`：构建本地镜像，确保 kind 集群存在，部署开发栈，初始化 Nexus，并等待服务 Ready
- `make deploy`：把最新本地镜像部署到现有 kind 环境
- `make build`：构建本地 `catalog` / `skills-web` 镜像
- `make verify`：执行顶层校验
- `make clean-dev`：删除本地 Nexus bootstrap 状态目录

## 本地开发

首次启动：

```bash
make dev
make verify
```

代码改完后重新部署本地环境：

```bash
make build
make deploy
```

严格鉴权模式：

```bash
make dev STRICT=1
```

默认本地入口：

- Web: <http://127.0.0.1:8080>
- Nexus UI / API: <http://127.0.0.1:8081>
- Catalog API: <http://127.0.0.1:18080>
- Nexus npm repository: <http://127.0.0.1:8081/repository/skills-npm/>
- Nexus `.npmrc`: `/private/tmp/skill-registry-nexus/.npmrc`

相关清单：

- [zarf/k8s/dev/kind-config.yaml](zarf/k8s/dev/kind-config.yaml)
- [zarf/k8s/dev/kind-stack.yaml](zarf/k8s/dev/kind-stack.yaml)

## 本地 Nexus E2E

仓库不再通过 Makefile 包装 CLI。需要发布、搜索、安装时，直接使用独立 `skills` CLI。

```bash
make dev

export SKILLS_REGISTRY_URL=http://127.0.0.1:18080
export SKILLS_REGISTRY_TOKEN=publisher-secret
export NPM_CONFIG_USERCONFIG=/private/tmp/skill-registry-nexus/.npmrc
export npm_config_registry=http://127.0.0.1:8081/repository/skills-npm/

skills publish /abs/path/to/skill
skills find your-skill
skills add @company.skill/your-skill -y
skills import-local /abs/path/to/legacy-skill \
  --package-name @company.skill/legacy-skill \
  --version 0.1.0 \
  --npm-registry http://127.0.0.1:8081/repository/skills-npm/
```

说明：

- `publish` / `import-local` 需要可发布凭据，通常使用 `publisher-secret`
- 只读查询可以把 `SKILLS_REGISTRY_TOKEN` 换成 `reader-secret`
- `NPM_CONFIG_USERCONFIG` 指向 `make dev` / `make deploy` 生成的本地 Nexus `.npmrc`
- `skills add` 安装结果默认写入当前项目的 `./.agents/skills/`

批量导入不再通过 Makefile 暴露，直接运行脚本：

```bash
uv run python zarf/tools/batch-import/batch_import_skills.py \
  --manifest /abs/path/to/import.json \
  --catalog-url http://127.0.0.1:18080 \
  --npm-registry http://127.0.0.1:8081/repository/skills-npm/
```

详细步骤见 [docs/local-nexus-e2e.zh-CN.md](docs/local-nexus-e2e.zh-CN.md) 和 [docs/batch-import-skills.zh-CN.md](docs/batch-import-skills.zh-CN.md)。

## Web UI

常用命令：

```bash
npm --prefix apps/skills-web test
npm --prefix apps/skills-web run build
npm --prefix apps/skills-web run dev
```

直连本地 Catalog 调试：

```bash
make dev

cd apps/skills-web
VITE_SKILL_REGISTRY_URL=http://127.0.0.1:18080 \
VITE_SKILL_REGISTRY_TOKEN=publisher-secret \
npm run dev
```

Web 真实模式现在默认走生产路径：

- 浏览器优先走 session / 真实 API
- 导入页负责 `prepare -> validate -> pack`
- `npm publish` 仍然在 Web 之外完成

Playwright 冒烟见 [docs/local-web-catalog-e2e.zh-CN.md](docs/local-web-catalog-e2e.zh-CN.md)。

## 客户交付

客户现场部署统一看：

- [docs/customer-k3s-deployment.zh-CN.md](docs/customer-k3s-deployment.zh-CN.md)
- [../skill-registry-helm-chart/README.md](/Users/ultraman/workspace/skill-registry-helm-chart/README.md)
- [../skill-registry-helm-chart/values.yaml](/Users/ultraman/workspace/skill-registry-helm-chart/values.yaml)
- [../skill-registry-helm-chart/values.192.168.10.124.yaml](/Users/ultraman/workspace/skill-registry-helm-chart/values.192.168.10.124.yaml)

## 清理与重建

只清理本地 Nexus bootstrap 状态：

```bash
make clean-dev
```

如果要完整重建本地 kind 环境：

```bash
kind delete cluster --name skill-registry
make clean-dev
make dev
```

## 关键配置

CLI：

- `SKILLS_REGISTRY_URL`
- `SKILLS_REGISTRY_TOKEN`
- `NPM_CONFIG_USERCONFIG`
- `npm_config_registry`

Catalog：

- `SKILL_REGISTRY_ADDR`
- `SKILL_REGISTRY_ARTIFACTS_PATH`
- `SKILL_REGISTRY_DATABASE_URL`
- `SKILL_REGISTRY_READER_TOKEN`
- `SKILL_REGISTRY_AUTHORING_TOKEN`
- `SKILL_REGISTRY_PUBLISHER_TOKEN`
- `SKILL_REGISTRY_NPM_REGISTRY_URL`
- `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL`
- `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION`
- `SKILL_REGISTRY_NPM_PUBLISH_TOKEN`

补充：

- `SKILL_REGISTRY_ARTIFACTS_PATH` 用于显式指定离线 artifact 元数据文件
- `SKILL_REGISTRY_NPM_REGISTRY_URL` 给 Catalog 后端自己读取 npm 元数据
- `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` 给 CLI / 浏览器下载 tarball
- 运行时已收敛为单一生产路径，不再提供额外启动分支

### `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION`

这个值给 Catalog 后端自己执行 server-side `npm publish` 使用，不是给浏览器，也不是给你本机 CLI。

典型来源是 Nexus `.npmrc`：

1. token 形式

```ini
//127.0.0.1:8081/repository/skills-npm/:_authToken=NpmToken.xxxxx
```

对应：

```text
Bearer NpmToken.xxxxx
```

2. basic 形式

```ini
//127.0.0.1:8081/repository/skills-npm/:_auth=base64xxxx
```

对应：

```text
Basic base64xxxx
```

### 本地 kind 与 Nexus bootstrap

`make dev` 和 `make deploy` 会在部署流程里自动：

1. 初始化集群内 Nexus 仓
2. 重写 `/private/tmp/skill-registry-nexus/.npmrc`
3. 从 `.npmrc` 提取可用的 Authorization 头
4. patch `catalog-secrets.skill_registry_npm_publish_authorization`
5. 重启并等待 `deployment/catalog`

如果需要手工 patch：

```bash
AUTH_HEADER='Bearer NpmToken.xxxxx'
kubectl -n skill-registry patch secret catalog-secrets --type merge -p "{\"stringData\":{\"skill_registry_npm_publish_authorization\":\"$AUTH_HEADER\"}}"
kubectl -n skill-registry rollout restart deployment/catalog
kubectl -n skill-registry rollout status deployment/catalog
```
