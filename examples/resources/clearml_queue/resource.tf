resource "clearml_queue" "example" {
  name         = "gpu-production"
  display_name = "Production GPU queue"
  tags         = ["gpu", "production"]

  metadata = {
    owner = {
      type  = "string"
      value = "ml-platform"
    }
    gpu_count = {
      type  = "integer"
      value = "4"
    }
  }
}
