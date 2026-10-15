# Runs the harvest + image build + deploy (cloudbuild.yaml), invoked
# manually with `gcloud builds submit --service-account=...`. Scoped to only
# what that pipeline touches, rather than Cloud Build's broad legacy
# default service account.
resource "google_service_account" "sal_builder" {
  project      = var.project_id
  account_id   = var.sal_builder_service_account_id
  display_name = "sal build + deploy (geoconnex knowledge-and-insights)"
}

# A custom Cloud Build service account must be able to write its own build
# logs; the legacy default service account gets this implicitly, a custom
# one does not.
resource "google_project_iam_member" "sal_builder_log_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = google_service_account.sal_builder.member
}

resource "google_artifact_registry_repository_iam_member" "sal_builder_push" {
  project    = var.project_id
  location   = google_artifact_registry_repository.sal.location
  repository = google_artifact_registry_repository.sal.repository_id
  role       = "roles/artifactregistry.writer"
  member     = google_service_account.sal_builder.member
}

resource "google_cloud_run_v2_service_iam_member" "sal_builder_deploy" {
  project  = var.project_id
  location = google_cloud_run_v2_service.sal.location
  name     = google_cloud_run_v2_service.sal.name
  role     = "roles/run.developer"
  member   = google_service_account.sal_builder.member
}

# Deploying a revision that runs as sal_run requires the deployer to be able
# to act as it.
resource "google_service_account_iam_member" "sal_builder_act_as_runtime" {
  service_account_id = google_service_account.sal_run.name
  role               = "roles/iam.serviceAccountUser"
  member             = google_service_account.sal_builder.member
}

# Identity the Cloud Run revision itself runs as. It needs nothing beyond
# the defaults: the harvested data is baked into the image (see
# demo/Dockerfile), so sal serve never makes an outbound request at request
# time, and Cloud Run captures container stdout/stderr at the platform
# level without requiring IAM roles on the runtime identity.
resource "google_service_account" "sal_run" {
  project      = var.project_id
  account_id   = var.cloud_run_service_account_id
  display_name = "sal serve runtime (geoconnex knowledge-and-insights)"
}
