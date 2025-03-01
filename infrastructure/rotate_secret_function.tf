locals {
  rotate_secret_function_name = "RotateSecret"
}

resource "google_service_account" "rotate-secret" {
  account_id   = "rotate-secret-cloud-function"
  display_name = "Rotate Secret Cloud Function"
}

resource "google_project_iam_member" "rotate-secret-viewer" {
  project = var.project
  member  = "serviceAccount:${google_service_account.rotate-secret.email}"
  role    = "roles/secretmanager.viewer"
}

resource "google_project_iam_member" "rotate-secret-version-manager" {
  project = var.project
  member  = "serviceAccount:${google_service_account.rotate-secret.email}"
  role    = "roles/secretmanager.secretVersionManager"
}

resource "google_cloudfunctions_function" "rotate-secret" {
  name                  = local.rotate_secret_function_name
  entry_point           = local.rotate_secret_function_name
  description           = "Background function to rotate secrets in the project"
  runtime               = "go122"
  docker_registry       = "ARTIFACT_REGISTRY"
  source_archive_bucket = google_storage_bucket.source-code.name
  source_archive_object = google_storage_bucket_object.source-code.name
  service_account_email = google_service_account.rotate-secret.email
  max_instances         = 1
  event_trigger {
    event_type = "google.pubsub.topic.publish"
    resource   = google_pubsub_topic.rotate-secret.id
  }
}
