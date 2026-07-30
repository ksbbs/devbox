# Stage 1: Build frontend
FROM node:20-alpine AS frontend
RUN corepack enable && corepack prepare pnpm@10 --activate
WORKDIR /app/web
COPY web/pnpm-lock.yaml web/package.json ./
RUN pnpm install --frozen-lockfile
COPY web/ .
RUN pnpm run build

# Stage 2: Build Go binary (CGO_ENABLED=0 cross-compile to Linux)
FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o devbox ./cmd/devbox/

# Stage 3: Minimal runtime image
FROM alpine:3.21
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
