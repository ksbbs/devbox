# Stage 1: Build frontend (amd64 only, avoid slow QEMU emulation for npm)
FROM --platform=linux/amd64 node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Stage 2: Build Go binary per-platform (pure Go cross-compile is fast)
FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o devbox ./cmd/devbox/

# Stage 3: Final image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates git
COPY --from=backend /app/devbox /usr/local/bin/devbox
COPY --from=frontend /app/web/dist /usr/share/devbox/frontend
COPY configs/devbox.yaml /etc/devbox/default.yaml
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["devbox"]
CMD ["-c", "/etc/devbox/default.yaml", "-f", "/usr/share/devbox/frontend"