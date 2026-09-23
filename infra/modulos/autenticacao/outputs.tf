output "app_android_id" {
  description = "Identificador do app Android no Firebase."
  value       = google_firebase_android_app.operador.app_id
}

output "google_services_nome_arquivo" {
  description = "Nome do arquivo de configuração do app Android (google-services.json)."
  value       = data.google_firebase_android_app_config.operador.config_filename
}

output "google_services_json" {
  description = "Conteúdo decodificado do google-services.json. Sensível; gravar em aplicativo/android/app/ sem versionar."
  value       = base64decode(data.google_firebase_android_app_config.operador.config_file_contents)
  sensitive   = true
}
