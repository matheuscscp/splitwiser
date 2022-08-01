terraform {
  required_version = "1.2.6"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.31.0"
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
  region  = local.region
  project = local.project
}

provider "google-beta" {
  region  = local.region
  project = local.project
}

locals {
  region         = "europe-west1" # Low CO₂
  project        = "splitwiser-telegram-bot"
  project_number = data.google_project.splitwiser-telegram-bot.number
}

data "google_project" "splitwiser-telegram-bot" {
}

module "infrastructure" {
  source         = "./infrastructure"
  region         = local.region
  project        = local.project
  project_number = local.project_number
}
