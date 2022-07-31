locals {
  rotate_jwt_secret_function_name = "RotateJWTSecret"
}

resource "google_service_account" "rotate-jwt-secret" {
  account_id   = "rotate-jwt-secret-cf"
  display_name = "Rotate JWT Secret Cloud Function"
}

resource "google_secret_manager_secret_iam_member" "rotate-jwt-secret-version-manager" {
  secret_id = google_secret_manager_secret.start-bot-jwt-secret.id
  member    = "serviceAccount:${google_service_account.rotate-jwt-secret.email}"
  role      = "roles/secretmanager.secretVersionManager"
}

resource "google_pubsub_topic_iam_member" "service-agent-identity-rotate-jwt-secret-publisher" {
  topic  = google_pubsub_topic.rotate-jwt-secret.name
  member = "serviceAccount:service-281540018258@gcp-sa-secretmanager.iam.gserviceaccount.com"
  role   = "roles/pubsub.publisher"
}

resource "google_cloudfunctions_function" "rotate-jwt-secret" {
  name                  = local.rotate_jwt_secret_function_name
  entry_point           = local.rotate_jwt_secret_function_name
  description           = "Background function to rotate the JWT secret for the StartBot HTTPS function"
  runtime               = "go116"
  docker_registry       = "ARTIFACT_REGISTRY"
  source_archive_bucket = google_storage_bucket.functions-source-code.name
  source_archive_object = google_storage_bucket_object.functions-source-code.name
  service_account_email = google_service_account.rotate-jwt-secret.email
  event_trigger {
    event_type = "google.pubsub.topic.publish"
    resource   = google_pubsub_topic.rotate-jwt-secret.id
  }
  environment_variables = {
    PARENT = google_secret_manager_secret.start-bot-jwt-secret.id
  }
}
