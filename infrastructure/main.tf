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
  config_path                       = "/etc/secrets/config"
  config_file                       = "/latest.yml"
  config_file_path                  = format("%s%s", local.config_path, local.config_file)
  storage_location                  = upper(var.region)
  secretmanager_agent_iam_grant_tag = md5(google_pubsub_topic_iam_member.secretmanager-agent-rotate-secret-publisher.etag)
}

resource "google_storage_bucket" "source-code" {
  name     = "splitwiser-source-code-${var.project_number}"
  location = local.storage_location
}

data "archive_file" "source-code" {
  type        = "zip"
  source_dir  = "./"
  output_path = "./source.zip"
}

resource "google_storage_bucket_object" "source-code" {
  name   = data.archive_file.source-code.output_sha
  bucket = google_storage_bucket.source-code.name
  source = data.archive_file.source-code.output_path
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
