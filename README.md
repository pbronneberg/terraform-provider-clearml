# ClearML Terraform Provider

Terraform provider for managing ClearML queues through the documented ClearML REST API.

## Requirements

- Terraform `>= 1.6`
- Go `1.26.6` for local development
- Docker for Make-driven development outside the devcontainer
- ClearML hosted (`https://api.clear.ml`) or a compatible self-hosted API

The v1 provider uses Terraform Plugin Framework and protocol v6. Existing `clearml_queue` configuration, import IDs, and state attributes (`id`, `name`, and `tags`) are preserved from the SDKv2 provider.

## Configuration

```hcl
provider "clearml" {
  # api_url    = "https://api.clear.ml" # optional
  # access_key = var.clearml_access_key  # optional when environment is set
  # secret_key = var.clearml_secret_key  # optional when environment is set
}
```

When omitted, `api_url` defaults to `https://api.clear.ml`; credentials are read from `CLEARML_ACCESS_KEY` and `CLEARML_SECRET_KEY`. Set `CLEARML_API_URL` to target a self-hosted API.

## Development

Open the repository in a Dev Container to use the pinned Go and Terraform
toolchain, or run the same commands from a host with Docker installed. The
Makefile detects the devcontainer: host invocations build or reuse it, while
commands run inside it execute directly.

```sh
make test
make build
make generate-check
make security
```

The default devcontainer uses Terraform 1.15.8. To run a target with the
supported minimum version, set `TERRAFORM_VERSION=1.6.6`, for example:

```sh
make test TERRAFORM_VERSION=1.6.6
```

`make testacc` runs live acceptance tests and is intentionally excluded from
CI. It requires an explicitly supplied non-production ClearML account through
`CLEARML_ACCESS_KEY`, `CLEARML_SECRET_KEY`, and, when needed,
`CLEARML_API_URL`.

The provider has fixture-based HTTP contract tests and intentionally does not use ClearML credentials in CI. See [the verification checklist](docs/verification.md) for the release-owner live validation procedure and [the release-trust guide](docs/release-trust.md) for signing, approval, provenance, and verification requirements.

## Dependency maintenance

Renovate proposes reviewed dependency updates weekly after a seven-day cooling period. Security alerts may bypass that delay but never auto-merge. Repository administrators must install the Renovate GitHub App for this configuration to run. See [the supply-chain review skill](.github/skills/dependency-supply-chain-review/SKILL.md) for the review workflow.
