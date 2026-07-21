GO ?= go
PYTHON ?= python3
COVERAGE_MIN ?= 80

GO_MODULES := services/yacynode libraries/yacymodel libraries/yacyproto libraries/yacycrawlcontract libraries/bytesize libraries/serviceruntime libraries/canonicalurl libraries/pagemarkdownstore services/yacycrawler services/yacytextindexer services/yacyvisitcrawl services/corpusmarkdown services/renderproxy
PY_MODULES := plugins/searxng/searxng-result-router plugins/searxng/searxng-crawled-text-search

COVER_PROFILE := coverage.out
COVER_EXCLUDE := /internal/vaulttest/|/test/e2e/|/internal/cdprender/

TOOLS_BIN := $(CURDIR)/.toolchain/bin
TOOLS_STAMP := $(TOOLS_BIN)/.installed
GOLANGCI_LINT := $(TOOLS_BIN)/golangci-lint
GO_ARCH_LINT := $(TOOLS_BIN)/go-arch-lint
RUFF := $(TOOLS_BIN)/ruff

PY_VENV_STAMPS := $(foreach m,$(PY_MODULES),$(m)/.venv/.installed)

define for_each_go
echo "==> $(1)"; \
for m in $(GO_MODULES); do \
	if ! out=$$(cd $$m && $(2) 2>&1); then \
		echo "==> $(1) $$m FAILED"; echo "$$out"; exit 1; \
	fi; \
done
endef

define for_each_py
echo "==> $(1)"; \
for m in $(PY_MODULES); do \
	if ! out=$$(cd $$m && $(2) 2>&1); then \
		echo "==> $(1) $$m FAILED"; echo "$$out"; exit 1; \
	fi; \
done
endef

.PHONY: tools \
	fmt fmt-go fmt-py \
	fmt-check fmt-check-go fmt-check-py \
	tidy tidy-check \
	lint lint-go lint-py \
	vet arch \
	test test-go test-py \
	cover cover-go cover-py \
	cover-check cover-check-go cover-check-py \
	build verify peer-hash \
	e2e e2e-images

fmt:         fmt-go fmt-py
fmt-check:   fmt-check-go fmt-check-py
lint:        lint-go lint-py
test:        test-go test-py
cover:       cover-go cover-py
cover-check: cover-check-go cover-check-py
build:       build-go
verify:      fmt-check tidy-check vet lint arch test cover-check build
	@echo "==> verify SUCCESS"

$(TOOLS_STAMP): tools/install tools/tools.lock
	./tools/install
	@touch $@

tools: $(TOOLS_STAMP)

$(PY_VENV_STAMPS): %/.venv/.installed: %/requirements-dev.txt
	$(PYTHON) -m venv $*/.venv
	$*/.venv/bin/pip install --quiet -r $*/requirements-dev.txt
	@touch $@

# ---- Go stack ----

fmt-go: $(TOOLS_STAMP)
	@$(call for_each_go,fmt-go,$(GOLANGCI_LINT) fmt)

fmt-check-go: $(TOOLS_STAMP)
	@$(call for_each_go,fmt-check-go,$(GOLANGCI_LINT) fmt --diff)

tidy:
	@$(call for_each_go,tidy,$(GO) mod tidy)

tidy-check:
	@$(call for_each_go,tidy-check,$(GO) mod tidy -diff)

lint-go: $(TOOLS_STAMP)
	@$(call for_each_go,lint-go,$(GOLANGCI_LINT) run ./...)

vet:
	@$(call for_each_go,vet,$(GO) vet ./...)

arch: $(TOOLS_STAMP)
	@$(call for_each_go,arch,$(GO_ARCH_LINT) check)

test-go:
	@$(call for_each_go,test-go,$(GO) test -race ./...)

build-go:
	@$(call for_each_go,build-go,$(GO) build ./...)

cover-go:
	@set -e; for m in $(GO_MODULES); do \
		echo "==> cover $$m"; \
		( cd $$m && $(GO) test -coverprofile=$(COVER_PROFILE) ./... && \
			grep -vE '$(COVER_EXCLUDE)' $(COVER_PROFILE) > $(COVER_PROFILE).gated; \
			$(GO) tool cover -func=$(COVER_PROFILE).gated ); \
	done

cover-check-go:
	@echo "==> cover-check-go (min $(COVERAGE_MIN)%)"; \
	for m in $(GO_MODULES); do \
		if ! out=$$( cd $$m && $(GO) test -race -coverprofile=$(COVER_PROFILE) ./... >/dev/null && \
			grep -vE '$(COVER_EXCLUDE)' $(COVER_PROFILE) > $(COVER_PROFILE).gated; \
			stmts=$$(awk 'NR > 1 { sum += $$2 } END { print sum + 0 }' $(COVER_PROFILE).gated); \
			if [ "$$stmts" -eq 0 ]; then echo "    no statements to cover"; exit 0; fi; \
			total=$$($(GO) tool cover -func=$(COVER_PROFILE).gated | \
				awk '/^total:/ { gsub(/%/, "", $$3); print $$3 }'); \
			echo "    total: $${total:-0}%"; \
			awk -v c="$${total:-0}" -v min="$(COVERAGE_MIN)" \
				'BEGIN { if (c + 0 < min + 0) { exit 1 } }' || \
				{ echo "coverage $${total:-0}% below $(COVERAGE_MIN)% in $$m"; exit 1; } ) 2>&1; then \
			echo "==> cover-check-go $$m FAILED"; echo "$$out"; exit 1; \
		fi; \
	done

# ---- Python stack ----

fmt-py: $(TOOLS_STAMP)
	@$(call for_each_py,fmt-py,$(RUFF) format .)

fmt-check-py: $(TOOLS_STAMP)
	@$(call for_each_py,fmt-check-py,$(RUFF) format --check .)

lint-py: $(TOOLS_STAMP)
	@$(call for_each_py,lint-py,$(RUFF) check .)

test-py: $(PY_VENV_STAMPS)
	@$(call for_each_py,test-py,.venv/bin/python -m pytest -q)

cover-py: $(PY_VENV_STAMPS)
	@$(call for_each_py,cover-py,.venv/bin/python -m pytest -q --cov --cov-report=term-missing)

cover-check-py: $(PY_VENV_STAMPS)
	@$(call for_each_py,cover-check-py,.venv/bin/python -m pytest -q --cov --cov-fail-under=$(COVERAGE_MIN))

# ---- misc ----

peer-hash:
	cd services/yacynode && $(GO) run ./cmd/yacy-peer-hash

# ---- e2e ----

E2E_TIMEOUT ?= 10m

E2E_CONTAINER_CLI := $(shell command -v docker >/dev/null 2>&1 && echo docker || \
	(command -v podman >/dev/null 2>&1 && echo podman || echo "distrobox-host-exec podman"))
E2E_RUNTIME_DIR := $(or $(XDG_RUNTIME_DIR),/run/user/$(shell id -u))
E2E_DOCKER_HOST := $(or $(DOCKER_HOST),unix://$(E2E_RUNTIME_DIR)/podman/podman.sock)
E2E_DOCKER_ENV := DOCKER_HOST=$(E2E_DOCKER_HOST) TESTCONTAINERS_RYUK_DISABLED=true

# Modules that build a docker image for e2e testing, and the tag each produces.
E2E_IMAGE_MODULES := yacynode yacycrawler yacytextindexer corpusmarkdown yacyvisitcrawl renderproxy

E2E_PATH_yacynode        := services/yacynode
E2E_PATH_yacycrawler     := services/yacycrawler
E2E_PATH_yacytextindexer := services/yacytextindexer
E2E_PATH_corpusmarkdown  := services/corpusmarkdown
E2E_PATH_yacyvisitcrawl  := services/yacyvisitcrawl
E2E_PATH_renderproxy     := services/renderproxy

E2E_IMAGE_yacynode        := yacy-rwi-node:e2e
E2E_IMAGE_yacycrawler     := yacy-rwi-crawler:e2e
E2E_IMAGE_yacytextindexer := yacy-rwi-textindexer:e2e
E2E_IMAGE_corpusmarkdown  := corpusmarkdown:e2e
E2E_IMAGE_yacyvisitcrawl  := yacyvisitcrawl:e2e
E2E_IMAGE_renderproxy     := renderproxy:e2e

define e2e_image_rule
e2e-$(1)-image:
	DOCKER_BUILDKIT=1 $$(E2E_CONTAINER_CLI) build -f $$(E2E_PATH_$(1))/Dockerfile -t $$(E2E_IMAGE_$(1)) .
endef
$(foreach m,$(E2E_IMAGE_MODULES),$(eval $(call e2e_image_rule,$(m))))

e2e-images: $(foreach m,$(E2E_IMAGE_MODULES),e2e-$(m)-image)

# Modules that own a test/e2e suite, and the images each suite needs.
E2E_SUITE_MODULES := yacynode yacycrawler yacytextindexer corpusmarkdown searxng-result-router searxng-crawled-text-search renderproxy

E2E_PATH_searxng-result-router         := plugins/searxng/searxng-result-router
E2E_PATH_searxng-crawled-text-search   := plugins/searxng/searxng-crawled-text-search

E2E_ENV_yacynode                       := YACY_NODE_IMAGE=$(E2E_IMAGE_yacynode)
E2E_ENV_yacycrawler                    := YACYCRAWLER_IMAGE=$(E2E_IMAGE_yacycrawler)
E2E_ENV_yacytextindexer                := YACY_NODE_IMAGE=$(E2E_IMAGE_yacynode) YACYCRAWLER_IMAGE=$(E2E_IMAGE_yacycrawler) YACYTEXTINDEXER_IMAGE=$(E2E_IMAGE_yacytextindexer)
E2E_ENV_corpusmarkdown                 := YACY_NODE_IMAGE=$(E2E_IMAGE_yacynode) YACYCRAWLER_IMAGE=$(E2E_IMAGE_yacycrawler) CORPUSMARKDOWN_IMAGE=$(E2E_IMAGE_corpusmarkdown)
E2E_ENV_searxng-result-router          := YACYVISITCRAWL_IMAGE=$(E2E_IMAGE_yacyvisitcrawl)
E2E_ENV_searxng-crawled-text-search    :=
E2E_ENV_renderproxy                    := RENDERPROXY_IMAGE=$(E2E_IMAGE_renderproxy)

define e2e_suite_rule
e2e-$(1):
	@echo "==> e2e-$(1)"; \
	if ! out=$$$$(cd $$(E2E_PATH_$(1))/test/e2e && GOWORK=off $$(E2E_DOCKER_ENV) $$(E2E_ENV_$(1)) \
		$$(GO) test -tags e2e -timeout $$(E2E_TIMEOUT) -count=1 -v ./... 2>&1); then \
		echo "==> e2e-$(1) FAILED"; echo "$$$$out"; exit 1; \
	fi
endef
$(foreach m,$(E2E_SUITE_MODULES),$(eval $(call e2e_suite_rule,$(m))))

e2e: $(foreach m,$(E2E_SUITE_MODULES),e2e-$(m))
	@echo "==> e2e SUCCESS"
