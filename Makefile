.PHONY: build run test tools
.DEFAULT_GOAL := build

TAGS=containers_image_openpgp,exclude_graphdriver_btrfs,exclude_graphdriver_devicemapper

# Define the local bin directory (absolute path)
LOCALBIN := $(CURDIR)/.bin
ERRTRACE := $(LOCALBIN)/errtrace

tools:
	@mkdir -p "$(LOCALBIN)"
	@if [ ! -x "$(ERRTRACE)" ]; then \
		echo "Installing errtrace to $(LOCALBIN)..."; \
		GOBIN="$(LOCALBIN)" go install braces.dev/errtrace/cmd/errtrace; \
	fi

build: tools
	go build -toolexec="$(ERRTRACE)" -tags "$(TAGS)" .

run: tools
	go run -toolexec="$(ERRTRACE)" -tags "$(TAGS)" .

test: tools
	go test -toolexec="$(ERRTRACE)" -v -tags "$(TAGS)" ./pkg/cache
