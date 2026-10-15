variable "project_id" {
  description = "The GCP project ID to deploy into (e.g. the real project ID behind the \"knowledge-and-insights\" project, which may differ from its display name)."
  type        = string
  default     = "knowledge-and-insights"
}

variable "region" {
  description = "Region for Cloud Run and Artifact Registry."
  type        = string
  default     = "us-central1"
}

variable "service_name" {
  description = "Name of the Cloud Run service running `sal serve --with-ui`."
  type        = string
  default     = "geoconnex-sal"
}

variable "artifact_registry_repository_id" {
  description = "Artifact Registry repository (Docker format) that holds the built `sal serve` image."
  type        = string
  default     = "geoconnex-sal"
}

variable "sal_builder_service_account_id" {
  description = "Service account Cloud Build runs the harvest + image build + deploy as."
  type        = string
  default     = "geoconnex-sal-builder"
}

variable "cloud_run_service_account_id" {
  description = "Service account the Cloud Run revision runs `sal serve` as."
  type        = string
  default     = "geoconnex-sal-run"
}

variable "cloud_run_cpu" {
  description = "vCPUs allocated to the sal serve container. DuckDB scanning the Iceberg table benefits from more than the Cloud Run default."
  type        = string
  default     = "2"
}

variable "cloud_run_memory" {
  description = "Memory allocated to the sal serve container."
  type        = string
  default     = "2Gi"
}

variable "cloud_run_max_instance_count" {
  description = "Upper bound on concurrent sal serve instances. min_instance_count is fixed at 0 so the service scales to zero when idle."
  type        = number
  default     = 2
}

variable "placeholder_image" {
  description = "Image the Cloud Run service is created with before the first manual build has ever pushed a real sal serve image. Replaced out-of-band by cloudbuild.yaml's deploy step; Terraform never manages the running image after that (see the ignore_changes lifecycle block on google_cloud_run_v2_service.sal)."
  type        = string
  default     = "us-docker.pkg.dev/cloudrun/container/hello"
}
