resource "clearml_project" "production" {
  name                       = "platform/production"
  description                = "Production machine-learning workloads"
  tags                       = ["managed-by-terraform", "production"]
  default_output_destination = "s3://clearml-output/production"
}
