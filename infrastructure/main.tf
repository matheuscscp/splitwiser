variable "region" {
  type = string
}

variable "project" {
  type = string
}

variable "project_number" {
  type = string
}

locals {
  config_path      = "/etc/secrets/config"
  config_file      = "/latest.yml"
  config_file_path = format("%s%s", local.config_path, local.config_file)
  storage_location = upper(var.region)
}

resource "google_storage_bucket" "source-code" {
  name     = "splitwiser-source-code-${var.project_number}"
  location = local.storage_location
}

resource "google_storage_bucket_object" "source-code" {
  name   = filesha256("./source.zip")
  bucket = google_storage_bucket.source-code.name
  source = "./source.zip"
}

resource "google_pubsub_topic" "start-bot" {
  name = "start-bot"
}

resource "google_pubsub_topic" "rotate-secret" {
  name = "rotate-secret"
}

resource "google_project_service_identity" "secretmanager-agent" {
  provider = google-beta
  project  = var.project
  service  = "secretmanager.googleapis.com"
}

resource "google_pubsub_topic_iam_member" "secretmanager-agent-rotate-secret-publisher" {
  topic  = google_pubsub_topic.rotate-secret.name
  member = "serviceAccount:${google_project_service_identity.secretmanager-agent.email}"
  role   = "roles/pubsub.publisher"
}
