# 使用 Artifact Keeper 作为 npm 仓库

当前 Catalog 的 `SKILL_REGISTRY_NPM_REGISTRY_URL` 必须指向一个 npm-compatible repository；Artifact Keeper 的 npm 仓库可以直接替换本地 Nexus。下面的配置假设 Catalog Pod 能访问 `192.168.10.133:30883`，且给出的 token 具有该 npm 仓库的读写权限。

> 安全提示：token 已出现在聊天记录中。生产环境建议立即轮换，并把新 token 放入 Kubernetes Secret、CI Secret 或本机 `.npmrc`，不要提交到 Git。

## 1. 先验证 Artifact Keeper

```bash
export ARTIFACT_KEEPER_NPM_REGISTRY='http://192.168.10.133:30883/npm/npm-repo/'
export ARTIFACT_KEEPER_TOKEN='<替换为新 token>'

npm ping --registry "$ARTIFACT_KEEPER_NPM_REGISTRY"

cat > /tmp/skill-registry-artifact-keeper.npmrc <<EOF
registry=$ARTIFACT_KEEPER_NPM_REGISTRY
//192.168.10.133:30883/npm/npm-repo/:_authToken=$ARTIFACT_KEEPER_TOKEN
EOF

NPM_CONFIG_USERCONFIG=/tmp/skill-registry-artifact-keeper.npmrc \
  npm whoami --registry "$ARTIFACT_KEEPER_NPM_REGISTRY"
```

验证结果应为 `PONG`，并返回 Artifact Keeper 用户名。若 `whoami` 失败，先检查 token 类型：如果平台要求 Basic Auth，需要改成：

```ini
//192.168.10.133:30883/npm/npm-repo/:_auth=<base64(username:password)>
```

## 2. 配置 Catalog

确保 Catalog Deployment 设置以下运行时变量：

```text
SKILL_REGISTRY_NPM_REGISTRY_URL=http://192.168.10.133:30883/npm/npm-repo/
SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL=http://192.168.10.133:30883/npm/npm-repo/
SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION=Bearer <token>
```

本仓库的 Catalog 已支持给读取请求单独配置认证：

```text
SKILL_REGISTRY_NPM_READ_AUTHORIZATION=Bearer <token>
```

未配置 `SKILL_REGISTRY_NPM_READ_AUTHORIZATION` 时，会回退使用 `SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION`，因此最小改动只需设置 publish authorization。但生产部署建议分别使用只读 token 和发布 token。

修改 Secret 或 Deployment 后重启 Catalog：

```bash
kubectl -n skill-registry rollout restart deployment/<catalog-deployment-name>
kubectl -n skill-registry rollout status deployment/<catalog-deployment-name>
kubectl -n skill-registry logs deployment/<catalog-deployment-name>
```

### Kubernetes Deployment / Secret

如果通过 Kubernetes Deployment 部署，确保 Catalog 引用了以下 Secret keys：

```yaml
- name: SKILL_REGISTRY_NPM_REGISTRY_URL
  value: http://192.168.10.133:30883/npm/npm-repo/
- name: SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL
  value: http://192.168.10.133:30883/npm/npm-repo/
- name: SKILL_REGISTRY_NPM_READ_AUTHORIZATION
  valueFrom:
    secretKeyRef:
      name: artifact-keeper-npm
      key: authorization
- name: SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION
  valueFrom:
    secretKeyRef:
      name: artifact-keeper-npm
      key: authorization
```

如果 Artifact Keeper 只允许 HTTPS，请将所有 URL 改为 HTTPS，并保证 Catalog 镜像信任对应 CA 证书。

## 3. 配置 CLI / 本机 npm

创建本机 npm 配置：

```bash
cat > ~/.npmrc-artifact-keeper <<'EOF'
registry=http://192.168.10.133:30883/npm/npm-repo/
//192.168.10.133:30883/npm/npm-repo/:_authToken=<替换为新 token>
EOF
chmod 600 ~/.npmrc-artifact-keeper

export NPM_CONFIG_USERCONFIG="$HOME/.npmrc-artifact-keeper"
export npm_config_registry='http://192.168.10.133:30883/npm/npm-repo/'
```

配置 `skills` CLI 时，Catalog URL 仍然是 Catalog/Web 地址，不是 npm 地址：

```bash
export SKILLS_REGISTRY_URL='http://<catalog-or-web-address>'
export SKILLS_REGISTRY_TOKEN='<catalog publisher token>'
export NPM_CONFIG_USERCONFIG="$HOME/.npmrc-artifact-keeper"
export npm_config_registry='http://192.168.10.133:30883/npm/npm-repo/'

skills publish /abs/path/to/skill
skills find your-skill
skills add @company.skill/your-skill -y
```

如果使用显式参数：

```bash
skills import-local /abs/path/to/legacy-skill \
  --package-name @company.skill/legacy-skill \
  --version 0.1.0 \
  --registry "$SKILLS_REGISTRY_URL" \
  --publish-registry 'http://192.168.10.133:30883/npm/npm-repo/'
```

`--registry` 是 Catalog 地址；`--publish-registry` 才是 npm 仓库地址，二者不要混用。

## 4. 发布前做一次端到端检查

```bash
curl -fsS "$SKILLS_REGISTRY_URL/healthz"
curl -fsS "$SKILLS_REGISTRY_URL/readyz"

skills publish /abs/path/to/skill
```

检查：

1. Artifact Keeper 中出现 package 和 version。
2. Catalog 的 release record 创建成功。
3. `skills find` 能查到该 skill。
4. `skills add <package-name> -y` 能下载 tarball 并安装。

## 5. 常见问题

- **Artifact Keeper `npm ping` 成功，但 Catalog 发布失败**：检查 Catalog Pod 到 `192.168.10.133:30883` 的网络、token 是否包含 `Bearer ` 前缀，以及仓库是否允许 PUT 发布。
- **Catalog 能读 metadata，但下载 tarball 失败**：`SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL` 必须是 CLI 实际可访问的地址；不能填 Pod 内部 DNS。
- **返回 404**：确认 URL 是 Artifact Keeper 的 npm repository endpoint，而不是 Web UI 地址；路径应为 `/npm/npm-repo/`。
- **返回 401**：检查 token 是否已过期/被撤销、token 作用域是否覆盖 npm repository，并用 `npm whoami` 验证。
- **返回 409**：npm 版本不可覆盖。换一个新的 package version，不要重复发布相同版本。
- **仍然请求 Nexus**：检查 Catalog Deployment 的实际环境变量和 Secret，确认 `SKILL_REGISTRY_NPM_REGISTRY_URL` 已更新，并确认 Helm 模板没有继续生成内部 Nexus URL。
