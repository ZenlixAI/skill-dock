# 本地 Verdaccio 真实 E2E

这套联调路径的目标不是替代生产仓库，而是把本地 `/mock-npm` 换成一个真实的 npm-compatible registry，验证下面这条完整链路：

1. `skills publish` 或批量导入脚本生成 bundle
2. `npm publish` 把 bundle 推到 Verdaccio
3. Catalog `release-record` 从 Verdaccio 读取 package metadata
4. `skills add` 根据 install target 下载 tarball 并安装

## 适用场景

- 想做比 `/mock-npm` 更真实的本地 E2E
- 想先低成本验证 npm metadata / tarball / release-record 契约
- 暂时不想引入 Nexus 那样更重的部署和权限模型

仓库内置的是一个“仅用于本机联调”的 Verdaccio 配置：

- 监听 `127.0.0.1:4873`
- `publish/access/unpublish` 对本机匿名开放
- 只适合本地开发，不适合生产

如果你要验证“发布需要登录”，看 [docs/local-verdaccio-auth-e2e.zh-CN.md](/Users/ultraman/workspace/skill_registry/docs/local-verdaccio-auth-e2e.zh-CN.md)。

## 启动步骤

### 1. 启动 Verdaccio

```bash
make verdaccio-run
```

默认配置文件：

```bash
zarf/docker/verdaccio/public.yaml
```

默认状态目录：

```bash
/private/tmp/skill-registry-verdaccio
```

### 2. 启动指向 Verdaccio 的 Catalog

另一个终端执行：

```bash
make registry-run-verdaccio
```

这会把下面两个配置一起指向 Verdaccio：

- `SKILL_REGISTRY_SAMPLE_NPM_BASE_URL`
- `SKILL_REGISTRY_NPM_REGISTRY_URL`

## 常用联调命令

发布本地 skill 到 Verdaccio：

```bash
make cli-publish-verdaccio SKILL_PATH=/abs/path/to/skill
```

批量导入到 Verdaccio：

```bash
make batch-import-verdaccio MANIFEST=/abs/path/to/import.json
```

搜索和安装仍然通过 Catalog：

```bash
make cli-find QUERY=pdf
make cli-add PACKAGE=@platform.skill/pdf-skill
```

## 推荐验证顺序

1. 启动 Verdaccio
2. 启动 `make registry-run-verdaccio`
3. 运行 `make cli-publish-verdaccio SKILL_PATH=...`
4. 运行 `make cli-find QUERY=...`
5. 运行 `make cli-add PACKAGE=...`

如果第 3 步成功但第 5 步失败，优先检查：

- `release-record` 是否成功
- Verdaccio metadata 中的 `dist.tarball` 是否可访问
- Catalog seed 或 release-record 指向的 registry URL 是否一致

## 和 Nexus 的差别

对这套系统，Verdaccio 和 Nexus 最关键的共同点是：

- 都会承载 npm metadata
- 都会返回 tarball URL
- 都会影响 `release-record` 的真实行为

但本地联调体验不同：

- Verdaccio 更轻，适合快速 E2E
- Nexus 更重，更像客户现场的企业仓库治理环境

因此建议：

- 日常本地真实 E2E 用 Verdaccio
- 准生产验收再补一轮 Nexus
