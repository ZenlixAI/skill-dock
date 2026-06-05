FROM --platform=$BUILDPLATFORM truth-ai-registry.cn-hangzhou.cr.aliyuncs.com/base/node:22 AS build

WORKDIR /src/apps/skills-web

COPY apps/skills-web/package.json apps/skills-web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci

COPY apps/skills-web/ ./
RUN npm run build

FROM truth-ai-registry.cn-hangzhou.cr.aliyuncs.com/base/nginx:1.27 AS runtime

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata

ENV CATALOG_UPSTREAM=http://catalog:8080
ENV CATALOG_AUTHORIZATION_HEADER=

COPY zarf/docker/nginx/skills-web.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /src/apps/skills-web/dist/ /usr/share/nginx/html/

EXPOSE 8080
