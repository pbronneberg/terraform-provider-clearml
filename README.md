# ClearML Terraform Provider

Terraform provider for managing ClearML projects and queues, plus focused
ClearML Enterprise access and resource-policy configuration.

## Requirements

- Terraform `>= 1.6`
- Go `1.26.6` for local development
- ClearML hosted (`https://api.clear.ml`) or a compatible self-hosted API for
  projects and queues
- ClearML Enterprise for identity lookups, access rules, and resource policies

The provider uses Terraform Plugin Framework and protocol v6.

## Configuration

```hcl
terraform {
  required_providers {
    clearml = {
      source  = "pbronneberg/clearml"
      version = "~> 1.0"
    }
  }
}

provider "clearml" {
  # api_host   = "https://api.clear.ml"
  # access_key = var.clearml_access_key
  # secret_key = var.clearml_secret_key
}
```

Explicit provider arguments take precedence over the standard ClearML
environment variables:

- `api_host` → `CLEARML_API_HOST` → `https://api.clear.ml`
- `access_key` → `CLEARML_API_ACCESS_KEY`
- `secret_key` → `CLEARML_API_SECRET_KEY`

## Provider scope

The provider manages hierarchical projects, queues, access rules, resource
policies, and policy-profile connections. It looks up projects, queues, service
accounts, user groups, and vendor-provisioned resource profiles by exact name.

ClearML/vendor administrators remain responsible for the resource policy
manager and profile definitions. Identity providers remain responsible for
people and group membership. The provider does not manage storage, mounts,
vaults, human users, service-account creation, resource pools, or the global
policy manager.

Policy-profile connections create execution queues. Do not manage the same
queue separately with `clearml_queue`.

See the [registry documentation](docs/index.md) for schemas and examples. Users
upgrading from `v0.3.2` must follow the [v1 migration guide](docs/guides/v1-migration.md).

## Development

Open the repository in its Dev Container, or run the same Make targets from a
host with Docker installed:

```sh
make test
make lint
make build
make generate-check
make validate-examples
make security
```

The default devcontainer uses Terraform 1.15.8. Use
`TERRAFORM_VERSION=1.6.6` for the supported minimum.

`make testacc` runs live core acceptance tests with
`CLEARML_API_ACCESS_KEY` and `CLEARML_API_SECRET_KEY`; `CLEARML_API_HOST` is
optional. Enterprise tests run only when `CLEARML_ENTERPRISE_ACC=1` and the
required vendor-provisioned IDs are supplied. CI never exposes credentials to
fork pull requests.

Acceptance objects use a unique `tfacc-clearml-` prefix. Queue cleanup is
limited to exact CI-owned names older than 24 hours and never force-deletes a
queue.

See the [verification checklist](docs/verification.md) and
[release-trust guide](docs/release-trust.md) for release gates, signing,
provenance, and SBOM verification.

## Dependency maintenance

Renovate proposes reviewed dependency updates after a seven-day cooling
period. Security alerts may bypass that delay but never auto-merge. See the
[supply-chain review skill](.github/skills/dependency-supply-chain-review/SKILL.md).
