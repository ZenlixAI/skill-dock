# Skills CLI 的 npx 执行与 npm 认证指南

本指南面向手动使用 `skills` CLI 的场景。

核心原则：

- 直接使用 `npx skills ...`
- npm 认证使用官方 `npm login`，或者在 `.npmrc` 中直接配置
- 不需要设置 `NPM_CONFIG_USERCONFIG`
- Catalog 认证仍然是另一套凭据，通常通过 `SKILLS_REGISTRY_TOKEN` 提供

## 1. 直接通过 npx 运行 CLI

最常见的用法就是直接运行：

```bash
npx skills find pdf
npx skills add @company.skill/pdf-skill -y
npx skills publish ./my-skill
```

如果 `skills` CLI 包本身发布在私有 npm registry，需要给 `npx` 单独指定 registry：

```bash
npx --registry https://registry.example.com/ skills find pdf
```

注意：这个 `--registry` 只影响 `npx` 去哪里下载 `skills` CLI 包，不等于 `skills` 安装/发布时使用的 npm 认证配置。

## 2. 推荐方案：使用 npm login

这是最接近 npm 官方默认行为的方式，也是首选方式。

```bash
npm login --registry https://registry.example.com/
npm whoami --registry https://registry.example.com/
npm ping --registry https://registry.example.com/
```

如果你的 registry 只支持传统用户名/密码流程，可以显式指定：

```bash
npm login --registry https://registry.example.com/ --auth-type=legacy
```

登录成功后，npm 会把凭据写入默认用户配置位置，通常是 `~/.npmrc`。之后直接执行：

```bash
npx skills find pdf
npx skills add @company.skill/pdf-skill -y
```

即可复用同一套 npm 认证。

## 3. 直接配置 `.npmrc`

如果你不想交互式登录，也可以直接写 `.npmrc`。

### 3.1 用户级 `.npmrc`

适合本机长期使用：

```ini
registry=https://registry.example.com/
//registry.example.com/:_authToken=xxxxxxxxxxxxxxxx
always-auth=true
```

把这段内容保存到 `~/.npmrc` 即可。

### 3.2 项目级 `.npmrc`

适合单个项目或仓库：

```ini
@company:registry=https://registry.example.com/
//registry.example.com/:_authToken=xxxxxxxxxxxxxxxx
always-auth=true
```

把这段内容保存到当前项目根目录的 `./.npmrc`。

### 3.3 路径型 registry 的写法

如果 registry 带有路径前缀，`_authToken` 那一行必须匹配完整前缀：

```ini
registry=http://127.0.0.1:8081/repository/skills-npm/
//127.0.0.1:8081/repository/skills-npm/:_authToken=xxxxxxxxxxxxxxxx
always-auth=true
```

这一点对 Nexus、Artifact Keeper 这类 npm-compatible registry 尤其重要。

## 4. 推荐的日常流程

1. 先用 `npm login`，或者把 token 写进 `~/.npmrc` / `./.npmrc`
2. 用 `npm whoami` 或 `npm ping` 验证 registry 认证是否可用
3. 直接执行 `npx skills ...`

例如：

```bash
npm login --registry https://registry.example.com/

npm whoami --registry https://registry.example.com/

npx skills find pdf
npx skills add @company.skill/pdf-skill -y
```

## 5. 常见问题

### 5.1 `npx skills` 仍然无法下载 CLI 包

这通常说明 `npx` 访问的是另一个 npm registry，而不是你为技能 bundle 配好的那个 registry。

检查两件事：

- `npx --registry ...` 是否指向了 `skills` CLI 包所在的 npm registry
- 该 registry 是否需要额外的 npm 认证

### 5.2 `skills add` 或 `skills publish` 返回 401

通常是 `.npmrc` 没有命中，或者 registry URL 与 auth 前缀不一致。

优先检查：

- registry 是否写成了完全一致的 host + path
- `_authToken` 的前缀是否与 registry 完全匹配
- token 是否过期、撤销或权限不足

### 5.3 我已经登录了，但命令仍然报找不到凭据

检查当前 shell 的 `HOME` 是否被改写，或者当前目录下是否存在另一个 `.npmrc` 覆盖了用户级配置。

如果你使用的是项目级 `.npmrc`，确认你是在该项目目录里执行 `npx skills ...`。

### 5.4 这个指南为什么不使用 `NPM_CONFIG_USERCONFIG`

因为 `npm login` 和 `.npmrc` 本身就能完成标准 npm 认证流程。

`NPM_CONFIG_USERCONFIG` 更适合内部自动化或临时脚本，不适合作为日常手动操作的默认方案。
