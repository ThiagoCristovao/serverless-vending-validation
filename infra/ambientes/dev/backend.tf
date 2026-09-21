# Estado remoto no Cloud Storage (bucket criado pelo bootstrap).
# O nome do bucket vem de backend.hcl (ignorado pelo git):
#   make infra-dev-init   ->   terraform init -backend-config=backend.hcl
terraform {
  backend "gcs" {
    prefix = "ambientes/dev"
  }
}
