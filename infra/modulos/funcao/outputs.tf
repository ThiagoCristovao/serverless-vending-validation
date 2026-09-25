output "nome" {
  description = "Nome da função (e do serviço Cloud Run subjacente)."
  value       = google_cloudfunctions2_function.validacao.name
}

output "url" {
  description = "URL do serviço Cloud Run da função; alvo do API Gateway e audiência do JWT do gateway."
  value       = google_cloudfunctions2_function.validacao.service_config[0].uri
}

output "objeto_fonte" {
  description = "Objeto do código-fonte implantado (carrega o hash do conteúdo)."
  value       = google_storage_bucket_object.fonte.name
}
