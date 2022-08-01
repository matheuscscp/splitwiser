terraform {
  required_version = "1.2.6"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.30.0"
    }
  }

  cloud {
    organization = "matheuscscp"
    workspaces {
      name = "splitwiser"
    }
  }
}

provider "google" {
  project = local.project
  region  = local.region
}

locals {
  project          = "splitwiser-356519"
  region           = "europe-west1" # Low CO₂
  config_path      = "/etc/secrets/config"
  config_file      = "/latest.yml"
  config_file_path = format("%s%s", local.config_path, local.config_file)
  storage_location = upper(local.region)
  project_number   = data.google_project.splitwiser.number
}

data "google_project" "splitwiser" {
}

resource "google_storage_bucket" "functions-source-code" {
  name     = "splitwiser-source-code-${local.project_number}"
  location = local.storage_location
}

data "archive_file" "source-code" {
  type        = "zip"
  source_dir  = "./"
  output_path = "./source.zip"
}

resource "google_storage_bucket_object" "functions-source-code" {
  name   = data.archive_file.source-code.output_sha
  bucket = google_storage_bucket.functions-source-code.name
  source = data.archive_file.source-code.output_path
}

resource "google_pubsub_topic" "start-bot" {
  name = "start-bot"
}

resource "google_pubsub_topic" "rotate-secret" {
  name = "rotate-secret"
}

resource "google_pubsub_topic_iam_member" "service-agent-rotate-secret-publisher" {
  topic  = google_pubsub_topic.rotate-secret.name
  member = "serviceAccount:service-${local.project_number}@gcp-sa-secretmanager.iam.gserviceaccount.com"
  role   = "roles/pubsub.publisher"
}
