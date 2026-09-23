# user_project_override + billing_project: com credenciais de usuário (ADC), as
# APIs do Firebase e de orçamentos exigem um projeto de cota explícito; sem isso
# respondem "Error 403: The caller does not have permission".
provider "google" {
  project        = var.projeto_id
  region         = var.regiao
  default_labels = local.rotulos

  user_project_override = true
  billing_project       = var.projeto_id
}

provider "google-beta" {
  project        = var.projeto_id
  region         = var.regiao
  default_labels = local.rotulos

  user_project_override = true
  billing_project       = var.projeto_id
}
