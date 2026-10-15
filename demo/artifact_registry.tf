# Holds the image cloudbuild.yaml bakes the built SAL project (including its
# harvested .sal/data) into, which Cloud Run then serves.
resource "google_artifact_registry_repository" "sal" {
  project       = var.project_id
  location      = var.region
  repository_id = var.artifact_registry_repository_id
  format        = "DOCKER"
  description   = "sal serve images for the geoconnex knowledge-and-insights demo"

  depends_on = [google_project_service.required]
}
