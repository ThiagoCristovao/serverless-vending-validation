# -----------------------------------------------------------------------------
# Firestore (modo nativo): banco, backups, regras de segurança e índices.
# Modelo de dados em docs/modelo-dados.md.
# -----------------------------------------------------------------------------

resource "google_firestore_database" "principal" {
  project     = var.projeto_id
  name        = "(default)"
  location_id = var.regiao
  type        = "FIRESTORE_NATIVE"

  concurrency_mode                  = "OPTIMISTIC"
  app_engine_integration_mode       = "DISABLED"
  point_in_time_recovery_enablement = "POINT_IN_TIME_RECOVERY_DISABLED"

  delete_protection_state = var.protecao_exclusao ? "DELETE_PROTECTION_ENABLED" : "DELETE_PROTECTION_DISABLED"
  deletion_policy         = "DELETE"
}

resource "google_firestore_backup_schedule" "diario" {
  count = var.backup_diario ? 1 : 0

  project   = var.projeto_id
  database  = google_firestore_database.principal.name
  retention = var.retencao_backup

  daily_recurrence {}
}

# Regras de segurança: negam todo acesso de cliente. O aplicativo nunca fala
# com o Firestore; só o serviço, via conta de serviço (que ignora as regras).
resource "google_firebaserules_ruleset" "firestore" {
  project = var.projeto_id

  source {
    files {
      name    = "firestore.rules"
      content = file("${path.module}/firestore.rules")
    }
  }

  lifecycle {
    create_before_destroy = true
  }

  depends_on = [google_firestore_database.principal]
}

resource "google_firebaserules_release" "firestore" {
  project      = var.projeto_id
  name         = "cloud.firestore"
  ruleset_name = "projects/${var.projeto_id}/rulesets/${google_firebaserules_ruleset.firestore.name}"

  lifecycle {
    replace_triggered_by = [google_firebaserules_ruleset.firestore]
  }
}

# Índices compostos (docs/modelo-dados.md, seção 3). Índices de campo único
# são automáticos.
locals {
  indices = {
    validacoes_por_maquina = {
      colecao = "validacoes"
      campos = [
        { campo = "idMaquina", ordem = "ASCENDING" },
        { campo = "iniciadaEm", ordem = "DESCENDING" },
      ]
    }
    validacoes_por_operador = {
      colecao = "validacoes"
      campos = [
        { campo = "idOperador", ordem = "ASCENDING" },
        { campo = "iniciadaEm", ordem = "DESCENDING" },
      ]
    }
    validacoes_expiracao = {
      colecao = "validacoes"
      campos = [
        { campo = "status", ordem = "ASCENDING" },
        { campo = "expiraEm", ordem = "ASCENDING" },
      ]
    }
  }
}

resource "google_firestore_index" "composto" {
  for_each = local.indices

  project     = var.projeto_id
  database    = google_firestore_database.principal.name
  collection  = each.value.colecao
  query_scope = "COLLECTION"

  dynamic "fields" {
    for_each = each.value.campos
    content {
      field_path = fields.value.campo
      order      = fields.value.ordem
    }
  }
}
