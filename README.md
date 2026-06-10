# skill_registry

## 启动模式

这套仓库现在本地只保留 `kind` 开发与真实 E2E 这一条运行路径：

- `make dev-run`
- `make dev-status`
- `make dev-bootstrap`
- `skills-web` / `catalog` / `postgres` / `nexus` 都在 `kind` 里
- 本地镜像默认使用 `zarf/docker/local/` 下的 Dockerfile 构建

如果你现在关心的是**客户现场交付 / k3s 部署 / CLI 下发方式**，直接看：

- [docs/customer-k3s-deployment.zh-CN.md](docs/customer-k3s-deployment.zh-CN.md)

## 本地开发

### 1. 快速验证

```bash
make verify
```

如果需要额外跑 CLI dist smoke test，再执行：

```bash
make verify-full
```

### 2. 本地 Kind 开发模式

推荐路径：

```bash
make dev-run
make dev-status
```

说明：

- `make dev-run` 会：
  - 构建 `catalog` / `skills-web` 本地镜像
  - 创建 kind 集群
  - 把镜像 load 到集群
  - apply 本地开发清单
  - bootstrap 集群内 Nexus 仓并写出本地 `.npmrc`
- 默认部署：
  - `skills-web`
  - `catalog`
  - `postgres`
  - `nexus`
- 默认访问入口：
  - Web：<http://127.0.0.1:8080>
  - Nexus：<http://127.0.0.1:8081>
  - Catalog：<http://127.0.0.1:18080>
- 默认浏览器登录凭据：
  - `reader / reader-secret`
  - `operator / op-secret`
- `kind` 本地 Web 现在默认走浏览器 session 登录，不再注入固定 `reader-secret` 到 web 代理层
- 如果只想重跑 Nexus 初始化：

```bash
make dev-bootstrap
```

相关清单：

- [zarf/k8s/dev/kind-config.yaml](zarf/k8s/dev/kind-config.yaml)
- [zarf/k8s/dev/kind-stack.yaml](zarf/k8s/dev/kind-stack.yaml)

### 3. 本地真实 Nexus E2E

本地发布 / 搜索 / 安装统一走 `kind`：

```bash
make dev-run
make cli ARGS="publish /abs/path/to/skill" NPM_BACKEND=nexus
make cli ARGS="find your-skill"
make cli ARGS="add @company.skill/your-skill -y" NPM_BACKEND=nexus
```

注意区分两类能力：

- `make cli-publish SKILL_PATH=...` 现在走的是 **CLI 本地 pack -> npm publish -> Catalog release-record**
- 所以它已经**不要求 Catalog 直接读你的 host 本地目录**
- `make cli-import-local SKILL_PATH=... PACKAGE=... VERSION=...` 现在走的是 **CLI 本地归档源码 -> Catalog prepare/validate/pack -> npm publish -> Catalog release-record**
- 正式跨环境发布应收敛到两条命令：`publish` 或 `import-local`

严格鉴权：

```bash
make dev-run STRICT=1
make cli ARGS="publish /abs/path/to/skill" NPM_BACKEND=nexus
make cli ARGS="add @company.skill/your-skill -y" NPM_BACKEND=nexus
```

详细步骤看 [docs/local-nexus-e2e.zh-CN.md](docs/local-nexus-e2e.zh-CN.md)。

### 4. CLI 使用

```bash
make cli ARGS="find pdf"
make cli ARGS="find pdf --json"
make cli ARGS="add @company.skill/pdf-skill -y" NPM_BACKEND=nexus
make cli ARGS="publish /abs/path/to/skill" NPM_BACKEND=nexus
make cli ARGS="import-local /abs/path/to/skill --package-name @company.skill/legacy-skill --version 0.1.0" NPM_BACKEND=nexus
make cli-list CLI_ARGS="--json"
make cli-update CLI_ARGS="@company.skill/pdf-skill -g"
```

说明：

- `make cli ARGS="..."` 是通用入口，本质上还是 `node apps/skills-cli/src/cli.ts ...`，但会自动按命令类型注入默认 token
- 默认 token 规则：
  - `find` / `add` / `list` / `remove` / `update` 走 `reader-secret`
  - `init` / `inspect` 走 `authoring-secret`
  - `publish` / `import-local` 走 `publisher-secret`
- 对支持 Catalog 的命令，`make cli` 会自动补 `--registry http://127.0.0.1:18080`
- 对 `import-local + NPM_BACKEND=nexus`，`make cli` 会自动补 `--npm-registry http://127.0.0.1:8081/repository/skills-npm/`
- `make cli-find` / `cli-add` / `cli-publish` / `cli-import-local` / `cli-list` / `cli-remove` / `cli-update` / `cli-init` / `cli-inspect` 都保留着，但现在只是对 `make cli` 的薄封装
- 需要切换 npm 仓场景时，通过 `NPM_BACKEND=nexus`
- 安装内容会落到当前工作目录下的 `./.agents/skills/`
- registry 包本地目录会优先使用短目录名；发生 basename 冲突时会稳定退化到带 scope 信息的目录名
- 项目级安装状态会写到当前工作目录下的 `skills-lock.json`
- `.dist/` 只是内部打包产物目录，不会进入最终安装结果
- 用户级安装请直接使用 CLI 并传 `-g`

### 4.1 真实用户怎么用

区分两类人：

- 仓库开发者 / 本地联调：用 `make cli`，因为它会自动注入本地 `kind + Nexus + Catalog` 的地址、token 和 `.npmrc`
- 客户 / CI / 非仓库环境：直接用 `skills` CLI，本质上不应该依赖这个仓库里的 Makefile

真实用户通常这样用：

```bash
export SKILLS_REGISTRY_URL=https://skills.example.com
export SKILLS_REGISTRY_TOKEN=reader-or-publisher-token

skills find pdf
skills add @company.skill/pdf-skill -y
skills publish /abs/path/to/skill
skills import-local /abs/path/to/skill --package-name @company.skill/legacy-skill --version 0.1.0 --npm-registry https://skills.example.com/repository/skills-npm/
```

也就是说：

- Makefile 主要是给这个 repo 的维护者和本地 E2E 用
- 真正的产品接口还是 `skills` CLI + `SKILLS_REGISTRY_URL` / `SKILLS_REGISTRY_TOKEN`

### 4.2 用 CLI 走完整流程

本地 `kind + Nexus + Catalog` 联调时，建议按这个顺序：

1. 启动本地系统

```bash
make dev-run
```

2. 发布一个标准 skill 包

```bash
make cli ARGS="publish /abs/path/to/skill" NPM_BACKEND=nexus
```

适用场景：

- 你的 skill 目录已经是可发布形态
- CLI 本地会做校验、打包和 `npm publish`
- Catalog 只负责 release-record 等服务端记录

3. 搜索已发布 skill

```bash
make cli ARGS="find your-skill"
make cli ARGS="find your-skill --json"
```

4. 安装已发布 skill

```bash
make cli ARGS="add @company.skill/your-skill -y" NPM_BACKEND=nexus
```

结果：

- skill 会安装到当前项目的 `./.agents/skills/`
- 项目安装状态会写进 `skills-lock.json`

5. 导入一个还不规范的本地旧 skill

```bash
make cli ARGS="import-local /abs/path/to/legacy-skill --package-name @company.skill/legacy-skill --version 0.1.0" NPM_BACKEND=nexus
```

适用场景：

- 原目录还不是一个现成可直接 publish 的包
- CLI 会先归档源码，再交给 Catalog prepare / validate / pack
- 最后再 `npm publish` 到 Nexus，并写 release-record

6. 查看当前安装结果

```bash
make cli-list
make cli-list CLI_ARGS="--json"
```

7. 更新已安装 skill

```bash
make cli-update
make cli-update CLI_ARGS="@company.skill/your-skill -g"
```

8. 删除已安装 skill

```bash
make cli-remove CLI_ARGS="@company.skill/your-skill -y"
```

9. 本地初始化 / 检查 skill 目录

```bash
make cli-init NAME=my-skill
make cli-inspect SKILL_PATH=/abs/path/to/skill
```

### 5. Web UI

安装与启动：

```bash
make web-install
make web-dev
```

直连本地 `kind` Catalog：

```bash
make dev-run

cd apps/skills-web
VITE_SKILL_REGISTRY_URL=http://127.0.0.1:18080 \
VITE_SKILL_REGISTRY_TOKEN=publisher-secret \
npm run dev
```

当前 Web 的真实边界：

- 浏览器默认走 session 登录，不再依赖固定 `reader-secret` 注入
- Web 导入/发布页现在是 **pack-only handoff**
- 也就是：
  - Web 负责 `prepare -> validate -> pack`
  - 页面会给出 bundle / release handoff 信息
  - 真正的 `npm publish` 仍在 Web 外完成
  - `release-record` 仍然发生在包已经进入 npm registry 之后

构建验证：

```bash
make typecheck-web
make build-web
make test-web
```

Web E2E 见 [docs/local-web-catalog-e2e.zh-CN.md](docs/local-web-catalog-e2e.zh-CN.md)。

### 6. 批量导入

dry-run：

```bash
uv run python zarf/tools/batch-import/batch_import_skills.py \
  --manifest zarf/tools/batch-import/example.json \
  --dry-run
```

执行：

```bash
make batch-import MANIFEST=/abs/path/to/import.json NPM_BACKEND=nexus
```

详细说明看 [docs/batch-import-skills.zh-CN.md](./docs/batch-import-skills.zh-CN.md)。

### 7. k3s 交付

部署说明统一看 [docs/customer-k3s-deployment.zh-CN.md](docs/customer-k3s-deployment.zh-CN.md)。

如果目标是客户现场 Helm 交付，直接看：

- [docs/customer-k3s-deployment.zh-CN.md](docs/customer-k3s-deployment.zh-CN.md)
- [zarf/k8s/customer-k3s-example.yaml](zarf/k8s/customer-k3s-example.yaml)

### 8. 重置与清理

```bash
# 重置 kind 集群 + 本地临时状态，并重新拉起整套 dev stack
make dev-reset

# 只清掉本地临时状态，不重新启动
make clean-dev
```

## 关键配置

CLI：

- `SKILLS_REGISTRY_URL`
- `SKILLS_REGISTRY_TOKEN`
- `SKILLS_REGISTRY_ROLE`
- `DEV_POSTGRES_IMAGE`
- `DEV_NEXUS_IMAGE`

Catalog：

- `SKILL_REGISTRY_ADDR`
- `SKILL_REGISTRY_DATABASE_URL`
- `SKILL_REGISTRY_READER_TOKEN`
- `SKILL_REGISTRY_AUTHORING_TOKEN`
- `SKILL_REGISTRY_PUBLISHER_TOKEN`
- `SKILL_REGISTRY_NPM_REGISTRY_URL`
- `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION` or `SKILL_REGISTRY_NPM_PUBLISH_TOKEN`
- `SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL`

补充：

- 本地开发默认就是 `kind`：Catalog 暴露在 `http://127.0.0.1:18080`，Nexus 暴露在 `http://127.0.0.1:8081`
- `make dev-bootstrap` 会重新初始化集群内 Nexus 仓，并重写本地 `/private/tmp/skill-registry-nexus/.npmrc`
- `SKILL_REGISTRY_NPM_REGISTRY_URL` 给 Catalog 自己读 registry 元数据；`SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` 给宿主机 CLI 下载 tarball
- 如果要让 Web 里的“发布 skill”由 Catalog 直接发到 Nexus，还需要给 Catalog 配 `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION`（例如 `Bearer NpmToken.xxxxx`）；也可改用 `SKILL_REGISTRY_NPM_PUBLISH_TOKEN`
- CLI 永远连 Catalog；切换后端 npm registry 时，需要同时确认 Catalog 的内部访问地址和对外下载地址
- Docker 镜像默认关闭 sample data 与 `/mock-npm`

### `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION` 怎么取

这个值不是给浏览器，也不是给你本机 CLI；它是给 **Catalog 后端自己** 在执行 server-side `npm publish` 时用的。

适用场景：

- Web 里的“发布 skill”
- 任何由 Catalog 后端代发 npm 包的流程

不适用场景：

- 你本机直接跑 `npx --registry ... skills publish ...`
  - 这个看的是你本机 `.npmrc` / `npm adduser`

常见来源就是 Nexus 登录后生成的 `.npmrc`。典型有两种：

1. token 形式

```ini
//127.0.0.1:8081/repository/skills-npm/:_authToken=NpmToken.xxxxx
```

Catalog 里要配成：

```text
Bearer NpmToken.xxxxx
```

2. basic 形式

```ini
//127.0.0.1:8081/repository/skills-npm/:_auth=base64xxxx
```

Catalog 里要配成：

```text
Basic base64xxxx
```

如果你想从 `.npmrc` 自动提取，可以用：

```bash
node -e 'const fs=require("fs");const p=process.argv[1];for(const raw of fs.readFileSync(p,"utf8").split(/\r?\n/)){const line=raw.trim();if(!line||line.startsWith("#")||line.startsWith(";")) continue;let m=line.match(/:_authToken=(.+)$/);if(m){console.log("Bearer "+m[1].trim());process.exit(0)}m=line.match(/:_auth=(.+)$/);if(m){console.log("Basic "+m[1].trim());process.exit(0)}}process.exit(1)' /path/to/.npmrc
```

### 本地 kind 怎么配

本地 `kind` 默认会把 Nexus 的独立 npm 登录态写到：

```text
/private/tmp/skill-registry-nexus/.npmrc
```

现在 `make dev-bootstrap` 会自动：

1. 初始化集群内 Nexus
2. 重写这个 `.npmrc`
3. 从中提取可用的 Authorization 头
4. patch 到 `catalog-secrets.skill_registry_npm_publish_authorization`
5. 重启 `deployment/catalog`

所以本地推荐直接执行：

```bash
make dev-apply
make dev-bootstrap
```

如果你要手工 patch：

```bash
AUTH_HEADER='Bearer NpmToken.xxxxx'
kubectl -n skill-registry patch secret catalog-secrets --type merge -p "{\"stringData\":{\"skill_registry_npm_publish_authorization\":\"$AUTH_HEADER\"}}"
kubectl -n skill-registry rollout restart deployment/catalog
kubectl -n skill-registry rollout status deployment/catalog
```
