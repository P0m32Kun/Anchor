.PHONY: build run dev test clean

GO_FILES := $(shell find . -name '*.go' -not -path './frontend/*')

build:
	go build -o bin/secbench .

run: build
	./bin/secbench

dev:
	go run .

test:
	go test ./...

clean:
	rm -rf bin/

tauri-dev:
	cd frontend && npm install && npm run tauri dev

tauri-build:
	cd frontend && npm install && npm run tauri build
