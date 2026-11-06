ifneq ($(realpath $(dir $(TOOLCHAIN_DIR))),$(CURDIR))
$(error No toolchain of this checkout is active. Run ./tools/install, then . .toolchain/activate)
endif
VERIFY_TOOLS_STATE := $(shell ./tools/install --check verify)
ifneq ($(VERIFY_TOOLS_STATE),)
$(error $(VERIFY_TOOLS_STATE))
endif

COVERAGE_MIN ?= 80

# make workspace writes go.work for the editor; every target here builds each module as a standalone consumer sees it.
export GOWORK := off

GO_MODULES := $(patsubst %/go.mod,%,$(wildcard libraries/*/go.mod libraries/*/*/go.mod services/*/go.mod services/*/contract/go.mod distributions/*/go.mod test/contracts/*/go.mod))
GO_E2E_MODULES := $(patsubst %/go.mod,%,$(wildcard services/*/test/e2e/go.mod plugins/*/*/test/e2e/go.mod test/*/go.mod))
PY_MODULES := $(patsubst %/pyproject.toml,%,$(wildcard libraries/*/pyproject.toml services/*/pyproject.toml plugins/*/*/pyproject.toml))

COVER_PROFILE := coverage.out
COVER_PATTERN := $(CURDIR)/tools/covignore-pattern
COVER_GATE := $(CURDIR)/tools/gate-coverage
COVER_PACKAGES := $(CURDIR)/tools/coverage-packages
RACE_FLAG ?=

export COVERAGE_MIN COVER_PROFILE COVER_PATTERN COVER_PACKAGES RACE_FLAG

JOBS ?= $(shell nproc 2>/dev/null || echo 4)

ARCH_DIAGRAM_DIR := $(CURDIR)/arch-diagrams

ANDROID_APPS := $(patsubst %/settings.gradle.kts,%,$(wildcard distributions/*android/settings.gradle.kts))
ANDROID_HOST := libraries/androidservicehost

PY_VENV := $(CURDIR)/.venv
PY_VENV_STAMP := $(PY_VENV)/.installed
PY_CONFIG := $(CURDIR)/pyproject.toml

WEB_FILES = git ls-files -z '*.js' '*.css' ':!:*.min.js' ':!:*.min.css'
WEBUISTYLE_STYLESHEET = libraries/webuistyle/static/webuistyle.css
JS_FILES = git ls-files -z '*.js' ':!:*.min.js'

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

.PHONY: \
	fmt fmt-go fmt-go-e2e fmt-py fmt-web \
	fmt-check fmt-check-go fmt-check-go-e2e fmt-check-py fmt-check-web \
	tidy tidy-go tidy-go-e2e tidy-py \
	tidy-check tidy-check-go tidy-check-go-e2e tidy-check-py \
	workspace \
	lint lint-go lint-go-e2e lint-py lint-md lint-js lint-css \
	arch arch-go arch-py arch-diagram \
	test test-go test-py \
	cover cover-go cover-py \
	cover-check cover-check-go cover-check-py \
	build build-go verify peer-key refresh-asmap \
	e2e e2e-images android-apps android-apps-verification-metadata android-apps-reproducible

fmt:         fmt-go fmt-go-e2e fmt-py fmt-web
fmt-check:   fmt-check-go fmt-check-go-e2e fmt-check-py fmt-check-web
tidy:        tidy-go tidy-go-e2e tidy-py
tidy-check:  tidy-check-go tidy-check-go-e2e tidy-check-py
lint:        lint-go lint-go-e2e lint-py lint-md lint-js lint-css
arch:        arch-go arch-py
test:        test-go test-py
cover:       cover-go cover-py
cover-check: cover-check-go cover-check-py
build:       build-go
verify:      fmt-check tidy-check lint arch cover-check build
	@echo "==> verify SUCCESS"

android-apps:
	@./tools/install --check android
	@for app in $(ANDROID_APPS); do \
		$(ANDROID_HOST)/gradlew -p "$$app" assembleRelease lintRelease ktlintCheck --console=plain || exit 1; \
	done

android-apps-verification-metadata:
	@./tools/install --check android
	@for app in $(ANDROID_APPS); do \
		$(ANDROID_HOST)/update-verification-metadata "$$app" || exit 1; \
	done

android-apps-reproducible:
	@./tools/install --check android
	$(ANDROID_HOST)/check-reproducible $(ANDROID_APPS)

$(PY_VENV_STAMP): .python-version pyproject.toml uv.lock $(PY_MODULES:%=%/pyproject.toml)
	uv sync --locked --all-packages --quiet
	@touch $@

# ---- Go stack ----

fmt-go:
	@$(call for_each_go,fmt-go,golangci-lint fmt)

fmt-check-go:
	@$(call for_each_go,fmt-check-go,golangci-lint fmt --diff)

fmt-go-e2e:
	@$(call for_each_go_e2e,fmt-go-e2e,golangci-lint fmt)

fmt-check-go-e2e:
	@$(call for_each_go_e2e,fmt-check-go-e2e,golangci-lint fmt --diff)

tidy-go:
	@$(call for_each_go,tidy-go,go mod tidy)

tidy-go-e2e:
	@$(call for_each_go_e2e,tidy-go-e2e,go mod tidy)

tidy-check-go:
	@$(call for_each_go,tidy-check-go,go mod tidy -diff)

tidy-check-go-e2e:
	@$(call for_each_go_e2e,tidy-check-go-e2e,go mod tidy -diff)

lint-go:
	@$(call for_each_go,lint-go,golangci-lint run --allow-parallel-runners ./...)

lint-go-e2e:
	@$(call for_each_go_e2e,lint-go-e2e,golangci-lint run --build-tags e2e ./...)

arch-go:
	@$(call for_each_go,arch-go,go-arch-lint check)

arch-diagram:
	@mkdir -p $(ARCH_DIAGRAM_DIR)
	@$(call for_each_go,arch-diagram,go-arch-lint graph \
		--out $(ARCH_DIAGRAM_DIR)/$$(echo {} | tr / -).svg)
	@echo "    written to $(ARCH_DIAGRAM_DIR)/"

test-go:
	@$(call for_each_go,test-go,go test -race ./...)

build-go:
	@$(call for_each_go,build-go,go build ./...)

cover-go:
	@set -e; for m in $(GO_MODULES); do \
		echo "==> cover $$m"; \
		( cd $$m && go test -coverpkg=$$($(COVER_PACKAGES)) -coverprofile=$(COVER_PROFILE) ./... && \
			grep -vE "$$($(COVER_PATTERN))" $(COVER_PROFILE) > $(COVER_PROFILE).gated; \
			go tool cover -func=$(COVER_PROFILE).gated ); \
	done

cover-check-go:
	@$(call for_each_go,cover-check-go,$(COVER_GATE))

# ---- Python stack ----

fmt-py:
	@$(call for_each_py,fmt-py,ruff format .)

fmt-check-py:
	@$(call for_each_py,fmt-check-py,ruff format --check .)

tidy-py:
	@echo "==> tidy-py"
	@uv lock --quiet

tidy-check-py:
	@echo "==> tidy-check-py"
	@uv lock --check --quiet

lint-py: $(PY_VENV_STAMP)
	@$(call for_each_py,lint-py,ruff check . && $(PY_VENV)/bin/mypy --config-file $(PY_CONFIG) .)

arch-py: $(PY_VENV_STAMP)
	@$(call for_each_py,arch-py,$(PY_VENV)/bin/tach check && $(PY_VENV)/bin/tach check-external)

test-py: $(PY_VENV_STAMP)
	@$(call for_each_py,test-py,$(PY_VENV)/bin/python -m pytest -q)

cover-py: $(PY_VENV_STAMP)
	@$(call for_each_py,cover-py,$(PY_VENV)/bin/python -m pytest -q --cov --cov-report=term-missing)

cover-check-py: $(PY_VENV_STAMP)
	@$(call for_each_py,cover-check-py,$(PY_VENV)/bin/python -m pytest -q --cov --cov-fail-under=$(COVERAGE_MIN))

# ---- Markdown ----

lint-md:
	@echo "==> lint-md"
	@if ! out=$$(git ls-files -z '*.md' | xargs -0 mado check 2>&1); then \
		echo "$$out"; exit 1; \
	fi

# ---- Web assets ----

fmt-web:
	@echo "==> fmt-web"
	@$(WEB_FILES) | xargs -0 biome format --write --no-errors-on-unmatched

fmt-check-web:
	@echo "==> fmt-check-web"
	@if ! out=$$($(WEB_FILES) | xargs -0 biome format --no-errors-on-unmatched 2>&1); then \
		echo "$$out"; exit 1; \
	fi

lint-js:
	@echo "==> lint-js"
	@out=$$($(JS_FILES) | xargs -0 -I{} esbuild {} --target=es2022 --log-level=warning 2>&1 >/dev/null); \
	if [ -n "$$out" ]; then echo "$$out"; exit 1; fi

lint-css:
	@echo "==> lint-css"
	@out=$$(git ls-files -z '*.css' ':!:*.min.css' ':!:$(WEBUISTYLE_STYLESHEET)' \
		| xargs -0 -r grep -l -e 'var(--webui-' | xargs -r grep -nE '#[0-9a-fA-F]{3,8}\b|rgba?\('); \
	if [ -n "$$out" ]; then echo "colours belong in $(WEBUISTYLE_STYLESHEET):"; echo "$$out"; exit 1; fi

# ---- misc ----

workspace:
	@echo "==> workspace"
	@rm -f go.work go.work.sum
	@GOWORK= go work init $(GO_MODULES)

peer-key:
	cd services/yacynode && go run ./cmd/yacy-peer-key

refresh-asmap:
	./tools/refresh-asmap

# ---- e2e ----

E2E_TIMEOUT ?= 10m

E2E_CONTAINER_CLI := $(shell command -v docker >/dev/null 2>&1 && echo docker || \
	(command -v podman >/dev/null 2>&1 && echo podman || echo "distrobox-host-exec podman"))
E2E_RUNTIME_DIR := $(or $(XDG_RUNTIME_DIR),/run/user/$(shell id -u))
E2E_DOCKER_HOST := $(or $(DOCKER_HOST),unix://$(E2E_RUNTIME_DIR)/podman/podman.sock)
E2E_DOCKER_ENV := DOCKER_HOST=$(E2E_DOCKER_HOST) TESTCONTAINERS_RYUK_DISABLED=true
E2E_SEARXNG_ENV := SEARXNG_IMAGE=$(shell cat plugins/searxng/supported-image)

# Modules that build a docker image for e2e testing, and the tag each produces.
E2E_IMAGE_MODULES := yacynode yacycrawler corpustext corpusmarkdown visitcrawl renderproxy webarchivescrape webresearchmcp pagescrape pagerwi yacydhtsearch crawlscrapebridge

E2E_PATH_yacynode        := services/yacynode
E2E_PATH_yacycrawler     := services/yacycrawler
E2E_PATH_corpustext      := services/corpustext
E2E_PATH_corpusmarkdown  := services/corpusmarkdown
E2E_PATH_visitcrawl      := services/visitcrawl
E2E_PATH_renderproxy     := services/renderproxy
E2E_PATH_webarchivescrape := services/webarchivescrape
E2E_PATH_webresearchmcp   := services/webresearchmcp
E2E_PATH_pagescrape       := services/pagescrape
E2E_PATH_pagerwi          := services/pagerwi
E2E_PATH_yacydhtsearch    := services/yacydhtsearch
E2E_PATH_crawlscrapebridge := services/crawlscrapebridge

E2E_IMAGE_ENV_yacynode        := YACY_NODE_IMAGE
E2E_IMAGE_ENV_yacycrawler     := YACYCRAWLER_IMAGE
E2E_IMAGE_ENV_corpustext      := CORPUSTEXT_IMAGE
E2E_IMAGE_ENV_corpusmarkdown  := CORPUSMARKDOWN_IMAGE
E2E_IMAGE_ENV_visitcrawl      := VISITCRAWL_IMAGE
E2E_IMAGE_ENV_renderproxy     := RENDERPROXY_IMAGE
E2E_IMAGE_ENV_webarchivescrape := WEBARCHIVESCRAPE_IMAGE
E2E_IMAGE_ENV_webresearchmcp   := WEBRESEARCHMCP_IMAGE
E2E_IMAGE_ENV_pagescrape       := PAGESCRAPE_IMAGE
E2E_IMAGE_ENV_pagerwi          := PAGERWI_IMAGE
E2E_IMAGE_ENV_yacydhtsearch    := YACYDHTSEARCH_IMAGE
E2E_IMAGE_ENV_crawlscrapebridge := CRAWLSCRAPEBRIDGE_IMAGE

E2E_IMAGE_yacynode        := yacy-rwi-node:e2e
E2E_IMAGE_yacycrawler     := yacy-rwi-crawler:e2e
E2E_IMAGE_corpustext      := corpustext:e2e
E2E_IMAGE_corpusmarkdown  := corpusmarkdown:e2e
E2E_IMAGE_visitcrawl      := visitcrawl:e2e
E2E_IMAGE_renderproxy     := renderproxy:e2e
E2E_IMAGE_webarchivescrape := webarchivescrape:e2e
E2E_IMAGE_webresearchmcp   := webresearchmcp:e2e
E2E_IMAGE_pagescrape       := pagescrape:e2e
E2E_IMAGE_pagerwi          := pagerwi:e2e
E2E_IMAGE_yacydhtsearch    := yacydhtsearch:e2e
E2E_IMAGE_crawlscrapebridge := crawlscrapebridge:e2e

define e2e_image_rule
.PHONY: e2e-$(1)-image
e2e-$(1)-image:
	DOCKER_BUILDKIT=1 $$(E2E_CONTAINER_CLI) build -f $$(E2E_PATH_$(1))/Dockerfile -t $$(E2E_IMAGE_$(1)) .
endef
$(foreach m,$(E2E_IMAGE_MODULES),$(eval $(call e2e_image_rule,$(m))))

e2e-images: $(foreach m,$(E2E_IMAGE_MODULES),e2e-$(m)-image)

# Every e2e suite, where it lives, and the images it needs.
E2E_SUITE_MODULES := yacynode yacycrawler corpustext corpusmarkdown searxng-result-router searxnglocalindex renderproxy webarchivescrape webresearchmcp pageofferfanout pagerwiintake pagescrape yacydhtsearch

E2E_PATH_searxng-result-router         := plugins/searxng/searxng-result-router
E2E_SUITE_DIR_pageofferfanout          := test/pageofferfanout
E2E_SUITE_DIR_searxnglocalindex        := test/searxnglocalindex
E2E_SUITE_DIR_pagerwiintake            := test/pagerwiintake

E2E_SUITE_IMAGES_yacynode                    := yacynode
E2E_SUITE_IMAGES_yacycrawler                 := yacycrawler crawlscrapebridge
E2E_SUITE_IMAGES_corpustext                  := yacynode yacycrawler corpustext pagescrape crawlscrapebridge
E2E_SUITE_IMAGES_corpusmarkdown              := yacynode yacycrawler corpusmarkdown pagescrape crawlscrapebridge
E2E_SUITE_IMAGES_searxng-result-router       := visitcrawl
E2E_SUITE_IMAGES_searxnglocalindex           := corpustext pagescrape
E2E_SUITE_IMAGES_renderproxy                 := renderproxy
E2E_SUITE_IMAGES_webarchivescrape             := webarchivescrape corpustext pagescrape
E2E_SUITE_IMAGES_webresearchmcp              := corpusmarkdown webresearchmcp pagescrape
E2E_SUITE_IMAGES_pageofferfanout             := corpustext corpusmarkdown pagescrape
E2E_SUITE_IMAGES_pagerwiintake               := pagescrape pagerwi yacynode
E2E_SUITE_IMAGES_pagescrape                  := pagescrape
E2E_SUITE_IMAGES_yacydhtsearch               := yacydhtsearch yacynode

# A suite reads the tag of each image it needs from that image's env var.
e2e_suite_image_env = $(foreach i,$(E2E_SUITE_IMAGES_$(1)),$(E2E_IMAGE_ENV_$(i))=$(E2E_IMAGE_$(i)))
e2e_suite_image_targets = $(foreach i,$(E2E_SUITE_IMAGES_$(1)),e2e-$(i)-image)

define e2e_suite_rule
.PHONY: e2e-$(1)
e2e-$(1): $$(call e2e_suite_image_targets,$(1))
	@echo "==> e2e-$(1)"; \
	if ! out=$$$$(cd $$(or $$(E2E_SUITE_DIR_$(1)),$$(E2E_PATH_$(1))/test/e2e) && GOWORK=off $$(E2E_DOCKER_ENV) $$(E2E_SEARXNG_ENV) $$(call e2e_suite_image_env,$(1)) \
		go test -tags e2e -timeout $$(E2E_TIMEOUT) -count=1 -v ./... 2>&1); then \
		echo "==> e2e-$(1) FAILED"; echo "$$$$out"; exit 1; \
	fi
endef
$(foreach m,$(E2E_SUITE_MODULES),$(eval $(call e2e_suite_rule,$(m))))

e2e: $(foreach m,$(E2E_SUITE_MODULES),e2e-$(m))
	@echo "==> e2e SUCCESS"
