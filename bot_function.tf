locals {
  bot_function_name = "Bot"
}

resource "google_service_account" "bot" {
  account_id   = "bot-cloud-function"
  display_name = "Bot Cloud Function"
}

resource "google_project_iam_member" "bot-secret-accessor" {
  project = local.project
  member  = "serviceAccount:${google_service_account.bot.email}"
  role    = "roles/secretmanager.secretAccessor"
}

resource "google_project_iam_member" "bot-object-admin" {
  project = local.project
  member  = "serviceAccount:${google_service_account.bot.email}"
  role    = "roles/storage.objectAdmin"
}

resource "google_storage_bucket" "bot-checkpoint" {
  name     = "splitwiser-checkpoint"
  location = local.storage_location
}

resource "google_cloudfunctions_function" "bot" {
  name                  = local.bot_function_name
  entry_point           = local.bot_function_name
  description           = "The bot background function"
  runtime               = "go116"
  timeout               = 540
  docker_registry       = "ARTIFACT_REGISTRY"
  source_archive_bucket = google_storage_bucket.functions-source-code.name
  source_archive_object = google_storage_bucket_object.functions-source-code.name
  service_account_email = google_service_account.bot.email
  event_trigger {
    event_type = "google.pubsub.topic.publish"
    resource   = google_pubsub_topic.start-bot.id
  }
  secret_volumes {
    mount_path = local.config_path
    secret     = "bot-config"
    versions {
      path    = local.config_file
      version = "latest"
    }
  }
  environment_variables = {
    CONF_FILE = local.config_file_path
  }
}

# resource "google_secret_manager_secret" "bot-config" {
#   secret_id = "bot-config"
#   replication {}
# }

# resource "google_secret_manager_secret_version" "bot-config" {
#   secret      = google_secret_manager_secret.bot-config.id
#   secret_data = yamlencode({
#     "telegram": {
#     },
#     "checkpointBucket": google_storage_bucket.bot-checkpoint.name,
#   })
# }
