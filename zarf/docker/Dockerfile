FROM truth-ai-registry.cn-hangzhou.cr.aliyuncs.com/golang:1.26-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY app ./app
COPY business ./business
COPY cmd ./cmd
COPY zarf/data ./data
COPY foundation ./foundation

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/skill-registry ./cmd/skill-registry

FROM truth-ai-registry.cn-hangzhou.cr.aliyuncs.com/debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /out/skill-registry /usr/local/bin/skill-registry
COPY zarf/data ./data

RUN mkdir -p /var/lib/skill-registry \
    && useradd --system --create-home --uid 10001 skill-registry \
    && chown -R skill-registry:skill-registry /app /var/lib/skill-registry

USER skill-registry

ENV SKILL_REGISTRY_ADDR=:8080
ENV SKILL_REGISTRY_DATABASE_URL=file:/var/lib/skill-registry/skill-registry.db
ENV SKILL_REGISTRY_ENABLE_SAMPLE_DATA=false
ENV SKILL_REGISTRY_ENABLE_MOCK_REGISTRY=false

EXPOSE 8080
VOLUME ["/var/lib/skill-registry"]

ENTRYPOINT ["/usr/local/bin/skill-registry"]
