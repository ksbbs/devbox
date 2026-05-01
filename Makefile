.PHONY: all frontend backend docker dev clean

all: backend

frontend:
	cd web && npm ci && npm run build

backend:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o devbox ./cmd/devbox/

docker:
	docker build -t devbox:latest .

dev-frontend:
	cd web && npm run dev

dev-backend:
	CGO_ENABLED=0 go run ./cmd/devbox/ -c configs/devbox.yaml -f web/dist

clean:
	rm -f devbox devbox.exe
	cd web && rm -rf dist node_modules