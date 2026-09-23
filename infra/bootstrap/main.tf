# -----------------------------------------------------------------------------
# Bootstrap do projeto GCP.
#
# Provisiona o que precisa existir ANTES dos demais módulos Terraform:
#   1. habilitação das APIs usadas pelo ecossistema;
#   2. bucket do Cloud Storage que guardará o estado remoto dos ambientes;
#   3. orçamento mensal com alertas (opcional, mas recomendado).
#
# Executado UMA vez por projeto, com estado LOCAL (ainda não existe bucket).
# O arquivo terraform.tfstate gerado aqui fica fora do versionamento; guarde
# uma cópia em local seguro. Ver README.md neste diretório.
# -----------------------------------------------------------------------------

provider "google" {
  project = var.projeto_id
  region  = var.regiao

  # A API de orçamentos (billingbudgets) exige um projeto de cota quando a
  # autenticação usa credenciais de usuário (ADC). Sem estas duas linhas o
  # apply falha com "Error 403: ... requires a quota project".
  user_project_override = true
  billing_project       = var.projeto_id
}

data "google_project" "atual" {
  project_id = var.projeto_id
}

locals {
  rotulos = {
    projeto        = "svv"
    ambiente       = "compartilhado"
    gerenciado_por = "terraform"
  }

  # APIs necessárias ao longo de todas as sprints. Habilitar tudo agora evita
  # falhas intermitentes de "API not enabled" durante a Sprint 3 e a Sprint 5.
  apis = [
    "serviceusage.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "run.googleapis.com",
    "cloudfunctions.googleapis.com",
    "cloudbuild.googleapis.com",
    "artifactregistry.googleapis.com",
    "eventarc.googleapis.com",
    "apigateway.googleapis.com",
    "servicecontrol.googleapis.com",
    "servicemanagement.googleapis.com",
    "pubsub.googleapis.com",
    "firestore.googleapis.com",
    "firebase.googleapis.com",
    "firebaserules.googleapis.com",
    "identitytoolkit.googleapis.com",
    "secretmanager.googleapis.com",
    "logging.googleapis.com",
    "monitoring.googleapis.com",
    "cloudtrace.googleapis.com",
    "storage.googleapis.com",
    "billingbudgets.googleapis.com",
  ]
}

resource "google_project_service" "apis" {
  for_each = toset(local.apis)

  project = var.projeto_id
  service = each.value

  # Desabilitar APIs ao destruir o bootstrap derrubaria todo o ambiente.
  disable_on_destroy         = false
  disable_dependent_services = false
}

# Bucket do estado remoto. Versionamento ligado permite recuperar um estado
# corrompido; o bloqueio de estado é nativo do backend GCS.
resource "google_storage_bucket" "estado_terraform" {
  name          = var.bucket_estado_nome
  project       = var.projeto_id
  location      = var.regiao
  storage_class = "STANDARD"

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = false

  versioning {
    enabled = true
  }

  lifecycle_rule {
    condition {
      num_newer_versions = 10
    }
    action {
      type = "Delete"
    }
  }

  labels = local.rotulos

  depends_on = [google_project_service.apis]
}

# Orçamento mensal com alertas em 50 %, 90 % e 100 %. Sem canais de notificação
# explícitos, os avisos vão por e-mail aos administradores da conta de
# faturamento. Risco registrado em docs/planejamento-sprints.md (Sprint 10).
resource "google_billing_budget" "orcamento" {
  count = var.orcamento_habilitado ? 1 : 0

  billing_account = var.conta_faturamento_id
  display_name    = "svv-orcamento-mensal"

  budget_filter {
    projects = ["projects/${data.google_project.atual.number}"]
  }

  amount {
    specified_amount {
      currency_code = var.orcamento_moeda
      units         = tostring(var.orcamento_valor)
    }
  }

  dynamic "threshold_rules" {
    for_each = [0.5, 0.9, 1.0]
    content {
      threshold_percent = threshold_rules.value
      spend_basis       = "CURRENT_SPEND"
    }
  }

  depends_on = [google_project_service.apis]
}
