.PHONY: build test fmt vet lint check release-pr release-build

BUMP ?= patch
TAG  ?= $(shell cat cmd/sava/version.txt)

build:
	go build -o sava ./cmd/sava

test:
	go test ./...

fmt:
	test -z "$$(gofmt -l .)"

vet:
	go vet ./...

lint:
	golangci-lint run

check: fmt vet test lint

# --- release (scripts run via bash: no exec permission needed) ---

release-pr:
	bash scripts/create-release-pr.sh $(BUMP)

release-build:
	bash scripts/build-release.sh $(TAG)
