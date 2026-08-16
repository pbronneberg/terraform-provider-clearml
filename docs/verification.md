# ClearML v1 release verification

CI verifies typed HTTP contracts, state semantics, generated documentation,
examples, and core live lifecycles. Enterprise tests run only with explicit
credentials and capability gates; skipped Enterprise tests are not evidence of
compatibility.

Before publishing `v1.0.0-rc.1`, record live evidence for:

1. Project create, hierarchy move, update, import, remote disappearance, and
   refusal to delete a non-empty project.
2. Queue create, metadata update and clearing, set-stable tags, import, remote
   disappearance, and refusal to delete queued work.
3. Access-rule create, update, import, drift, and deletion using both a user
   group and service account.
4. Resource-policy create, quota update, import, move preflight, and refusal to
   delete a connected policy or dequeue pending work.
5. Policy-profile connection create, import, disappearance, and refusal to
   disconnect a non-empty generated queue.
6. All five exact-name data sources, including missing, duplicate, and
   malformed responses.

Run examples with Terraform 1.6.6 and the current supported version. Run the
full race, vet, build, generated-doc, vulnerability, waiver, and SBOM checks.
Confirm credentials, bearer tokens, and API response bodies never appear in
logs or diagnostics.

Evidence must identify the ClearML deployment and version, Terraform version,
provider tag, date, executor, capability gates, and any skipped checks. Never
record credentials or production identifiers.

Publish `v1.0.0` only after the RC contract and the
[v0.3.2 migration](guides/v1-migration.md) pass. The tag must be reachable from
`main` and follow the [release trust controls](release-trust.md).
