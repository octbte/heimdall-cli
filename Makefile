VERSION := $(shell cat VERSION | tr -d '[:space:]')
COMMIT  := $(shell git rev-parse --short HEAD)
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)

.PHONY: build release

build:
	go build -ldflags "$(LDFLAGS)" -o heimdall .

# Usage: make release v=1.0.1
release:
	@test -n "$(v)" || (echo "Usage: make release v=1.0.1" && exit 1)
	@echo "$(v)" > VERSION
	git add VERSION
	git commit -m "chore: release v$(v)"
	git tag v$(v)
	git push origin HEAD
	git push origin v$(v)
	@echo "Released v$(v) — GitHub Actions will build and publish."
