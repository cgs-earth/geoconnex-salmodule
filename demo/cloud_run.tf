# Terraform only creates the service shell; cloudbuild.yaml's `gcloud run
# deploy` step is what actually updates the running image after each manual
# harvest. The lifecycle block below keeps a later `terraform apply` from
# reverting that deploy back to the placeholder image.
resource "google_cloud_run_v2_service" "sal" {
  project  = var.project_id
  name     = var.service_name
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    service_account = google_service_account.sal_run.email

    scaling {
      min_instance_count = 0
      max_instance_count = var.cloud_run_max_instance_count
    }

    containers {
      image = var.placeholder_image

      ports {
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = var.cloud_run_cpu
          memory = var.cloud_run_memory
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
      client,
      client_version,
    ]
  }

  depends_on = [google_project_service.required]
}

# "Public HTTPS" access: Cloud Run always terminates TLS on its *.run.app
# URL, so making the service reachable is purely an IAM question. Add a
# google_cloud_run_v2_service_iam_member with a custom domain mapping if
# you want this behind your own hostname instead.
resource "google_cloud_run_v2_service_iam_member" "public" {
  project  = var.project_id
  location = google_cloud_run_v2_service.sal.location
  name     = google_cloud_run_v2_service.sal.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
