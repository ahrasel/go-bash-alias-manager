SHELL := /bin/bash

.PHONY: help build tag release upload-install cut-release

VERSION ?= 

help:
	@echo "Usage:"
	@echo "  make build                # build release artifacts (scripts/build_release.sh)"
	@echo "  make tag VERSION=v1.2.3   # create git tag and push" 
	@echo "  make release VERSION=v1.2.3 # build, tag and create GitHub release (requires gh)"
	@echo "  make upload-install VERSION=v1.2.3 # upload install.sh to existing release"
	@echo "  make cut-release VERSION=v1.2.3  # convenience: build, tag, create release, upload install.sh"
	@echo "  Use FORCE=true to replace an existing tag: make tag VERSION=v1.2.3 FORCE=true"

build:
	@echo "Building artifacts..."
	bash scripts/build_release.sh

tag:
	@if [ -z "$(VERSION)" ]; then echo "ERROR: VERSION is required. Example: make tag VERSION=v1.2.3"; exit 1; fi
	@if git rev-parse -q --verify refs/tags/$(VERSION) >/dev/null; then \
		if [ "$(FORCE)" = "true" ]; then \
			echo "Tag $(VERSION) exists; deleting and recreating (force)"; \
			git tag -d $(VERSION) || true; git push --delete origin $(VERSION) || true; \
		else \
			echo "ERROR: tag '$(VERSION)' already exists. To replace it, run 'make tag VERSION=$(VERSION) FORCE=true'"; exit 1; \
		fi \
	fi
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@git push origin $(VERSION)

release:
	@if [ -z "$(VERSION)" ]; then echo "ERROR: VERSION is required. Example: make release VERSION=v1.2.3"; exit 1; fi
	@if [ ! -f "dist/bash-alias-manager_$(VERSION)_linux_amd64.tar.gz" ]; then echo "ERROR: artifact dist/bash-alias-manager_$(VERSION)_linux_amd64.tar.gz not found. Run 'make build' or use 'make cut-release'"; exit 1; fi
	@echo "Creating GitHub release $(VERSION)..."
	@gh release create $(VERSION) dist/bash-alias-manager_$(VERSION)_linux_amd64.tar.gz dist/bash-alias-manager_$(VERSION)_SHA256SUMS -t "$(VERSION)" -n "Release $(VERSION)" || echo "gh release create failed or release already exists"

upload-install:
	@if [ -z "$(VERSION)" ]; then echo "ERROR: VERSION is required. Example: make upload-install VERSION=v1.2.3"; exit 1; fi
	@echo "Uploading install.sh to release $(VERSION)..."
	@gh release upload $(VERSION) install.sh || echo "upload failed"

cut-release:
	@if [ -z "$(VERSION)" ]; then echo "ERROR: VERSION is required. Example: make cut-release VERSION=v1.2.3"; exit 1; fi
	@# Check for existing tag before doing heavy work
	@if git rev-parse -q --verify refs/tags/$(VERSION) >/dev/null; then \
		if [ "$(FORCE)" = "true" ]; then \
			echo "Tag $(VERSION) exists; deleting and recreating (force)"; \
			git tag -d $(VERSION) || true; git push --delete origin $(VERSION) || true; \
		else \
			echo "ERROR: tag '$(VERSION)' already exists. To replace it, run 'make cut-release VERSION=$(VERSION) FORCE=true'"; exit 1; \
		fi \
	fi
	@$(MAKE) tag VERSION=$(VERSION) FORCE=$(FORCE)
	@$(MAKE) build VERSION=$(VERSION)
	@$(MAKE) release VERSION=$(VERSION)
	@$(MAKE) upload-install VERSION=$(VERSION)
