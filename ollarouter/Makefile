.PHONY: build docker-build docker-run clean test test-short lint webui-build smoke

BINARY_NAME=ollarouter
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR=build

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build \
		-ldflags="-w -s -X github.com/aideveloper1976-arch/ollarouter/internal/router.FallbackVersion=$(VERSION)" \
		-o $(BUILD_DIR)/$(BINARY_NAME) \
		./cmd/$(BINARY_NAME)

docker-build:
	docker build -f Dockerfile -t $(BINARY_NAME):$(VERSION) ..
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest

docker-run:
	cd .. && docker compose up -d

clean:
	rm -rf $(BUILD_DIR)
	-docker rmi $(BINARY_NAME):latest 2>/dev/null

test:
	go test ./... -v -race -count=1

test-short:
	go test ./... -count=1

lint:
	gofmt -l . | (! grep .) && go vet ./...

# End-to-end check against a mock provider (needs Python 3 and curl).
smoke:
	../scripts/smoke-test.sh

# Rebuild the admin UI and refresh the copy embedded in the binary.
webui-build:
	cd ../webui && npm ci && npm run build
	rm -rf internal/webui/dist && mkdir -p internal/webui/dist
	cp -R ../webui/dist/. internal/webui/dist/

.PHONY: all
all: build test
