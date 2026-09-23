output "canal_email_id" {
  description = "Identificador do canal de notificação por e-mail."
  value       = google_monitoring_notification_channel.email.id
}

output "painel_id" {
  description = "Identificador do painel de monitoramento."
  value       = google_monitoring_dashboard.principal.id
}

output "metrica_erros_nome" {
  description = "Nome da métrica derivada de log de erros por código."
  value       = google_logging_metric.erros_por_codigo.name
}
