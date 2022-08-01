locals {
  rotate_secret_function_name = "RotateSecret"
}

resource "google_service_account" "rotate-secret" {
  account_id   = "rotate-secret-cloud-function"
  display_name = "Rotate Secret Cloud Function"
}

resource "google_project_iam_member" "rotate-secret-version-manager" {
  project = local.project
  member  = "serviceAccount:${google_service_account.rotate-secret.email}"
  role    = "roles/secretmanager.secretVersionManager"
}

resource "google_pubsub_topic_iam_member" "service-agent-rotate-secret-publisher" {
  topic  = google_pubsub_topic.rotate-secret.name
  member = "serviceAccount:service-281540018258@gcp-sa-secretmanager.iam.gserviceaccount.com"
  role   = "roles/pubsub.publisher"
}

resource "google_cloudfunctions_function" "rotate-secret" {
  name                  = local.rotate_secret_function_name
  entry_point           = local.rotate_secret_function_name
  description           = "Background function to rotate secrets in the project"
  runtime               = "go116"
  docker_registry       = "ARTIFACT_REGISTRY"
  source_archive_bucket = google_storage_bucket.functions-source-code.name
  source_archive_object = google_storage_bucket_object.functions-source-code.name
  service_account_email = google_service_account.rotate-secret.email
  event_trigger {
    event_type = "google.pubsub.topic.publish"
    resource   = google_pubsub_topic.rotate-secret.id
  }
}
