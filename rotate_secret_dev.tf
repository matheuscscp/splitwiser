resource "google_service_account" "rotate-secret-dev" {
  account_id   = "rotate-secret-dev"
  display_name = "RotateSecret Development"
}

resource "google_project_iam_member" "rotate-secret-version-manager-dev" {
  project = local.project
  member  = "serviceAccount:${google_service_account.rotate-secret-dev.email}"
  role    = "roles/secretmanager.secretVersionManager"
}
