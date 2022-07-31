resource "google_pubsub_topic_iam_member" "service-agent-identity-rotate-jwt-secret-publisher" {
  topic  = google_pubsub_topic.rotate-jwt-secret.name
  member = "serviceAccount:service-281540018258@gcp-sa-secretmanager.iam.gserviceaccount.com"
  role   = "roles/pubsub.publisher"
}
