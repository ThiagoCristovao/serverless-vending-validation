variable "projeto_id" {
  type        = string
  description = "ID do projeto GCP (já habilitado no Firebase pelo bootstrap)."
}

variable "pacote_android" {
  type        = string
  description = "Nome do pacote Android do aplicativo (ex.: br.edu.utfpr.svv)."
}

variable "nome_app_android" {
  type        = string
  description = "Nome de exibição do aplicativo no Firebase."
  default     = "svv-aplicativo"
}
