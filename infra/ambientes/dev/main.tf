# -----------------------------------------------------------------------------
# Ambiente de desenvolvimento (raiz Terraform).
#
# Compõe os módulos de infra/modulos. Sprint 3: recursos que não dependem do
# código da aplicação. Sprint 5: função e gateway (comentados abaixo).
#
# Firebase e Identity Platform (irreversíveis) são geridos no bootstrap, o que
# mantém este ambiente inteiramente destrutível e recriável.
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

module "iam" {
  source = "../../modulos/iam"

  projeto_id = var.projeto_id
  prefixo    = local.prefixo
}

module "autenticacao" {
  source = "../../modulos/autenticacao"

  projeto_id     = var.projeto_id
  pacote_android = var.pacote_android
}

module "firestore" {
  source = "../../modulos/firestore"

  projeto_id        = var.projeto_id
  regiao            = var.regiao
  protecao_exclusao = false
}

module "pubsub" {
  source = "../../modulos/pubsub"

  projeto_id             = var.projeto_id
  prefixo                = local.prefixo
  conta_publicador_email = module.iam.conta_funcao_email
  conta_assinante_email  = module.iam.conta_central_email
}

module "observabilidade" {
  source = "../../modulos/observabilidade"

  projeto_id              = var.projeto_id
  prefixo                 = local.prefixo
  email_alertas           = var.email_alertas
  nome_funcao             = "${local.prefixo}-validacao"
  assinatura_central_nome = module.pubsub.assinatura_central_nome
  assinatura_dlq_nome     = module.pubsub.assinatura_dlq_inspecao_nome
}

# --- Sprint 5: função e gateway ------------------------------------------------
#
# module "funcao" {
#   source = "../../modulos/funcao"
#
#   projeto_id          = var.projeto_id
#   regiao              = var.regiao
#   prefixo             = local.prefixo
#   conta_funcao_email  = module.iam.conta_funcao_email
#   conta_gateway_email = module.iam.conta_gateway_email
#   topico_validacoes   = module.pubsub.topico_validacoes_nome
#   segredo_hmac_id     = module.iam.segredo_hmac_id
# }
#
# module "gateway" {
#   source = "../../modulos/gateway"
#
#   projeto_id          = var.projeto_id
#   regiao              = var.regiao
#   prefixo             = local.prefixo
#   conta_gateway_email = module.iam.conta_gateway_email
#   url_funcao          = module.funcao.url
# }
