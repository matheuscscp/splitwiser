resource "google_service_account" "bot-dev" {
  account_id   = "bot-dev"
  display_name = "Bot Development"
}

resource "google_storage_bucket_iam_member" "bot-checkpoint-dev-object-admin" {
  bucket = google_storage_bucket.bot-checkpoint-dev.name
  member = "serviceAccount:${google_service_account.bot-dev.email}"
  role   = "roles/storage.objectAdmin"
}

resource "google_storage_bucket" "bot-checkpoint-dev" {
  name     = "splitwiser-checkpoint-dev"
  location = local.storage_location
}
