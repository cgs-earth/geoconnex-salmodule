# Services this stack needs. sal build itself also reaches out to GitHub,
# the Geoconnex STAC catalog, and schema.org at build time, which Cloud
# Build's default networking already allows. No storage.googleapis.com
# here: the harvested data is baked into the serve image (see
# demo/Dockerfile), not uploaded to a bucket, since nothing reads one back.
locals {
  required_apis = [
    "run.googleapis.com",
    "cloudbuild.googleapis.com",
    "artifactregistry.googleapis.com",
    "iam.googleapis.com",
  ]
}

resource "google_project_service" "required" {
  for_each = toset(local.required_apis)

  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}
