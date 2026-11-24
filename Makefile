.PHONY: sdk server build test clean

sdk:
	cd sdk && npm ci && npm run build
	rm -rf internal/sdkassets/dist
	mkdir -p internal/sdkassets/dist
	cp -R sdk/dist/. internal/sdkassets/dist/
	touch internal/sdkassets/dist/.gitkeep

server:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/annotation-service ./cmd/annotation-service

build: sdk server

test:
	go vet ./...
	go test ./...
	cd sdk && npm run typecheck && npm test

clean:
	rm -rf bin internal/sdkassets/dist sdk/dist
