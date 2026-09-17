variable "projeto_id" {
  type        = string
  description = "ID do projeto GCP já criado e vinculado ao faturamento (ex.: svv-dev-1234)."

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{4,28}[a-z0-9]$", var.projeto_id))
    error_message = "projeto_id deve seguir as regras de ID de projeto da GCP (6 a 30 caracteres, minúsculas, dígitos e hifens)."
  }
}

variable "regiao" {
  type        = string
  description = "Região padrão dos recursos. us-east1 é a região suportada pelo API Gateway mais próxima do Brasil (ADR-0003)."
  default     = "us-east1"
}

variable "bucket_estado_nome" {
  type        = string
  description = "Nome globalmente único do bucket que guardará o estado remoto do Terraform (ex.: svv-dev-1234-tfstate)."

  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9._-]{1,61}[a-z0-9]$", var.bucket_estado_nome))
    error_message = "bucket_estado_nome deve seguir as regras de nome de bucket do Cloud Storage."
  }
}

variable "orcamento_habilitado" {
  type        = bool
  description = "Cria um orçamento mensal com alertas (recomendado)."
  default     = true
}

variable "conta_faturamento_id" {
  type        = string
  description = "ID da conta de faturamento no formato XXXXXX-XXXXXX-XXXXXX. Obrigatório quando orcamento_habilitado = true. Obtenha com `gcloud billing accounts list`."
  default     = null

  validation {
    condition     = !var.orcamento_habilitado || (var.conta_faturamento_id != null && can(regex("^[0-9A-F]{6}-[0-9A-F]{6}-[0-9A-F]{6}$", var.conta_faturamento_id)))
    error_message = "Com orcamento_habilitado = true, conta_faturamento_id deve ser informado no formato XXXXXX-XXXXXX-XXXXXX."
  }
}

variable "orcamento_valor" {
  type        = number
  description = "Valor mensal do orçamento, na moeda da conta de faturamento."
  default     = 20
}

variable "orcamento_moeda" {
  type        = string
  description = "Código da moeda do orçamento. Deve coincidir com a moeda da conta de faturamento (BRL para contas brasileiras, USD para contas em dólar)."
  default     = "BRL"
}
