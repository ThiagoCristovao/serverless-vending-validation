output "topico_validacoes_nome" {
  description = "Nome do tópico principal."
  value       = google_pubsub_topic.validacoes.name
}

output "topico_validacoes_id" {
  description = "Identificador completo do tópico principal (projects/.../topics/...)."
  value       = google_pubsub_topic.validacoes.id
}

output "topico_dlq_nome" {
  description = "Nome do tópico de mensagens mortas."
  value       = google_pubsub_topic.validacoes_dlq.name
}

output "assinatura_central_nome" {
  description = "Nome da assinatura consumida pelo sistema central."
  value       = google_pubsub_subscription.sistema_central.name
}

output "assinatura_dlq_inspecao_nome" {
  description = "Nome da assinatura de inspeção da DLQ."
  value       = google_pubsub_subscription.dlq_inspecao.name
}
