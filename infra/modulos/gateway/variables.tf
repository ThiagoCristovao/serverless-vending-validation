variable "projeto_id" {
  type        = string
  description = "ID do projeto GCP."
}

variable "regiao" {
  type        = string
  description = "Região do gateway (precisa ser uma das suportadas pelo API Gateway, ADR-0003)."
}

variable "prefixo" {
  type        = string
  description = "Prefixo dos nomes de recurso (ex.: svv-dev)."
}

variable "conta_gateway_email" {
  type        = string
  description = "Conta de serviço com que o gateway invoca a função."
}

variable "url_funcao" {
  type        = string
  description = "URL do serviço Cloud Run da função (saída do módulo funcao)."
}

variable "contrato_caminho" {
  type        = string
  description = "Caminho do contrato OpenAPI canônico (api/openapi.yaml)."
}

variable "caminho_base_backend" {
  type        = string
  description = "Sufixo acrescentado à URL da função no x-google-backend. Vazio quando o gateway já repassa o prefixo /v1 do contrato."
  default     = ""
}
