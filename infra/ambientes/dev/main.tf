# -----------------------------------------------------------------------------
# Ambiente de desenvolvimento (raiz Terraform).
#
# Este arquivo compõe os módulos de infra/modulos. Na Sprint 0 ele só define
# convenções (prefixo, rótulos); os módulos entram na Sprint 3 (recursos que
# não dependem do código) e na Sprint 5 (função e gateway).
#
# Critério da Sprint 3: `terraform destroy` seguido de `terraform apply` recria
# o ambiente integralmente, sem intervenção manual no console.
# -----------------------------------------------------------------------------

locals {
  ambiente = "dev"
  prefixo  = "svv-${local.ambiente}"

  rotulos = {
    projeto        = "svv"
    ambiente       = local.ambiente
    gerenciado_por = "terraform"
  }
}

# --- Sprint 3: recursos independentes do código -------------------------------
#
# module "firestore" {
#   source     = "../../modulos/firestore"
#   projeto_id = var.projeto_id
#   regiao     = var.regiao
# }
#
# module "pubsub" {
#   source     = "../../modulos/pubsub"
#   projeto_id = var.projeto_id
#   prefixo    = local.prefixo
# }
#
# module "autenticacao" {
#   source     = "../../modulos/autenticacao"
#   projeto_id = var.projeto_id
# }
#
# module "iam" {
#   source     = "../../modulos/iam"
#   projeto_id = var.projeto_id
#   prefixo    = local.prefixo
# }
#
# module "observabilidade" {
#   source     = "../../modulos/observabilidade"
#   projeto_id = var.projeto_id
# }

# --- Sprint 5: função e gateway ------------------------------------------------
#
# module "funcao" {
#   source     = "../../modulos/funcao"
#   projeto_id = var.projeto_id
#   regiao     = var.regiao
#   prefixo    = local.prefixo
# }
#
# module "gateway" {
#   source     = "../../modulos/gateway"
#   projeto_id = var.projeto_id
#   regiao     = var.regiao
#   prefixo    = local.prefixo
# }
