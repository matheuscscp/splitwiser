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

resource "google_project_iam_member" "start-bot-pubsub-publisher" {
  project = local.project
  member  = "serviceAccount:${google_service_account.start-bot.email}"
  role    = "roles/pubsub.publisher"
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
  secret_volumes {
    mount_path = local.config_path
    secret     = google_secret_manager_secret.start-bot-config.secret_id
    versions {
      path    = local.config_file
      version = "latest"
    }
  }
  environment_variables = {
    CONF_FILE = local.config_file_path
  }
}

resource "google_cloudfunctions_function_iam_member" "invoker" {
  cloud_function = google_cloudfunctions_function.start-bot.name
  role           = "roles/cloudfunctions.invoker"
  member         = "allUsers"
}

resource "google_secret_manager_secret" "start-bot-config" {
  secret_id = "start-bot-config"
  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_version" "start-bot-config" {
  secret = google_secret_manager_secret.start-bot-config.id
  secret_data = yamlencode({
    "token" : data.google_secret_manager_secret_version.start-bot-token.secret_data,
    "projectID" : local.project,
    "topicID" : google_pubsub_topic.start-bot.name,
  })
}

data "google_secret_manager_secret_version" "start-bot-token" {
  secret = "start-bot-token"
}
