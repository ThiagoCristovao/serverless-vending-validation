# -----------------------------------------------------------------------------
# Contas de serviço e papéis segundo o menor privilégio.
# Papéis ligados a um recurso específico (tópico, assinatura, função) são
# concedidos no módulo que cria o recurso; aqui ficam só os de projeto.
# -----------------------------------------------------------------------------

resource "google_service_account" "funcao_validacao" {
  project      = var.projeto_id
  account_id   = "${var.prefixo}-funcao-validacao"
  display_name = "Serviço de validação (Cloud Run function)"
  description  = "Identidade da função svv-validacao: lê e escreve no Firestore, publica no Pub/Sub, lê a chave HMAC."
}

resource "google_service_account" "gateway" {
  project      = var.projeto_id
  account_id   = "${var.prefixo}-gateway"
  display_name = "API Gateway"
  description  = "Identidade com que o API Gateway invoca a função (roles/run.invoker concedido no módulo funcao)."
}

resource "google_service_account" "central_simulado" {
  project      = var.projeto_id
  account_id   = "${var.prefixo}-central-simulado"
  display_name = "Sistema central simulado"
  description  = "Consumidor da assinatura de validações (roles/pubsub.subscriber concedido no módulo pubsub)."
}

locals {
  papeis_funcao_projeto = [
    "roles/datastore.user",
    "roles/logging.logWriter",
    "roles/monitoring.metricWriter",
    "roles/cloudtrace.agent",
  ]
}

resource "google_project_iam_member" "funcao" {
  for_each = toset(local.papeis_funcao_projeto)

  project = var.projeto_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.funcao_validacao.email}"
}

# Segredo da chave HMAC usada para derivar a contrassenha (ADR-0002).
# O Terraform cria apenas o contêiner; o valor é adicionado fora dele com
# `make segredo-hmac-gerar`, para nunca passar pelo estado.
resource "google_secret_manager_secret" "chave_hmac" {
  project   = var.projeto_id
  secret_id = "${var.prefixo}-${var.nome_segredo_hmac}"

  replication {
    auto {}
  }
}

resource "google_secret_manager_secret_iam_member" "funcao_le_chave" {
  project   = var.projeto_id
  secret_id = google_secret_manager_secret.chave_hmac.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.funcao_validacao.email}"
}
