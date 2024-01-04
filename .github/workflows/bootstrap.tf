terraform {
  backend "gcs" {
    bucket = "splitwiser-bootstrap-tf-state"
  }
}

provider "google" {
  project = "splitwiser-telegram-bot"
}

data "google_project" "splitwiser_telegram_bot" {
}

locals {
  sa_email         = "terraform@splitwiser-telegram-bot.iam.gserviceaccount.com"
  wi_pool_name     = google_iam_workload_identity_pool.github_actions.name
  gh_sub_prefix    = "repo:matheuscscp/splitwiser:environment"
  wi_member_prefix = "principal://iam.googleapis.com/${local.wi_pool_name}/subject/${local.gh_sub_prefix}"
}

resource "google_service_account_iam_member" "workload_identity_user" {
  service_account_id = "projects/${data.google_project.splitwiser_telegram_bot.project_id}/serviceAccounts/${local.sa_email}"
  role               = "roles/iam.workloadIdentityUser"
  member             = "${local.wi_member_prefix}:terraform"
}

resource "google_storage_bucket" "terraform_state" {
  name                     = "splitwiser-tf-state"
  location                 = "us"
  public_access_prevention = "enforced"

  versioning {
    enabled = true
  }

  lifecycle_rule {
    action {
      type          = "SetStorageClass"
      storage_class = "ARCHIVE"
    }
    condition {
      num_newer_versions = 1
    }
  }

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      num_newer_versions = 100
    }
  }
}

resource "google_storage_bucket_iam_member" "tf_state_manager" {
  bucket = google_storage_bucket.terraform_state.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${local.sa_email}"
}

resource "google_iam_workload_identity_pool" "github_actions" {
  workload_identity_pool_id = "github-actions"
}

resource "google_iam_workload_identity_pool_provider" "github_actions" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github_actions.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-actions"
  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
  attribute_mapping = {
    "google.subject" = "assertion.sub" # repo:{repo_org}/{repo_name}:environment:{env_name}
  }
}
