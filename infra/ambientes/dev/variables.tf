variable "projeto_id" {
  type        = string
  description = "ID do projeto GCP do ambiente de desenvolvimento."
}

variable "regiao" {
  type        = string
  description = "Região de todos os recursos regionais (ADR-0003)."
  default     = "us-east1"
}
