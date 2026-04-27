.PHONY: build run dev test clean

GO_FILES := $(shell find . -name '*.go' -not -path './frontend/*')

build:
	go build -o bin/anchor .

run: build
	./bin/anchor

dev:
	@lsof -ti:17421 | xargs kill -9 2>/dev/null || true
	go run .

test:
	go test ./...

clean:
	rm -rf bin/

tauri-dev:
	@pkill -f "vite" 2>/dev/null || true
	@pkill -f "target/debug/anchor" 2>/dev/null || true
	cd frontend && npm install
	./frontend/node_modules/.bin/tauri dev

tauri-build:
	cd frontend && npm install
	./frontend/node_modules/.bin/tauri build
