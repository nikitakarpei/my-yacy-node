GO ?= $(CURDIR)/.toolchain/go/bin/go
PYTHON ?= python3
COVERAGE_MIN ?= 80

# make workspace writes go.work for the editor; every target here builds each module as a standalone consumer sees it.
export GOWORK := off

GO_MODULES := $(patsubst %/go.mod,%,$(wildcard libraries/*/go.mod libraries/*/*/go.mod services/*/go.mod services/*/contract/go.mod test/contracts/*/go.mod))
GO_E2E_MODULES := $(patsubst %/go.mod,%,$(wildcard services/*/test/e2e/go.mod plugins/*/*/test/e2e/go.mod test/*/go.mod))
PY_MODULES := plugins/searxng/searxng-result-router plugins/searxng/searxng-crawled-text-search

COVER_PROFILE := coverage.out
COVER_PATTERN := $(CURDIR)/tools/covignore-pattern
COVER_GATE := $(CURDIR)/tools/gate-coverage
COVER_PACKAGES := $(CURDIR)/tools/coverage-packages
# -count=1 because the test cache merges stale and fresh coverage under -coverpkg (golang/go#74873).
COVER_FLAGS := -count=1

export GO COVERAGE_MIN COVER_PROFILE COVER_PATTERN COVER_PACKAGES COVER_FLAGS

JOBS ?= $(shell nproc 2>/dev/null || echo 4)

ARCH_DIAGRAM_DIR := $(CURDIR)/arch-diagrams

TOOLS_BIN := $(CURDIR)/.toolchain/bin
TOOLS_STAMP := $(TOOLS_BIN)/.installed
GOLANGCI_LINT := $(TOOLS_BIN)/golangci-lint
GO_ARCH_LINT := $(TOOLS_BIN)/go-arch-lint
RUFF := $(TOOLS_BIN)/ruff
MADO := $(TOOLS_BIN)/mado

PY_VENV_STAMPS := $(foreach m,$(PY_MODULES),$(m)/.venv/.installed)

define for_each_go
echo "==> $(1)"; \
printf '%s\n' $(GO_MODULES) | xargs -P $(JOBS) -I{} sh -c \
	'if ! out=$$(cd {} && $(2) 2>&1); then \
		echo "==> $(1) {} FAILED"; echo "$$out"; exit 255; \
	fi'
endef

define for_each_go_e2e
echo "==> $(1)"; \
for m in $(GO_E2E_MODULES); do \
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
	fmt fmt-go fmt-go-e2e fmt-py \
	fmt-check fmt-check-go fmt-check-go-e2e fmt-check-py \
	tidy tidy-go tidy-go-e2e \
	tidy-check tidy-check-go tidy-check-go-e2e \
	workspace \
	lint lint-go lint-go-e2e lint-py lint-md \
	arch arch-diagram \
	test test-go test-py \
	cover cover-go cover-py \
	cover-check cover-check-go cover-check-py \
	build build-go verify peer-hash \
	proto \
	e2e e2e-images

fmt:         fmt-go fmt-go-e2e fmt-py
fmt-check:   fmt-check-go fmt-check-go-e2e fmt-check-py
tidy:        tidy-go tidy-go-e2e
tidy-check:  tidy-check-go tidy-check-go-e2e
lint:        lint-go lint-go-e2e lint-py lint-md
test:        test-go test-py
cover:       cover-go cover-py
cover-check: cover-check-go cover-check-py
build:       build-go
verify:      fmt-check tidy-check lint arch cover-check build
	@echo "==> verify SUCCESS"

$(TOOLS_STAMP): tools/install tools/tools.lock
	./tools/install
	@touch $@

tools: $(TOOLS_STAMP)

PROTOC := $(TOOLS_BIN)/protoc
PROTOC_INCLUDE := $(CURDIR)/.toolchain/include
PROTO_GEN_GO := $(TOOLS_BIN)/protoc-gen-go
PROTO_GEN_GO_GRPC := $(TOOLS_BIN)/protoc-gen-go-grpc
CORPUSMARKDOWN_API_DIR := services/corpusmarkdown/contract

proto: $(TOOLS_STAMP)
	@echo "==> proto"
	@PATH="$(TOOLS_BIN):$$PATH" $(PROTOC) \
		--proto_path=$(PROTOC_INCLUDE) \
		--proto_path=$(CORPUSMARKDOWN_API_DIR) \
		--go_out=$(CORPUSMARKDOWN_API_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(CORPUSMARKDOWN_API_DIR) --go-grpc_opt=paths=source_relative \
		corpusmarkdown/v1/markdowncorpus.proto

$(PY_VENV_STAMPS): %/.venv/.installed: %/requirements-dev.txt
	$(PYTHON) -m venv $*/.venv
	$*/.venv/bin/pip install --quiet -r $*/requirements-dev.txt
	@touch $@

# ---- Go stack ----

fmt-go: $(TOOLS_STAMP)
	@$(call for_each_go,fmt-go,$(GOLANGCI_LINT) fmt)

fmt-check-go: $(TOOLS_STAMP)
	@$(call for_each_go,fmt-check-go,$(GOLANGCI_LINT) fmt --diff)

fmt-go-e2e: $(TOOLS_STAMP)
	@$(call for_each_go_e2e,fmt-go-e2e,$(GOLANGCI_LINT) fmt)

fmt-check-go-e2e: $(TOOLS_STAMP)
	@$(call for_each_go_e2e,fmt-check-go-e2e,$(GOLANGCI_LINT) fmt --diff)

tidy-go: $(TOOLS_STAMP)
	@$(call for_each_go,tidy-go,$(GO) mod tidy)

tidy-go-e2e: $(TOOLS_STAMP)
	@$(call for_each_go_e2e,tidy-go-e2e,$(GO) mod tidy)

tidy-check-go: $(TOOLS_STAMP)
	@$(call for_each_go,tidy-check-go,$(GO) mod tidy -diff)

tidy-check-go-e2e: $(TOOLS_STAMP)
	@$(call for_each_go_e2e,tidy-check-go-e2e,$(GO) mod tidy -diff)

lint-go: $(TOOLS_STAMP)
	@$(call for_each_go,lint-go,$(GOLANGCI_LINT) run --allow-parallel-runners ./...)

lint-go-e2e: $(TOOLS_STAMP)
	@$(call for_each_go_e2e,lint-go-e2e,$(GOLANGCI_LINT) run --build-tags e2e ./...)

arch: $(TOOLS_STAMP)
	@$(call for_each_go,arch,$(GO_ARCH_LINT) check)

arch-diagram: $(TOOLS_STAMP)
	@mkdir -p $(ARCH_DIAGRAM_DIR)
	@$(call for_each_go,arch-diagram,$(GO_ARCH_LINT) graph \
		--out $(ARCH_DIAGRAM_DIR)/$$(echo {} | tr / -).svg)
	@echo "    written to $(ARCH_DIAGRAM_DIR)/"

test-go: $(TOOLS_STAMP)
	@$(call for_each_go,test-go,$(GO) test -race ./...)

build-go: $(TOOLS_STAMP)
	@$(call for_each_go,build-go,$(GO) build ./...)

cover-go: $(TOOLS_STAMP)
	@set -e; for m in $(GO_MODULES); do \
		echo "==> cover $$m"; \
		( cd $$m && $(GO) test -coverpkg=$$($(COVER_PACKAGES)) $(COVER_FLAGS) -coverprofile=$(COVER_PROFILE) ./... && \
			grep -vE "$$($(COVER_PATTERN))" $(COVER_PROFILE) > $(COVER_PROFILE).gated; \
			$(GO) tool cover -func=$(COVER_PROFILE).gated ); \
	done

cover-check-go: $(TOOLS_STAMP)
	@$(call for_each_go,cover-check-go,$(COVER_GATE))

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

# ---- Markdown ----

lint-md: $(TOOLS_STAMP)
	@echo "==> lint-md"
	@if ! out=$$(git ls-files -z '*.md' | xargs -0 $(MADO) check 2>&1); then \
		echo "$$out"; exit 1; \
	fi

# ---- misc ----

workspace: $(TOOLS_STAMP)
	@echo "==> workspace"
	@rm -f go.work go.work.sum
	@GOWORK= $(GO) work init $(GO_MODULES)

peer-hash: $(TOOLS_STAMP)
	cd services/yacynode && $(GO) run ./cmd/yacy-peer-hash

# ---- e2e ----

E2E_TIMEOUT ?= 10m

E2E_CONTAINER_CLI := $(shell command -v docker >/dev/null 2>&1 && echo docker || \
	(command -v podman >/dev/null 2>&1 && echo podman || echo "distrobox-host-exec podman"))
E2E_RUNTIME_DIR := $(or $(XDG_RUNTIME_DIR),/run/user/$(shell id -u))
E2E_DOCKER_HOST := $(or $(DOCKER_HOST),unix://$(E2E_RUNTIME_DIR)/podman/podman.sock)
E2E_DOCKER_ENV := DOCKER_HOST=$(E2E_DOCKER_HOST) TESTCONTAINERS_RYUK_DISABLED=true

# Modules that build a docker image for e2e testing, and the tag each produces.
E2E_IMAGE_MODULES := yacynode yacycrawler corpustext corpusmarkdown visitcrawl renderproxy webarchivescrape webresearchmcp pagescrape yacydhtsearch

E2E_PATH_yacynode        := services/yacynode
E2E_PATH_yacycrawler     := services/yacycrawler
E2E_PATH_corpustext      := services/corpustext
E2E_PATH_corpusmarkdown  := services/corpusmarkdown
E2E_PATH_visitcrawl      := services/visitcrawl
E2E_PATH_renderproxy     := services/renderproxy
E2E_PATH_webarchivescrape := services/webarchivescrape
E2E_PATH_webresearchmcp   := services/webresearchmcp
E2E_PATH_pagescrape       := services/pagescrape
E2E_PATH_yacydhtsearch    := services/yacydhtsearch

E2E_IMAGE_ENV_yacynode        := YACY_NODE_IMAGE
E2E_IMAGE_ENV_yacycrawler     := YACYCRAWLER_IMAGE
E2E_IMAGE_ENV_corpustext      := CORPUSTEXT_IMAGE
E2E_IMAGE_ENV_corpusmarkdown  := CORPUSMARKDOWN_IMAGE
E2E_IMAGE_ENV_visitcrawl      := VISITCRAWL_IMAGE
E2E_IMAGE_ENV_renderproxy     := RENDERPROXY_IMAGE
E2E_IMAGE_ENV_webarchivescrape := WEBARCHIVESCRAPE_IMAGE
E2E_IMAGE_ENV_webresearchmcp   := WEBRESEARCHMCP_IMAGE
E2E_IMAGE_ENV_pagescrape       := PAGESCRAPE_IMAGE
E2E_IMAGE_ENV_yacydhtsearch    := YACYDHTSEARCH_IMAGE

E2E_IMAGE_yacynode        := yacy-rwi-node:e2e
E2E_IMAGE_yacycrawler     := yacy-rwi-crawler:e2e
E2E_IMAGE_corpustext      := corpustext:e2e
E2E_IMAGE_corpusmarkdown  := corpusmarkdown:e2e
E2E_IMAGE_visitcrawl      := visitcrawl:e2e
E2E_IMAGE_renderproxy     := renderproxy:e2e
E2E_IMAGE_webarchivescrape := webarchivescrape:e2e
E2E_IMAGE_webresearchmcp   := webresearchmcp:e2e
E2E_IMAGE_pagescrape       := pagescrape:e2e
E2E_IMAGE_yacydhtsearch    := yacydhtsearch:e2e

define e2e_image_rule
.PHONY: e2e-$(1)-image
e2e-$(1)-image:
	DOCKER_BUILDKIT=1 $$(E2E_CONTAINER_CLI) build -f $$(E2E_PATH_$(1))/Dockerfile -t $$(E2E_IMAGE_$(1)) .
endef
$(foreach m,$(E2E_IMAGE_MODULES),$(eval $(call e2e_image_rule,$(m))))

e2e-images: $(foreach m,$(E2E_IMAGE_MODULES),e2e-$(m)-image)

# Every e2e suite, where it lives, and the images it needs.
E2E_SUITE_MODULES := yacynode yacycrawler corpustext corpusmarkdown searxng-result-router searxng-crawled-text-search renderproxy webarchivescrape webresearchmcp pageofferfanout pagescrape yacydhtsearch

E2E_PATH_searxng-result-router         := plugins/searxng/searxng-result-router
E2E_PATH_searxng-crawled-text-search   := plugins/searxng/searxng-crawled-text-search
E2E_SUITE_DIR_pageofferfanout          := test/pageofferfanout

E2E_SUITE_IMAGES_yacynode                    := yacynode
E2E_SUITE_IMAGES_yacycrawler                 := yacycrawler
E2E_SUITE_IMAGES_corpustext                  := yacynode yacycrawler corpustext pagescrape
E2E_SUITE_IMAGES_corpusmarkdown              := yacynode yacycrawler corpusmarkdown pagescrape
E2E_SUITE_IMAGES_searxng-result-router       := visitcrawl
E2E_SUITE_IMAGES_searxng-crawled-text-search := corpustext pagescrape
E2E_SUITE_IMAGES_renderproxy                 := renderproxy
E2E_SUITE_IMAGES_webarchivescrape             := webarchivescrape corpustext pagescrape
E2E_SUITE_IMAGES_webresearchmcp              := corpusmarkdown webresearchmcp pagescrape
E2E_SUITE_IMAGES_pageofferfanout             := corpustext corpusmarkdown pagescrape
E2E_SUITE_IMAGES_pagescrape                  := pagescrape
E2E_SUITE_IMAGES_yacydhtsearch               := yacydhtsearch

# A suite reads the tag of each image it needs from that image's env var.
e2e_suite_image_env = $(foreach i,$(E2E_SUITE_IMAGES_$(1)),$(E2E_IMAGE_ENV_$(i))=$(E2E_IMAGE_$(i)))
e2e_suite_image_targets = $(foreach i,$(E2E_SUITE_IMAGES_$(1)),e2e-$(i)-image)

define e2e_suite_rule
.PHONY: e2e-$(1)
e2e-$(1): $$(TOOLS_STAMP) $$(call e2e_suite_image_targets,$(1))
	@echo "==> e2e-$(1)"; \
	if ! out=$$$$(cd $$(or $$(E2E_SUITE_DIR_$(1)),$$(E2E_PATH_$(1))/test/e2e) && GOWORK=off $$(E2E_DOCKER_ENV) $$(call e2e_suite_image_env,$(1)) \
		$$(GO) test -tags e2e -timeout $$(E2E_TIMEOUT) -count=1 -v ./... 2>&1); then \
		echo "==> e2e-$(1) FAILED"; echo "$$$$out"; exit 1; \
	fi
endef
$(foreach m,$(E2E_SUITE_MODULES),$(eval $(call e2e_suite_rule,$(m))))

e2e: $(foreach m,$(E2E_SUITE_MODULES),e2e-$(m))
	@echo "==> e2e SUCCESS"
