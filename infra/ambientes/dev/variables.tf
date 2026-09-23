variable "projeto_id" {
  type        = string
  description = "ID do projeto GCP do ambiente de desenvolvimento."
}

variable "regiao" {
  type        = string
  description = "Região de todos os recursos regionais (ADR-0003)."
  default     = "us-east1"
}

variable "pacote_android" {
  type        = string
  description = "Nome do pacote do aplicativo Android registrado no Firebase."
  default     = "br.edu.utfpr.svv"
}

variable "email_alertas" {
  type        = string
  description = "E-mail que recebe os alertas do Cloud Monitoring."
}
