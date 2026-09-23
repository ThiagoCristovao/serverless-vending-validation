variable "projeto_id" {
  type        = string
  description = "ID do projeto GCP."
}

variable "prefixo" {
  type        = string
  description = "Prefixo dos nomes de recurso (ex.: svv-dev)."
}

variable "conta_publicador_email" {
  type        = string
  description = "E-mail da conta de serviço que publica no tópico (função de validação)."
}

variable "conta_assinante_email" {
  type        = string
  description = "E-mail da conta de serviço que consome a assinatura (sistema central simulado)."
}

variable "retencao_mensagens" {
  type        = string
  description = "Retenção de mensagens no tópico e nas assinaturas (máximo de 7 dias)."
  default     = "604800s"
}

variable "prazo_confirmacao_segundos" {
  type        = number
  description = "Prazo para o consumidor confirmar (ack) uma mensagem."
  default     = 60
}

variable "espera_minima" {
  type        = string
  description = "Espera mínima entre retentativas de entrega."
  default     = "10s"
}

variable "espera_maxima" {
  type        = string
  description = "Espera máxima entre retentativas de entrega."
  default     = "600s"
}

variable "max_tentativas_entrega" {
  type        = number
  description = "Tentativas de entrega antes de encaminhar à fila de mensagens mortas (5 a 100)."
  default     = 5
}
