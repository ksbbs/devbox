# Stage 1: Build frontend (architecture-independent static assets)
# $BUILDPLATFORM ensures this stage only ever runs on the native builder,
# since the same dist/ works for both amd64 and arm64 images.
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend
RUN corepack enable && corepack prepare pnpm@10 --activate
WORKDIR /app/web
COPY web/pnpm-lock.yaml web/package.json ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile
COPY web/ .
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm run build

# Stage 2: Cross-compile Go binary.
# CGO_ENABLED=0 lets us compile for the target arch on the native builder,
# avoiding slow QEMU emulation entirely.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS backend
WORKDIR /app
ARG TARGETOS TARGETARCH
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o devbox ./cmd/devbox/

# Stage 3: Minimal runtime image (per-target-architecture base)
FROM --platform=$TARGETPLATFORM alpine:3.21
RUN apk add --no-cache ca-certificates git curl
COPY --from=backend /app/devbox /usr/local/bin/devbox
COPY --from=frontend /app/web/dist /usr/share/devbox/frontend
COPY configs/devbox.yaml /etc/devbox/default.yaml
VOLUME /data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -sf http://localhost:8080/health || exit 1
ENTRYPOINT ["devbox"]
CMD ["-c", "/etc/devbox/default.yaml", "-f", "/usr/share/devbox/frontend"]
