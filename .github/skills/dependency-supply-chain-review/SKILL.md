---
name: dependency-supply-chain-review
description: Review or change Go dependencies, Renovate policy, GitHub Actions, SBOMs, vulnerability findings, or release provenance. Do not use for ordinary provider feature work.
---

# Dependency and Supply-Chain Review

1. Inspect `go.mod`, `go.sum`, imports, tools, workflows, Renovate rules, and
   existing scan output before proposing a dependency or policy change.
2. For every direct Go module, use `go mod why -m` and source imports to record
   whether it is runtime, test, documentation, or build tooling. Remove a
   dependency when no supported use remains.
3. Check upstream maintenance, published advisories, license implications, and
   compatibility with the repository's supported Go and Terraform versions.
4. Keep dependency updates reviewed and subject to the configured release-age
   delay. Security updates may bypass the delay but still require tests and
   normal review.
5. Run the repository security checks and distinguish scanner output from
   actionable findings. Any exception must be recorded in
   `.security/vulnerability-waivers.yaml` with an owner, rationale,
   compensating control, and future expiry.
6. For workflow changes, pin every action to a full commit SHA, preserve least
   privilege permissions, and verify SBOM and attestation outputs.
7. Report the modules/actions reviewed, evidence, changes, verification, and
   remaining risks.
