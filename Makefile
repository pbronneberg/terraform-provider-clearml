SHELL := /bin/sh

DEFAULT_TERRAFORM_VERSION := 1.15.8
TERRAFORM_VERSION ?= $(DEFAULT_TERRAFORM_VERSION)
TERRAFORM_IMAGE_1_15_8 := hashicorp/terraform:1.15.8@sha256:7ae513256f7ce67879e218ae8593d6fbe216ec9e123abe6c94e4e10704857963
TERRAFORM_IMAGE_1_6_6 := hashicorp/terraform:1.6.6@sha256:9a42ea97ea25b363f4c65be25b9ca52b1e511ea5bf7d56050a506ad2daa7af9d
TERRAFORM_IMAGE ?= $(TERRAFORM_IMAGE_$(subst .,_,$(TERRAFORM_VERSION)))

ifeq ($(strip $(TERRAFORM_IMAGE)),)
$(error Unsupported TERRAFORM_VERSION $(TERRAFORM_VERSION). Set TERRAFORM_IMAGE to an immutable Terraform image reference to use another version.)
endif

DOCKER ?= docker
DOCKER_BUILD ?= DOCKER_BUILDKIT=1 $(DOCKER) build
CONTAINER_MAKE ?= make
DEVCONTAINER_BUILD_FLAGS ?=
DEVCONTAINER_IMAGE ?= terraform-provider-clearml-devcontainer:terraform-$(subst .,-,$(TERRAFORM_VERSION))
DEVCONTAINER_CACHE_DIR ?= $(CURDIR)/.cache/devcontainer
WORKSPACE_DIR := /workspace/terraform-provider-clearml
USER_ID := $(shell id -u)
GROUP_ID := $(shell id -g)
CLEARML_ENV := --env CLEARML_API_URL --env CLEARML_ACCESS_KEY --env CLEARML_SECRET_KEY

.DEFAULT_GOAL := help

.PHONY: help image image-refresh image-ensure cache-dirs check-docker terraform-version test build generate generate-check security testacc cleanupacc

help:
	@printf '%s\n' \
		'Available targets:' \
		'  image                 Build the selected devcontainer image.' \
		'  image-refresh         Rebuild the selected devcontainer image from fresh layers.' \
		'  terraform-version     Print the Terraform version in the devcontainer.' \
		'  test                  Run race-enabled Go tests.' \
		'  build                 Build the provider.' \
		'  generate              Regenerate Terraform provider documentation.' \
		'  generate-check        Regenerate documentation and fail on an uncommitted diff.' \
		'  security              Run Go vulnerability, OSV, and waiver-policy checks.' \
		'  testacc               Run credentialed ClearML acceptance tests.' \
		'  cleanupacc            Delete CI-owned acceptance queues older than 24 hours.' \
		'' \
		'All development targets run in the devcontainer. Set TERRAFORM_VERSION=1.6.6' \
		'to select the supported minimum Terraform version. Set TERRAFORM_IMAGE to an' \
		'immutable image reference to use another Terraform version.'

ifeq ($(IN_DEVCONTAINER),1)

terraform-version:
	terraform version

test:
	go test -race ./...

build:
	go build ./...

generate:
	go generate ./...

generate-check: generate
	git diff --compact-summary --exit-code -- docs examples

security:
	@set -eu; \
	result='$(OSV_OUTPUT)'; \
	cleanup=0; \
	if [ -z "$$result" ]; then result="$$(mktemp)"; cleanup=1; fi; \
	trap '[ "$$cleanup" -eq 1 ] && rm -f "$$result"' EXIT; \
	go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...; \
	go run github.com/google/osv-scanner/v2/cmd/osv-scanner@v2.5.0 scan source --format json --output-file "$$result" . || true; \
	python3 scripts/check_vulnerability_waivers.py "$$result" .security/vulnerability-waivers.yaml

testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

cleanupacc:
	go run ./cmd/clearml-acceptance-cleanup

else

check-docker:
	@command -v $(DOCKER) >/dev/null 2>&1 || { \
		printf '%s\n' 'Docker is required to run Make targets outside the devcontainer.'; \
		printf '%s\n' 'Install Docker or set DOCKER to a compatible container runtime command.'; \
		exit 127; \
	}

cache-dirs:
	@mkdir -p "$(DEVCONTAINER_CACHE_DIR)/go-build" "$(DEVCONTAINER_CACHE_DIR)/go/pkg/mod"

image: check-docker
	$(DOCKER_BUILD) $(DEVCONTAINER_BUILD_FLAGS) \
		--build-arg TERRAFORM_VERSION=$(TERRAFORM_VERSION) \
		--build-arg TERRAFORM_IMAGE=$(TERRAFORM_IMAGE) \
		--tag $(DEVCONTAINER_IMAGE) \
		--file .devcontainer/Dockerfile .

image-refresh: check-docker
	$(DOCKER_BUILD) --no-cache --pull $(DEVCONTAINER_BUILD_FLAGS) \
		--build-arg TERRAFORM_VERSION=$(TERRAFORM_VERSION) \
		--build-arg TERRAFORM_IMAGE=$(TERRAFORM_IMAGE) \
		--tag $(DEVCONTAINER_IMAGE) \
		--file .devcontainer/Dockerfile .

image-ensure: check-docker
	@if $(DOCKER) image inspect "$(DEVCONTAINER_IMAGE)" >/dev/null 2>&1; then \
		printf 'Using prepared devcontainer image %s\n' "$(DEVCONTAINER_IMAGE)"; \
	else \
		$(MAKE) --no-print-directory image; \
	fi

define RUN_IN_DEVCONTAINER
	$(DOCKER) run --rm --init \
		--user $(USER_ID):$(GROUP_ID) \
		--env HOME=/tmp \
		--env GOPATH=/cache/go \
		--env GOCACHE=/cache/go-build \
		--env GOMODCACHE=/cache/go/pkg/mod \
		--env IN_DEVCONTAINER=1 \
		$(CLEARML_ENV) \
		--volume "$(CURDIR):$(WORKSPACE_DIR)" \
		--volume "$(DEVCONTAINER_CACHE_DIR):/cache" \
		--workdir $(WORKSPACE_DIR) \
		$(DEVCONTAINER_IMAGE) \
		$(CONTAINER_MAKE) --no-print-directory
endef

terraform-version test build generate generate-check security testacc cleanupacc: image-ensure cache-dirs
	$(RUN_IN_DEVCONTAINER) $@

endif
