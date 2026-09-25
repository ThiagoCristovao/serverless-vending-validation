output "hostname" {
  description = "Host público do gateway; base das chamadas do aplicativo (https://<hostname>/v1/...)."
  value       = google_api_gateway_gateway.gateway.default_hostname
}

output "api_config_id" {
  description = "Identificador da configuração de API implantada."
  value       = google_api_gateway_api_config.config.id
}
