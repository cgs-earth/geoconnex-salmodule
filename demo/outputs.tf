output "cloud_run_url" {
  description = "Public HTTPS URL of the sal serve web UI, once at least one build has run."
  value       = google_cloud_run_v2_service.sal.uri
}

output "artifact_registry_repository" {
  description = "Docker repository holding the sal serve images; use its path as the IMAGE in gcloud builds submit substitutions."
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.sal.repository_id}"
}

output "ar_image" {
  description = "Full image path (no tag) cloudbuild.yaml pushes to and deploys from: _AR_IMAGE."
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.sal.repository_id}/${var.service_name}"
}

output "project_id" {
  value = var.project_id
}

output "region" {
  value = var.region
}

output "service_name" {
  value = var.service_name
}

output "sal_builder_service_account_email" {
  description = "Pass this to gcloud builds submit --service-account=projects/<project_id>/serviceAccounts/<this email>."
  value       = google_service_account.sal_builder.email
}

output "build_command" {
  description = "Run this manually, from the root of the geoconnex-salmodule checkout, whenever you want to re-harvest and redeploy (see demo/README.md)."
  value = join(" ", [
    "gcloud builds submit . --config=demo/cloudbuild.yaml",
    "--service-account=projects/${var.project_id}/serviceAccounts/${google_service_account.sal_builder.email}",
    "--substitutions=_SAL_PROJECT_REPO=<your-existing-sal-project-repo-url>,_AR_IMAGE=${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.sal.repository_id}/${var.service_name},_SERVICE_NAME=${var.service_name},_REGION=${var.region}",
  ])
}
