output "banco_nome" {
  description = "Nome do banco Firestore."
  value       = google_firestore_database.principal.name
}

output "banco_id" {
  description = "Identificador completo do banco."
  value       = google_firestore_database.principal.id
}
