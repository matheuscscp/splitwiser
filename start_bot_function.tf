locals {
  start_bot_function_name = "StartBot"
}

resource "google_service_account" "start-bot" {
  account_id   = "start-bot-cloud-function"
  display_name = "StartBot Cloud Function"
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
  secret_environment_variables {
    key     = "JWT_SECRET"
    secret  = google_secret_manager_secret.start-bot-jwt-secret.secret_id
    version = "latest"
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
    "password" : data.google_secret_manager_secret_version.start-bot-password.secret_data,
    "projectID" : local.project,
    "topicID" : google_pubsub_topic.start-bot.name,
  })
}

data "google_secret_manager_secret_version" "start-bot-password" {
  secret = "start-bot-password"
}

resource "google_secret_manager_secret" "start-bot-jwt-secret" {
  secret_id = "start-bot-jwt-secret"
  replication {
    automatic = true
  }
  rotation {
    rotation_period    = "7776000s" # 90d
    next_rotation_time = timeadd(timestamp(), "1m")
  }
  topics {
    name = google_pubsub_topic.rotate-jwt-secret.id
  }
  labels = {
    "pubsub-topic-iam-membership-dependency" = google_pubsub_topic_iam_member.service-agent-identity-rotate-jwt-secret-publisher.etag
  }
}
