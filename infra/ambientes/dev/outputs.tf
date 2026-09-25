output "contas_de_servico" {
  description = "E-mails das contas de serviço do ambiente."
  value = {
    funcao_validacao = module.iam.conta_funcao_email
    gateway          = module.iam.conta_gateway_email
    central_simulado = module.iam.conta_central_email
  }
}

output "segredo_hmac_nome" {
  description = "Nome do segredo da chave HMAC (valor adicionado com `make segredo-hmac-gerar`)."
  value       = module.iam.segredo_hmac_nome
}

output "pubsub" {
  description = "Nomes dos tópicos e assinaturas."
  value = {
    topico_validacoes  = module.pubsub.topico_validacoes_nome
    topico_dlq         = module.pubsub.topico_dlq_nome
    assinatura_central = module.pubsub.assinatura_central_nome
    assinatura_dlq     = module.pubsub.assinatura_dlq_inspecao_nome
  }
}

output "firestore_banco" {
  description = "Nome do banco Firestore."
  value       = module.firestore.banco_nome
}

output "app_android_id" {
  description = "Identificador do app Android no Firebase."
  value       = module.autenticacao.app_android_id
}

output "google_services_json" {
  description = "Conteúdo do google-services.json (sensível). Gravar com `make app-google-services`."
  value       = module.autenticacao.google_services_json
  sensitive   = true
}

output "funcao_url" {
  description = "URL direta do serviço Cloud Run da função (invocável só por contas autorizadas)."
  value       = module.funcao.url
}

output "gateway_hostname" {
  description = "Host do API Gateway; o aplicativo chama https://<host>/v1/..."
  value       = module.gateway.hostname
}
