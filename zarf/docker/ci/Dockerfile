FROM --platform=$BUILDPLATFORM golang:1.26-trixie AS builder

ARG TARGETOS
ARG TARGETARCH
ARG GOPROXY=https://proxy.golang.org,direct

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    GOPROXY=$GOPROXY go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /usr/local/bin/skill-registry ./cmd/skill-registry

FROM debian:trixie-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates git tzdata \
    && rm -rf /var/lib/apt/lists/*

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
