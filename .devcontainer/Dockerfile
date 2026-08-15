# syntax=docker/dockerfile:1.7

ARG TERRAFORM_VERSION=1.15.8
ARG TERRAFORM_IMAGE=hashicorp/terraform:1.15.8@sha256:7ae513256f7ce67879e218ae8593d6fbe216ec9e123abe6c94e4e10704857963

FROM ${TERRAFORM_IMAGE} AS terraform

FROM golang:1.26.6-bookworm@sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36

ARG TERRAFORM_VERSION

LABEL org.opencontainers.image.title="terraform-provider-clearml development environment" \
	org.opencontainers.image.description="Go and Terraform toolchain for terraform-provider-clearml" \
	org.opencontainers.image.version="terraform-${TERRAFORM_VERSION}"

RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
		ca-certificates \
		git \
		make \
		python3 \
	&& rm -rf /var/lib/apt/lists/*

RUN groupadd --gid 1000 vscode \
	&& useradd --uid 1000 --gid vscode --create-home --shell /bin/bash vscode

COPY --from=terraform /bin/terraform /usr/local/bin/terraform

WORKDIR /workspace/terraform-provider-clearml
COPY --chown=vscode:vscode go.mod go.sum ./

USER vscode
RUN --mount=type=cache,target=/home/vscode/go/pkg/mod,uid=1000,gid=1000 \
	go mod download

ENV GOPATH=/home/vscode/go \
	PATH=/home/vscode/go/bin:${PATH}
