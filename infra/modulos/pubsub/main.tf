# -----------------------------------------------------------------------------
# Mensageria entre o serviço de validação e o sistema central.
# Desenho em docs/arquitetura.md, seção 6.
# -----------------------------------------------------------------------------

resource "google_pubsub_topic" "validacoes" {
  project = var.projeto_id
  name    = "${var.prefixo}-validacoes"

  message_retention_duration = var.retencao_mensagens
}

resource "google_pubsub_topic" "validacoes_dlq" {
  project = var.projeto_id
  name    = "${var.prefixo}-validacoes-dlq"

  message_retention_duration = var.retencao_mensagens
}

# Assinatura consumida pelo sistema central. Ordenação por idMaquina garante
# que "concluida" não chegue antes de "iniciada" para a mesma máquina.
resource "google_pubsub_subscription" "sistema_central" {
  project = var.projeto_id
  name    = "${var.prefixo}-sistema-central"
  topic   = google_pubsub_topic.validacoes.id

  ack_deadline_seconds       = var.prazo_confirmacao_segundos
  message_retention_duration = var.retencao_mensagens
  retain_acked_messages      = false
  enable_message_ordering    = true

  # Nunca expira por inatividade: o sistema central pode ficar dias fora.
  expiration_policy {
    ttl = ""
  }

  retry_policy {
    minimum_backoff = var.espera_minima
    maximum_backoff = var.espera_maxima
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.validacoes_dlq.id
    max_delivery_attempts = var.max_tentativas_entrega
  }

  depends_on = [
    google_pubsub_topic_iam_member.agente_publica_dlq,
  ]
}

# Assinatura de inspeção da DLQ (testes da Sprint 8 e análise manual).
resource "google_pubsub_subscription" "dlq_inspecao" {
  project = var.projeto_id
  name    = "${var.prefixo}-validacoes-dlq-inspecao"
  topic   = google_pubsub_topic.validacoes_dlq.id

  ack_deadline_seconds       = 60
  message_retention_duration = var.retencao_mensagens

  expiration_policy {
    ttl = ""
  }
}

# O agente de serviço do Pub/Sub precisa publicar na DLQ e assinar a origem
# para que o encaminhamento de mensagens mortas funcione.
resource "google_project_service_identity" "pubsub" {
  provider = google-beta

  project = var.projeto_id
  service = "pubsub.googleapis.com"
}

resource "google_pubsub_topic_iam_member" "agente_publica_dlq" {
  project = var.projeto_id
  topic   = google_pubsub_topic.validacoes_dlq.id
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${google_project_service_identity.pubsub.email}"
}

resource "google_pubsub_subscription_iam_member" "agente_assina_origem" {
  project      = var.projeto_id
  subscription = google_pubsub_subscription.sistema_central.id
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${google_project_service_identity.pubsub.email}"
}

# Publicador (função de validação) e assinante (sistema central simulado).
resource "google_pubsub_topic_iam_member" "publicador" {
  project = var.projeto_id
  topic   = google_pubsub_topic.validacoes.id
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${var.conta_publicador_email}"
}

resource "google_pubsub_subscription_iam_member" "assinante" {
  project      = var.projeto_id
  subscription = google_pubsub_subscription.sistema_central.id
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${var.conta_assinante_email}"
}

resource "google_pubsub_subscription_iam_member" "assinante_dlq" {
  project      = var.projeto_id
  subscription = google_pubsub_subscription.dlq_inspecao.id
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${var.conta_assinante_email}"
}
