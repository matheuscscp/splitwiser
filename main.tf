terraform {
  backend "gcs" {
    bucket = "splitwiser-tf-state"
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
