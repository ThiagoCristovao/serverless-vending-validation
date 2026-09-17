provider "google" {
  project        = var.projeto_id
  region         = var.regiao
  default_labels = local.rotulos
}

provider "google-beta" {
  project        = var.projeto_id
  region         = var.regiao
  default_labels = local.rotulos
}
