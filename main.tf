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
  project = "splitwiser-356519"
  region  = "europe-west1" # Low CO₂
}

resource "google_storage_bucket" "functions-source-code" {
  location = upper(local.region)
  name     = "functions-source-code"
}

data "archive_file" "source-code" {
  type        = "zip"
  source_dir  = "./"
  output_path = "/tmp/source.zip"
}

resource "google_storage_bucket_object" "functions-source-code" {
  name   = "source.zip"
  bucket = google_storage_bucket.functions-source-code.name
  source = data.archive_file.source-code.output_path
}

resource "google_pubsub_topic" "start-bot" {
  name = "start-bot"
}
