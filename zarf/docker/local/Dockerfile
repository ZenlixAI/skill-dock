FROM --platform=$BUILDPLATFORM golang:1.26-trixie AS builder

ARG TARGETOS
ARG TARGETARCH
ARG GOPROXY=https://goproxy.cn,direct

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY app ./app
COPY business ./business
COPY cmd ./cmd
COPY foundation ./foundation

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /usr/local/bin/skill-registry ./cmd/skill-registry

FROM debian:trixie-slim AS runtime

ARG APT_MIRROR=http://mirrors.aliyun.com/debian
ARG APT_SECURITY_MIRROR=http://mirrors.aliyun.com/debian-security

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    if [ -f /etc/apt/sources.list.d/debian.sources ]; then \
      sed -i \
        -e "s|http://deb.debian.org/debian|${APT_MIRROR}|g" \
        -e "s|https://deb.debian.org/debian|${APT_MIRROR}|g" \
        -e "s|http://deb.debian.org/debian-security|${APT_SECURITY_MIRROR}|g" \
        -e "s|https://deb.debian.org/debian-security|${APT_SECURITY_MIRROR}|g" \
        -e "s|http://security.debian.org/debian-security|${APT_SECURITY_MIRROR}|g" \
        -e "s|https://security.debian.org/debian-security|${APT_SECURITY_MIRROR}|g" \
        /etc/apt/sources.list.d/debian.sources; \
    fi \
    && if [ -f /etc/apt/sources.list ]; then \
      sed -i \
        -e "s|http://deb.debian.org/debian|${APT_MIRROR}|g" \
        -e "s|https://deb.debian.org/debian|${APT_MIRROR}|g" \
        -e "s|http://deb.debian.org/debian-security|${APT_SECURITY_MIRROR}|g" \
        -e "s|https://deb.debian.org/debian-security|${APT_SECURITY_MIRROR}|g" \
        -e "s|http://security.debian.org/debian-security|${APT_SECURITY_MIRROR}|g" \
        -e "s|https://security.debian.org/debian-security|${APT_SECURITY_MIRROR}|g" \
        /etc/apt/sources.list; \
    fi \
    && rm -f /etc/apt/apt.conf.d/docker-clean \
    && apt-get update -o Acquire::Retries=3 \
    && apt-get install -y --no-install-recommends ca-certificates git tzdata

WORKDIR /app

COPY --from=builder /usr/local/bin/skill-registry /usr/local/bin/skill-registry
COPY zarf/data ./data

RUN mkdir -p /var/lib/skill-registry \
    && useradd --system --create-home --uid 10001 skill-registry \
    && chown -R skill-registry:skill-registry /app /var/lib/skill-registry

USER skill-registry

ENV SKILL_REGISTRY_ADDR=:8080
ENV SKILL_REGISTRY_DATABASE_URL=file:/var/lib/skill-registry/skill-registry.db

EXPOSE 8080
VOLUME ["/var/lib/skill-registry"]

ENTRYPOINT ["/usr/local/bin/skill-registry"]
