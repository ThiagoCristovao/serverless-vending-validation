output "projeto_id" {
  description = "ID do projeto GCP."
  value       = var.projeto_id
}

output "projeto_numero" {
  description = "Número do projeto GCP (usado em políticas de IAM e no orçamento)."
  value       = data.google_project.atual.number
}

output "bucket_estado_nome" {
  description = "Bucket do estado remoto. Use este valor em infra/ambientes/<ambiente>/backend.hcl."
  value       = google_storage_bucket.estado_terraform.name
}

output "backend_hcl_sugerido" {
  description = "Conteúdo sugerido para infra/ambientes/dev/backend.hcl."
  value       = "bucket = \"${google_storage_bucket.estado_terraform.name}\""
}

output "projeto_firebase" {
  description = "Projeto habilitado no Firebase (adotado por import)."
  value       = google_firebase_project.principal.project
}
