APP_NAME ?= svelte-go-starter
OUT_DIR ?= tmp

dev:
	./dev.sh

frontend-install:
	cd frontend && npm ci

frontend-build:
	cd frontend && npm run build

build: frontend-install frontend-build
	mkdir -p $(OUT_DIR)
	go build -a -o ./$(OUT_DIR)/$(APP_NAME) .

