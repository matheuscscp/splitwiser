locals {
  start_bot_function_name = "StartBot"
}

resource "google_service_account" "start-bot" {
  account_id   = "start-bot-cloud-function"
  display_name = "Start Bot Cloud Function"
}

resource "google_secret_manager_secret_iam_member" "start-bot-config-secret-accessor" {
  secret_id = google_secret_manager_secret.start-bot-config.id
  member    = "serviceAccount:${google_service_account.start-bot.email}"
  role      = "roles/secretmanager.secretAccessor"
}

resource "google_secret_manager_secret_iam_member" "start-bot-jwt-secret-accessor" {
  secret_id = google_secret_manager_secret.start-bot-jwt-secret.id
  member    = "serviceAccount:${google_service_account.start-bot.email}"
  role      = "roles/secretmanager.secretAccessor"
}

resource "google_pubsub_topic_iam_member" "start-bot-publisher" {
  topic  = google_pubsub_topic.start-bot.name
  member = "serviceAccount:${google_service_account.start-bot.email}"
  role   = "roles/pubsub.publisher"
}

resource "google_cloudfunctions_function_iam_member" "all-users-start-bot-invoker" {
  cloud_function = google_cloudfunctions_function.start-bot.name
  member         = "allUsers"
  role           = "roles/cloudfunctions.invoker"
}

resource "google_cloudfunctions_function" "start-bot" {
  name                         = local.start_bot_function_name
  entry_point                  = local.start_bot_function_name
  description                  = "HTTPS function to trigger the bot via Cloud Pub/Sub"
  runtime                      = "go122"
  trigger_http                 = true
  https_trigger_security_level = "SECURE_ALWAYS"
  docker_registry              = "ARTIFACT_REGISTRY"
  source_archive_bucket        = google_storage_bucket.source-code.name
  source_archive_object        = google_storage_bucket_object.source-code.name
  service_account_email        = google_service_account.start-bot.email
  max_instances                = 1
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

resource "google_secret_manager_secret" "start-bot-config" {
  secret_id = "start-bot-config"
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "start-bot-config" {
  secret = google_secret_manager_secret.start-bot-config.id
  secret_data = yamlencode({
    "password" : data.google_secret_manager_secret_version.start-bot-password.secret_data,
    "projectID" : var.project,
    "topicID" : google_pubsub_topic.start-bot.name,
    "jwtSecretID" : google_secret_manager_secret.start-bot-jwt-secret.id,
  })
}

data "google_secret_manager_secret_version" "start-bot-password" {
  secret = "start-bot-password"
}

resource "google_secret_manager_secret" "start-bot-jwt-secret" {
  secret_id = "start-bot-jwt-secret"
  replication {
    auto {}
  }
  rotation {
    rotation_period    = "7776000s" # 90d
    next_rotation_time = timeadd(timestamp(), "1h")
  }
  topics {
    name = google_pubsub_topic.rotate-secret.id
  }
  labels = {
    "num-bytes" = "32"
  }
  depends_on = [
    google_pubsub_topic_iam_member.secretmanager-agent-rotate-secret-publisher
  ]
}
