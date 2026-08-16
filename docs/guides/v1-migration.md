# Migrate from v0.3.2 to v1

Version 1 deliberately removes the compatibility code that would otherwise
make the provider harder to maintain. Review a saved plan before applying the
upgrade.

## 1. Back up queue IDs and state

Before changing the provider version, record every managed queue ID and create
a backend-appropriate state backup. Queue IDs are needed only if Terraform
cannot decode the old list-based `tags` state.

```sh
terraform state show clearml_queue.example
terraform state pull > terraform-state-before-clearml-v1.json
```

Protect the state backup as a secret because it may contain unrelated
sensitive values.

## 2. Rename provider configuration

Replace `api_url` with `api_host`:

```hcl
provider "clearml" {
  api_host   = "https://api.clear.ml"
  access_key = var.clearml_access_key
  secret_key = var.clearml_secret_key
}
```

Replace the v0.3.2 environment variables:

| v0.3.2 | v1 |
|---|---|
| `CLEARML_API_URL` | `CLEARML_API_HOST` |
| `CLEARML_ACCESS_KEY` | `CLEARML_API_ACCESS_KEY` |
| `CLEARML_SECRET_KEY` | `CLEARML_API_SECRET_KEY` |

Explicit HCL remains higher precedence than environment variables. The hosted
API default remains `https://api.clear.ml`.

## 3. Update and inspect

Update the provider constraint, initialize, and refresh without applying:

```sh
terraform init -upgrade
terraform plan -refresh-only
terraform plan
```

Queue tags are now sets. Existing HCL such as `tags = ["gpu", "production"]`
remains valid, but Terraform removes duplicates and ignores ordering.

## 4. Re-import state only when required

If Terraform reports that it cannot decode a v0.3.2 queue because `tags`
changed from a list to a set, remove only that resource address from state and
re-import its previously recorded queue ID:

```sh
terraform state rm clearml_queue.example
terraform import clearml_queue.example queue-id
terraform plan
```

`terraform state rm` does not delete the ClearML queue. Do not remove or
recreate the remote queue, and do not remove unrelated state.

After import, review any tag normalization and the newly read optional
`display_name` and `metadata` values. Omitted optional values preserve their
remote values; explicitly configured empty strings, sets, or maps clear them.

## Release sequence

Validate the migration with `v1.0.0-rc.1` before adopting `v1.0.0`. There are
no alpha or beta release stages for this contract.
