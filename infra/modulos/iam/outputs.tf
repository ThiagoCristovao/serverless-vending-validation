output "conta_funcao_email" {
  description = "E-mail da conta de serviço da função de validação."
  value       = google_service_account.funcao_validacao.email
}

output "conta_gateway_email" {
  description = "E-mail da conta de serviço do API Gateway."
  value       = google_service_account.gateway.email
}

output "conta_central_email" {
  description = "E-mail da conta de serviço do sistema central simulado."
  value       = google_service_account.central_simulado.email
}

output "segredo_hmac_nome" {
  description = "Nome (secret_id) do segredo da chave HMAC."
  value       = google_secret_manager_secret.chave_hmac.secret_id
}

output "segredo_hmac_id" {
  description = "Identificador completo do segredo (projects/.../secrets/...)."
  value       = google_secret_manager_secret.chave_hmac.id
}
