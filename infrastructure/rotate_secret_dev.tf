resource "google_service_account" "rotate-secret-dev" {
  account_id   = "rotate-secret-dev"
  display_name = "RotateSecret Development"
}

resource "google_project_iam_member" "rotate-secret-viewer-dev" {
  project = var.project
  member  = "serviceAccount:${google_service_account.rotate-secret-dev.email}"
  role    = "roles/secretmanager.viewer"
}

resource "google_secret_manager_secret_iam_member" "rotate-secret-start-bot-jwt-secret-version-manager-dev" {
  secret_id = google_secret_manager_secret.start-bot-jwt-secret-dev.id
  member    = "serviceAccount:${google_service_account.rotate-secret-dev.email}"
  role      = "roles/secretmanager.secretVersionManager"
}
