---
applyTo: "**"
---

# Repository-Wide AI Instructions

## Source of truth

- Keep provider behavior in Go code, executable tests, generated docs, and
  release configuration. AI instructions and agents only route work to those
  sources; they must not become competing documentation.
- Keep global rules here, specialist lenses under `.github/agents/`, and
  repeatable workflows under `.github/skills/`.
- State assumptions explicitly and do not redesign public provider behavior
  without a documented compatibility and release decision.

## Working rules

- Keep changes small, scoped, and independently testable. Do not include
  unrelated working-tree changes in a commit.
- Preserve Terraform configuration, import, and state compatibility unless a
  planned major release explicitly says otherwise.
- Treat the ClearML REST API as an external boundary: use typed contracts,
  validate malformed responses, avoid logging credentials, and add tests for
  any endpoint behavior changed.
- Treat dependencies and GitHub Actions as supply-chain inputs. Update them
  through the reviewed Renovate policy; review advisories, licenses, and
  necessity before adding a dependency.
- Pin GitHub Actions to immutable commit SHAs. Record an owner, rationale,
  compensating control, and expiry for every vulnerability waiver.

## Verification

- Run focused unit/contract tests for code changes, then the relevant full
  build, test, generation, and security checks before release work.
- Report commands executed, their observable assertions, and unproven claims.
- Regenerate documentation rather than manually editing generated pages.
