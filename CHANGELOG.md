# Changelog

## 1.0.0 (Unreleased)

`v1.0.0-rc.1` is the single contract-validation release. `v1.0.0` follows
after core and explicitly gated Enterprise acceptance, migration, generated
documentation, security, signing, provenance, and SBOM checks pass.

### Breaking changes

- Replace provider argument `api_url` with `api_host`.
- Support only `CLEARML_API_HOST`, `CLEARML_API_ACCESS_KEY`, and
  `CLEARML_API_SECRET_KEY`; remove the v0.3.2 environment variable names.
- Normalize queue `tags` as an unordered set instead of a list.
- Establish the focused 5-resource/5-data-source v1 contract. See the
  [migration guide](docs/guides/v1-migration.md).

### Features

- Add hierarchical projects and exact-name project and queue lookups.
- Add queue display names and typed metadata.
- Add exact-name Enterprise service-account, user-group, and resource-profile
  lookups.
- Add Enterprise access rules for groups and service accounts.
- Add Enterprise resource policies and profile connections that create
  execution queues.

### Safety and verification

- Refuse force deletion and destructive operations involving non-empty
  projects or queues, connected policies, pending policy work, or non-empty
  policy queues.
- Redact ClearML response bodies from API errors and reject malformed
  successful responses.
- Validate access subjects, permission values, and policy quotas at plan time.
- Validate examples with Terraform 1.6 and the current supported release; gate
  Enterprise acceptance explicitly.

## 0.3.2

Published queue-only baseline for the breaking v1 migration.
