resource "google_service_account" "start-bot-dev" {
  account_id   = "start-bot-dev"
  display_name = "StartBot Development"
}

resource "google_secret_manager_secret_iam_member" "start-bot-jwt-secret-accessor-dev" {
  secret_id = google_secret_manager_secret.start-bot-jwt-secret-dev.id
  member    = "serviceAccount:${google_service_account.start-bot-dev.email}"
  role      = "roles/secretmanager.secretAccessor"
}

resource "google_secret_manager_secret" "start-bot-jwt-secret-dev" {
  secret_id = "start-bot-jwt-secret-dev"
  replication {
    auto {}
  }
  labels = {
    "num-bytes" = "32"
  }
}
