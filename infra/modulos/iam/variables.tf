variable "projeto_id" {
  type        = string
  description = "ID do projeto GCP."
}

variable "prefixo" {
  type        = string
  description = "Prefixo dos nomes de recurso (ex.: svv-dev)."
}

variable "nome_segredo_hmac" {
  type        = string
  description = "Sufixo do nome do segredo que guarda a chave HMAC da contrassenha."
  default     = "chave-hmac-contrassenha"
}
