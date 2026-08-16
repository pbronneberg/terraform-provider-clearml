data "clearml_user_group" "ml_engineers" {
  name = "ML Engineers"
}

data "clearml_resource_profile" "gpu_small" {
  name = "gpu-small"
}

resource "clearml_resource_policy" "interactive_gpu" {
  name          = "interactive-gpu"
  reservation   = 1
  limit         = 4
  user_group_id = data.clearml_user_group.ml_engineers.id
}

# This connection creates the queue. Do not declare the same queue with
# clearml_queue. The vendor administrator must enable the policy manager first.
resource "clearml_resource_policy_profile_connection" "gpu_small" {
  policy_id    = clearml_resource_policy.interactive_gpu.id
  profile_id   = data.clearml_resource_profile.gpu_small.id
  queue_name   = "interactive-gpu-small"
  display_name = "Interactive GPU (small)"
}
