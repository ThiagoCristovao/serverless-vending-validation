variable "projeto_id" {
  type        = string
  description = "ID do projeto GCP."
}

variable "regiao" {
  type        = string
  description = "Região da função (a mesma do gateway e do Firestore, ADR-0003)."
}

variable "prefixo" {
  type        = string
  description = "Prefixo dos nomes de recurso (ex.: svv-dev)."
}

variable "codigo_fonte_dir" {
  type        = string
  description = "Diretório do módulo Go do serviço (servico/), empacotado e enviado ao Cloud Storage."
}

variable "conta_funcao_email" {
  type        = string
  description = "Conta de serviço com que a função executa."
}

variable "conta_gateway_email" {
  type        = string
  description = "Conta de serviço do API Gateway, única autorizada a invocar a função pelo contrato público."
}

variable "conta_scheduler_email" {
  type        = string
  description = "Conta de serviço do Cloud Scheduler, autorizada a invocar o endpoint interno de reconciliação."
}

variable "topico_validacoes" {
  type        = string
  description = "Nome do tópico Pub/Sub em que a função publica os eventos."
}

variable "segredo_hmac_nome" {
  type        = string
  description = "Nome (secret_id) do segredo com a chave HMAC da contrassenha."
}

variable "runtime" {
  type        = string
  description = "Runtime Go das Cloud Run functions (GA em 2026-09: go127)."
  default     = "go127"
}

variable "ponto_de_entrada" {
  type        = string
  description = "Nome registrado em functions.HTTP no pacote raiz do serviço."
  default     = "ValidarMaquina"
}

variable "memoria" {
  type        = string
  description = "Memória por instância."
  default     = "256Mi"
}

variable "tempo_limite_segundos" {
  type        = number
  description = "Tempo máximo de uma requisição."
  default     = 30
}

variable "max_instancias" {
  type        = number
  description = "Limite de instâncias (controle de custo nos testes de carga)."
  default     = 10
}

variable "min_instancias" {
  type        = number
  description = "Instâncias mínimas. Zero em uso normal; 1 apenas durante a medição de cold start (Sprint 10)."
  default     = 0
}

variable "prazo_validacao" {
  type        = string
  description = "Prazo para concluir uma validação (RN-04), no formato de duração do Go."
  default     = "15m"
}

variable "nivel_log" {
  type        = string
  description = "Nível de log do serviço: debug, info, warn ou error."
  default     = "info"
}

variable "reconciliacao_cron" {
  type        = string
  description = "Agenda (cron) do job que reconcilia publicações pendentes (ADR-0012)."
  default     = "*/5 * * * *"
}
