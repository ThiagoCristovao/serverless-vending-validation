# -----------------------------------------------------------------------------
# Serviço de validação como Cloud Run function (2ª geração) e o job do Cloud
# Scheduler que reconcilia publicações pendentes (ADR-0012).
# -----------------------------------------------------------------------------

# Empacota o módulo Go. O nome do objeto carrega o hash do conteúdo, então
# qualquer mudança no código gera uma nova versão da função.
data "archive_file" "fonte" {
  type        = "zip"
  source_dir  = var.codigo_fonte_dir
  output_path = "${path.module}/.build/${var.prefixo}-fonte.zip"
  excludes    = ["coverage.out"]
}

resource "google_storage_bucket" "fonte" {
  name          = "${var.projeto_id}-fonte-funcao"
  project       = var.projeto_id
  location      = var.regiao
  storage_class = "STANDARD"

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = true

  lifecycle_rule {
    condition {
      age = 30
    }
    action {
      type = "Delete"
    }
  }
}

resource "google_storage_bucket_object" "fonte" {
  name   = "funcao-${data.archive_file.fonte.output_md5}.zip"
  bucket = google_storage_bucket.fonte.name
  source = data.archive_file.fonte.output_path
}

resource "google_cloudfunctions2_function" "validacao" {
  name        = "${var.prefixo}-validacao"
  project     = var.projeto_id
  location    = var.regiao
  description = "Serviço de validação de máquinas de vending (TCC)."

  build_config {
    runtime     = var.runtime
    entry_point = var.ponto_de_entrada

    source {
      storage_source {
        bucket = google_storage_bucket.fonte.name
        object = google_storage_bucket_object.fonte.name
      }
    }
  }

  service_config {
    available_memory               = var.memoria
    timeout_seconds                = var.tempo_limite_segundos
    max_instance_count             = var.max_instancias
    min_instance_count             = var.min_instancias
    ingress_settings               = "ALLOW_ALL"
    all_traffic_on_latest_revision = true
    service_account_email          = var.conta_funcao_email

    environment_variables = {
      SVV_PROJETO_ID        = var.projeto_id
      SVV_TOPICO_VALIDACOES = var.topico_validacoes
      SVV_PRAZO_VALIDACAO   = var.prazo_validacao
      SVV_NIVEL_LOG         = var.nivel_log
    }

    secret_environment_variables {
      key        = "SVV_CHAVE_HMAC"
      project_id = var.projeto_id
      secret     = var.segredo_hmac_nome
      version    = "latest"
    }
  }
}

# Sem allUsers: só o gateway (contrato público) e o Scheduler (endpoint interno)
# podem invocar a função (RNF-08).
resource "google_cloud_run_service_iam_member" "gateway_invoca" {
  project  = var.projeto_id
  location = var.regiao
  service  = google_cloudfunctions2_function.validacao.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${var.conta_gateway_email}"
}

resource "google_cloud_run_service_iam_member" "scheduler_invoca" {
  project  = var.projeto_id
  location = var.regiao
  service  = google_cloudfunctions2_function.validacao.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${var.conta_scheduler_email}"
}

# Reconciliação das publicações pendentes (CE-14, ADR-0012).
resource "google_cloud_scheduler_job" "reconciliacao" {
  name             = "${var.prefixo}-reconciliar-publicacoes"
  project          = var.projeto_id
  region           = var.regiao
  description      = "Republica eventos de validação cuja publicação ficou pendente."
  schedule         = var.reconciliacao_cron
  time_zone        = "Etc/UTC"
  attempt_deadline = "60s"

  retry_config {
    retry_count = 1
  }

  http_target {
    uri         = "${google_cloudfunctions2_function.validacao.service_config[0].uri}/interno/reconciliar-publicacoes"
    http_method = "POST"

    oidc_token {
      service_account_email = var.conta_scheduler_email
      audience              = google_cloudfunctions2_function.validacao.service_config[0].uri
    }
  }
}
