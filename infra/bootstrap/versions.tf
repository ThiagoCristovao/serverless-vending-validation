terraform {
  required_version = ">= 1.16.0, < 2.0.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 8.3"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 8.3"
    }
  }
}
