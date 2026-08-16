#!/usr/bin/env bash
set -euo pipefail

repository_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
validation_dir="$(mktemp -d -t clearml-example-validation.XXXXXX)"
trap 'find "$validation_dir" -depth -mindepth 1 -delete; rmdir "$validation_dir"' EXIT

mkdir -p "$validation_dir/provider" "$validation_dir/configuration"
go build -o "$validation_dir/provider/terraform-provider-clearml" "$repository_dir"

printf '%s\n' \
  'provider_installation {' \
  '  dev_overrides {' \
  "    \"registry.terraform.io/pbronneberg/clearml\" = \"$validation_dir/provider\"" \
  '  }' \
  '}' >"$validation_dir/terraform.rc"

while IFS= read -r example_file; do
  find "$validation_dir/configuration" -depth -mindepth 1 -delete
  cp "$example_file" "$validation_dir/configuration/example.tf"
  printf '%s\n' \
    'terraform {' \
    '  required_providers {' \
    '    clearml = { source = "pbronneberg/clearml" }' \
    '  }' \
    '}' >"$validation_dir/configuration/provider-source.tf"
  printf 'Validating %s\n' "${example_file#"$repository_dir/"}"
  TF_CLI_CONFIG_FILE="$validation_dir/terraform.rc" terraform -chdir="$validation_dir/configuration" validate -no-color
done < <(find "$repository_dir/examples" -type f -name '*.tf' -print | sort)
