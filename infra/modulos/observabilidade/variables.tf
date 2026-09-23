variable "projeto_id" {
  type        = string
  description = "ID do projeto GCP."
}

variable "prefixo" {
  type        = string
  description = "Prefixo dos nomes de recurso (ex.: svv-dev)."
}

variable "email_alertas" {
  type        = string
  description = "E-mail que recebe os alertas do Cloud Monitoring."
}

variable "nome_funcao" {
  type        = string
  description = "Nome do serviço Cloud Run da função de validação (usado nos filtros de métrica)."
}

variable "assinatura_central_nome" {
  type        = string
  description = "Nome da assinatura consumida pelo sistema central."
}

variable "assinatura_dlq_nome" {
  type        = string
  description = "Nome da assinatura de inspeção da fila de mensagens mortas."
}

variable "limiar_backlog" {
  type        = number
  description = "Mensagens não entregues na assinatura principal que disparam alerta após 10 minutos."
  default     = 100
}
