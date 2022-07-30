terraform {
  required_version = "1.2.6"

  required_providers {
    google = {
      source = "hashicorp/google"
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
  project = "splitwiser-356519"
  region = "europe-west1" # Low CO2
}
