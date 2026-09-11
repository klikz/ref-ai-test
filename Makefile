.PHONY: build buildWin run dev ui-dev install-dev

build:
	go build -v ./cmd/api_v3
	cd ui && npm run build

buildWin:
	GOOS=windows GOARCH=amd64 go build -v ./cmd/api_v3

run:
	go run -v ./cmd/api_v3

ui-dev:
	cd ui && npm run dev

install-dev:
	npm install

# Backend (api_v3) + Vite UI in one terminal (first time: make install-dev)
dev:
	npm run dev

.DEFAULT_GOAL := dev
