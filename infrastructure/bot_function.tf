locals {
  bot_function_name = "Bot"
}

resource "google_service_account" "bot" {
  account_id   = "bot-cloud-function"
  display_name = "Bot Cloud Function"
}

resource "google_secret_manager_secret_iam_member" "bot-config-secret-accessor" {
  secret_id = google_secret_manager_secret.bot-config.id
  member    = "serviceAccount:${google_service_account.bot.email}"
  role      = "roles/secretmanager.secretAccessor"
}

resource "google_storage_bucket_iam_member" "bot-checkpoint-bucket-reader" {
  bucket = google_storage_bucket.bot-checkpoint.name
  member = "serviceAccount:${google_service_account.bot.email}"
  role   = "roles/storage.legacyBucketReader"
}

resource "google_storage_bucket_iam_member" "bot-checkpoint-object-admin" {
  bucket = google_storage_bucket.bot-checkpoint.name
  member = "serviceAccount:${google_service_account.bot.email}"
  role   = "roles/storage.objectAdmin"
}

resource "google_storage_bucket" "bot-checkpoint" {
  name     = "splitwiser-checkpoint-${var.project_number}"
  location = local.storage_location
}

resource "google_cloudfunctions_function" "bot" {
  name                  = local.bot_function_name
  entry_point           = local.bot_function_name
  description           = "The bot background function"
  runtime               = "go122"
  timeout               = 540
  docker_registry       = "ARTIFACT_REGISTRY"
  source_archive_bucket = google_storage_bucket.source-code.name
  source_archive_object = google_storage_bucket_object.source-code.name
  service_account_email = google_service_account.bot.email
  max_instances         = 1
  event_trigger {
    event_type = "google.pubsub.topic.publish"
    resource   = google_pubsub_topic.start-bot.id
  }
  secret_volumes {
    mount_path = local.config_path
    secret     = google_secret_manager_secret.bot-config.secret_id
    versions {
      path    = local.config_file
      version = "latest"
    }
  }
  environment_variables = {
    CONF_FILE = local.config_file_path
  }
}

resource "google_secret_manager_secret" "bot-config" {
  secret_id = "bot-config"
  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_version" "bot-config" {
  secret = google_secret_manager_secret.bot-config.id
  secret_data = yamlencode({
    "openai" : {
      "token" : data.google_secret_manager_secret_version.bot-openai-token.secret_data,
    },
    "telegram" : {
      "token" : data.google_secret_manager_secret_version.bot-telegram-token.secret_data,
      "chatID" : tonumber(data.google_secret_manager_secret_version.bot-telegram-chat-id.secret_data),
    },
    "splitwise" : {
      "token" : data.google_secret_manager_secret_version.bot-splitwise-token.secret_data,
      "groupID" : tonumber(data.google_secret_manager_secret_version.bot-splitwise-group-id.secret_data),
      "anaID" : tonumber(data.google_secret_manager_secret_version.bot-splitwise-ana-id.secret_data),
      "matheusID" : tonumber(data.google_secret_manager_secret_version.bot-splitwise-matheus-id.secret_data),
    },
    "checkpointBucket" : google_storage_bucket.bot-checkpoint.name,
  })
}

data "google_secret_manager_secret_version" "bot-telegram-token" {
  secret = "bot-telegram-token"
}

data "google_secret_manager_secret_version" "bot-telegram-chat-id" {
  secret = "bot-telegram-chat-id"
}

data "google_secret_manager_secret_version" "bot-splitwise-token" {
  secret = "bot-splitwise-token"
}

data "google_secret_manager_secret_version" "bot-splitwise-group-id" {
  secret = "bot-splitwise-group-id"
}

data "google_secret_manager_secret_version" "bot-splitwise-ana-id" {
  secret = "bot-splitwise-ana-id"
}

data "google_secret_manager_secret_version" "bot-splitwise-matheus-id" {
  secret = "bot-splitwise-matheus-id"
}

data "google_secret_manager_secret_version" "bot-openai-token" {
  secret = "bot-openai-token"
}
