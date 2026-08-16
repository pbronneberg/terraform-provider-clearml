data "clearml_user_group" "ml_engineers" {
  name = "ML Engineers"
}

data "clearml_service_account" "deployment" {
  name = "deployment-agent"
}

resource "clearml_access_rule" "project_operators" {
  name                = "production-project-operators"
  description         = "Allow engineers and deployment automation to operate the production project"
  entity_type         = "project"
  entity_id           = "project-id"
  permission          = "read_write"
  group_ids           = [data.clearml_user_group.ml_engineers.id]
  service_account_ids = [data.clearml_service_account.deployment.id]
}
