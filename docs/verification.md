# ClearML v1 release verification

CI proves the provider against versioned HTTP fixtures, including the queue response surface represented by ClearML Server 3.28.8. On trusted repository pull requests, `main`, and scheduled runs, it also exercises the hosted ClearML queue lifecycle with both supported Terraform versions. Fork pull requests never receive ClearML credentials. Hosted CI validates the queue lifecycle only; it does not claim compatibility beyond the covered API surface.

Before cutting a release, the release owner must perform this checklist against both `https://api.clear.ml` and the selected current self-hosted ClearML Server:

1. Configure the provider with a non-production account having queue access.
2. Apply a `clearml_queue` with a name and two tags; verify the queue in the ClearML UI/API.
3. Change the name and tags, apply again, and verify the updated queue.
4. Import the queue by ID and confirm `terraform plan` is empty.
5. Remove the resource and confirm it is deleted; record any API response or schema discrepancy in the release issue before publishing.

The evidence must name the ClearML endpoint, Server version (for self-hosted), Terraform version, provider tag, date, and executor. Never commit credentials, access tokens, or production queue identifiers.

The tag must be created from `main`, and the release process must follow the controls in [release trust](release-trust.md). Record the issue or pull request that approved the release with the live-verification evidence.
