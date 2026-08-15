# Release trust model

This repository uses independent, verifiable controls for a release. A signed
commit alone only identifies the key that made the commit; it does not prove
that the released binaries were built from reviewed source. The release flow
therefore combines protected source, a narrowly scoped release environment,
Terraform Registry GPG signatures, and GitHub artifact attestations.

## Controls enforced by the repository

- The release workflow accepts `v*` tags only and fails unless the tag commit
  is reachable from `main`.
- The workflow runs in the `release` GitHub environment, which is restricted to
  `v*` tags and does not permit administrator bypass. Its secrets cannot be
  read by arbitrary branch workflows.
- GoReleaser signs the SHA-256 checksum manifest. Terraform Registry verifies
  that signature when installing a provider.
- GitHub generates OIDC-backed provenance for every release artifact listed in
  the GoReleaser checksum manifest, and an SPDX dependency SBOM attestation
  for the release archives. The SPDX file is also attached to the GitHub
  release for easy inspection.
- Actions are pinned to immutable commit SHAs; dependency scanning and SBOM
  submission run separately in the supply-chain workflow.

## Required administrator setup

These settings require trusted people and secrets, so they cannot be made
secure by a repository file alone.

1. Protect `main`: require pull requests, require the test and supply-chain
   checks, require at least one independent review, and disallow force pushes
   and branch deletion. Do not grant everyday contributors bypass permission.
2. In the `release` environment, add at least one independent release
   approver and enable **Prevent self-review**. Only people who do not control
   the release key should approve deployments. The current environment is
   already restricted to `v*` tags; approval is intentionally not enabled
   until an independent reviewer is named.
3. Create a dedicated RSA OpenPGP release key: keep the primary key offline;
   use an expiry-bound signing subkey in GitHub Actions; publish the public key
   fingerprint and rotate or revoke it through a documented incident process.
   Never reuse a maintainer's normal commit-signing key for automation.
4. Add the armored signing subkey and passphrase as environment secrets named
   `RELEASE_GPG_PRIVATE_KEY` and `RELEASE_GPG_PASSPHRASE`. Upload the matching
   public key to the Terraform Registry provider settings before publishing.
5. Restrict who can create version tags and releases. Ideally a separate
   release-manager team holds this permission; the workflow only accepts tags
   whose target is already in protected `main`.

GitHub does not make a single administrator independent from themself. If this
repository has only one trusted owner, its strongest honest posture is a
tag-gated environment with short-lived, dedicated credentials plus public
provenance—not an approval rule whose sole reviewer can approve their own
release.

## Verify a published release

Download the archive, `SHA256SUMS`, `SHA256SUMS.sig`, provider manifest, and
the published release public key. Import the public key and verify the
checksum signature before use:

```sh
gpg --import terraform-provider-clearml-release-public.asc
gpg --verify terraform-provider-clearml_<version>_SHA256SUMS.sig \
  terraform-provider-clearml_<version>_SHA256SUMS
sha256sum --check terraform-provider-clearml_<version>_SHA256SUMS
```

For the GitHub build identity, verify a released archive and inspect the
workflow, repository, and commit recorded in its attestation:

```sh
gh attestation verify terraform-provider-clearml_<version>_<os>_<arch>.zip \
  --repo pbronneberg/terraform-provider-clearml
```

The resulting provenance should identify this repository, the `Release`
workflow, the version-tag event, and the reviewed commit on `main`. Independently
checking the GPG signature and the OIDC attestation protects against different
failure modes: key misuse and an unexpected build identity, respectively.
