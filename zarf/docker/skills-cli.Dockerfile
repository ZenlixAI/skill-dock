FROM --platform=$BUILDPLATFORM truth-ai-registry.cn-hangzhou.cr.aliyuncs.com/base/node:24-bookworm

ARG NPM_REGISTRY=https://registry.npmmirror.com

WORKDIR /src/apps/skills-cli

COPY apps/skills-cli/package.json apps/skills-cli/pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    npm install --global pnpm@10.17.1 --registry="${NPM_REGISTRY}" \
    && pnpm config set registry "${NPM_REGISTRY}" \
    && pnpm install --frozen-lockfile

COPY apps/skills-cli ./
RUN mkdir -p /seed \
    && pnpm run build \
    && npm pack --pack-destination /seed

WORKDIR /seed
