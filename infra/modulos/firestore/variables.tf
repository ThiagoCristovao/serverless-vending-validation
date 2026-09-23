variable "projeto_id" {
  type        = string
  description = "ID do projeto GCP."
}

variable "regiao" {
  type        = string
  description = "Região do banco (regional). Deve ser a mesma da função para evitar latência entre regiões (ADR-0003)."
}

variable "protecao_exclusao" {
  type        = bool
  description = "Protege o banco contra exclusão. Falso em dev para permitir `destroy` + `apply` (critério da Sprint 3); verdadeiro em ambientes com dados reais."
  default     = false
}

variable "backup_diario" {
  type        = bool
  description = "Agenda backups diários do banco."
  default     = true
}

variable "retencao_backup" {
  type        = string
  description = "Retenção dos backups diários, em segundos (7 dias)."
  default     = "604800s"
}
