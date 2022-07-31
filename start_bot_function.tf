locals {
  start_bot_function_name = "StartBot"
}

resource "google_service_account" "start-bot" {
  account_id   = "start-bot-cloud-function"
  display_name = "StartBot Cloud Function"
}

resource "google_project_iam_member" "start-bot-secret-accessor" {
  project = local.project
  member  = "serviceAccount:${google_service_account.start-bot.email}"
  role    = "roles/secretmanager.secretAccessor"
}

resource "google_cloudfunctions_function" "start-bot" {
  name                         = local.start_bot_function_name
  entry_point                  = local.start_bot_function_name
  description                  = "HTTPS function to trigger the bot via Cloud Pub/Sub"
  runtime                      = "go116"
  trigger_http                 = true
  https_trigger_security_level = "SECURE_ALWAYS"
  docker_registry              = "ARTIFACT_REGISTRY"
  source_archive_bucket        = google_storage_bucket.functions-source-code.name
  source_archive_object        = google_storage_bucket_object.functions-source-code.name
  service_account_email        = google_service_account.start-bot.email
  secret_environment_variables {
    key     = "TOKEN"
    secret  = "start-bot-token"
    version = "latest"
  }
  environment_variables = {
    PROJECT_ID = local.project
    TOPIC_ID   = google_pubsub_topic.start-bot.name
  }
}

resource "google_cloudfunctions_function_iam_member" "invoker" {
  cloud_function = google_cloudfunctions_function.start-bot.name
  role           = "roles/cloudfunctions.invoker"
  member         = "allUsers"
}
