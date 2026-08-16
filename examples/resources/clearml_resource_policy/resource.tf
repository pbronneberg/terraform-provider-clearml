data "clearml_user_group" "ml_engineers" {
  name = "ML Engineers"
}

# The ClearML vendor administrator must enable the resource policy manager.
resource "clearml_resource_policy" "interactive_gpu" {
  name          = "interactive-gpu"
  description   = "Reserved interactive GPU capacity"
  reservation   = 1
  limit         = 4
  user_group_id = data.clearml_user_group.ml_engineers.id
}
