.PHONY: sdk server build test clean

sdk:
	cd sdk && npm ci && npm run build
	rm -rf internal/sdkassets/dist
	mkdir -p internal/sdkassets/dist
	cp -R sdk/dist/. internal/sdkassets/dist/
	touch internal/sdkassets/dist/.gitkeep

admin:
	cd admin && npm ci && npm run build
	rm -rf internal/adminassets/dist
	mkdir -p internal/adminassets/dist
	cp -R admin/dist/. internal/adminassets/dist/
	touch internal/adminassets/dist/.gitkeep

server:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/annotation-service ./cmd/annotation-service

build: sdk admin server

test:
	go vet ./...
	go test ./...
	cd sdk && npm run typecheck && npm test

clean:
	rm -rf bin internal/sdkassets/dist internal/adminassets/dist sdk/dist admin/dist
