# --- UI stage: build the Next.js static export served by the Go binary --------
# glibc base (not alpine) so Next/Turbopack native binaries load without musl
# compatibility shims.
FROM node:22-bookworm-slim AS ui

WORKDIR /ui
# Use the China npm mirror for both corepack (fetching pnpm) and dependency
# installs, so the build does not depend on direct registry.npmjs.org access.
ENV NEXT_TELEMETRY_DISABLED=1 \
    COREPACK_NPM_REGISTRY=https://registry.npmmirror.com \
    npm_config_registry=https://registry.npmmirror.com
RUN corepack enable && corepack prepare pnpm@10.17.0 --activate

# Install deps first against just the manifest + lockfile for layer caching.
COPY ui-next/package.json ui-next/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Build the static export. EMBED_TARGET=skip leaves the output in ./out instead
# of copying into the Go tree (the Go stage copies it across explicitly).
COPY ui-next/ ./
RUN EMBED_TARGET=skip pnpm build:embed

# --- Go stage: compile the server, embedding the freshly built UI -------------
FROM golang:1.26-alpine AS build

WORKDIR /src
ENV CGO_ENABLED=0 GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Replace any committed static export with the export built in the ui stage, so
# the //go:embed all:web directive bundles the current UI.
RUN rm -rf internal/http/static/web
COPY --from=ui /ui/out internal/http/static/web

RUN go build -trimpath -ldflags="-s -w" -o /out/mem9 ./cmd/mem9

# --- Runtime stage ------------------------------------------------------------
# Docker Hub base (reachable via the configured registry accelerator) instead of
# gcr.io distroless. The binary is static, but needs CA certificates for its
# outbound HTTPS calls (LLM/embeddings); those are copied from the golang stage,
# which already ships the ca-certificates bundle, so no apk network is required.
FROM alpine:3.21
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/mem9 /mem9

USER app
ENTRYPOINT ["/mem9", "serve"]
