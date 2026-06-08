FROM --platform=$BUILDPLATFORM node:24-bookworm AS build

ARG NPM_REGISTRY=https://registry.npmmirror.com

WORKDIR /src/apps/skills-web

COPY apps/skills-web/package.json apps/skills-web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --registry=${NPM_REGISTRY}

COPY apps/skills-web/index.html ./index.html
COPY apps/skills-web/tsconfig.json ./tsconfig.json
COPY apps/skills-web/vite.config.ts ./vite.config.ts
COPY apps/skills-web/src ./src
RUN npm run build

FROM nginx:1.29-alpine AS runtime

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata

ENV CATALOG_UPSTREAM=http://catalog:8080
ENV CATALOG_AUTHORIZATION_HEADER=

COPY zarf/docker/nginx/skills-web.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /src/apps/skills-web/dist/ /usr/share/nginx/html/

EXPOSE 8080
