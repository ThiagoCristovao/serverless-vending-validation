# -----------------------------------------------------------------------------
# Observabilidade: canal de notificação, métrica derivada de log, alertas e
# painel usados na avaliação técnica (Sprints 9 e 10).
# docs/arquitetura.md, seção 10.
# -----------------------------------------------------------------------------

resource "google_monitoring_notification_channel" "email" {
  project      = var.projeto_id
  display_name = "${var.prefixo}: e-mail do autor"
  type         = "email"

  labels = {
    email_address = var.email_alertas
  }
}

# Contagem de erros do serviço por código de domínio (jsonPayload.codigo).
resource "google_logging_metric" "erros_por_codigo" {
  project     = var.projeto_id
  name        = "${var.prefixo}-erros-por-codigo"
  description = "Respostas de erro do serviço de validação, por código de domínio (Problem Details)."
  filter      = "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"${var.nome_funcao}\" AND jsonPayload.codigo:*"

  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"

    labels {
      key         = "codigo"
      value_type  = "STRING"
      description = "Código de erro do domínio"
    }
  }

  label_extractors = {
    codigo = "EXTRACT(jsonPayload.codigo)"
  }
}

locals {
  filtro_metrica_assinatura = "resource.type = \"pubsub_subscription\" AND metric.type = \"pubsub.googleapis.com/subscription/num_undelivered_messages\""
}

# Qualquer mensagem na DLQ indica falha repetida de processamento (CE-16).
resource "google_monitoring_alert_policy" "dlq" {
  project      = var.projeto_id
  display_name = "${var.prefixo}: mensagens na fila de mensagens mortas"
  combiner     = "OR"

  conditions {
    display_name = "DLQ com mensagens não entregues"

    condition_threshold {
      filter          = "${local.filtro_metrica_assinatura} AND resource.labels.subscription_id = \"${var.assinatura_dlq_nome}\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0
      duration        = "300s"

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MAX"
      }

      trigger {
        count = 1
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.email.id]

  alert_strategy {
    auto_close = "1800s"
  }

  documentation {
    mime_type = "text/markdown"
    content   = "Mensagens chegaram à DLQ `${var.assinatura_dlq_nome}` após esgotar as tentativas de entrega. Inspecione com `gcloud pubsub subscriptions pull` e verifique o consumidor do sistema central."
  }
}

# Backlog crescente na assinatura principal: consumidor lento ou indisponível (CE-15).
resource "google_monitoring_alert_policy" "backlog" {
  project      = var.projeto_id
  display_name = "${var.prefixo}: backlog na assinatura do sistema central"
  combiner     = "OR"

  conditions {
    display_name = "Mensagens não entregues acima do limiar por 10 minutos"

    condition_threshold {
      filter          = "${local.filtro_metrica_assinatura} AND resource.labels.subscription_id = \"${var.assinatura_central_nome}\""
      comparison      = "COMPARISON_GT"
      threshold_value = var.limiar_backlog
      duration        = "600s"

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MAX"
      }

      trigger {
        count = 1
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.email.id]

  alert_strategy {
    auto_close = "1800s"
  }

  documentation {
    mime_type = "text/markdown"
    content   = "A assinatura `${var.assinatura_central_nome}` acumula mensagens. Esperado durante os cenários de indisponibilidade deliberada (Sprint 9); fora deles, verifique o consumidor."
  }
}

# Painel da avaliação técnica.
locals {
  filtro_funcao = "resource.type=\"cloud_run_revision\" resource.label.\"service_name\"=\"${var.nome_funcao}\""

  grafico_latencia = [
    for p in ["50", "95", "99"] : {
      plotType       = "LINE"
      legendTemplate = "p${p}"
      timeSeriesQuery = {
        timeSeriesFilter = {
          filter = "metric.type=\"run.googleapis.com/request_latencies\" ${local.filtro_funcao}"
          aggregation = {
            alignmentPeriod    = "60s"
            perSeriesAligner   = "ALIGN_PERCENTILE_${p}"
            crossSeriesReducer = "REDUCE_MEAN"
          }
        }
      }
    }
  ]

  grafico_assinaturas = [
    for nome in [var.assinatura_central_nome, var.assinatura_dlq_nome] : {
      plotType       = "LINE"
      legendTemplate = nome
      timeSeriesQuery = {
        timeSeriesFilter = {
          filter = "metric.type=\"pubsub.googleapis.com/subscription/num_undelivered_messages\" resource.type=\"pubsub_subscription\" resource.label.\"subscription_id\"=\"${nome}\""
          aggregation = {
            alignmentPeriod  = "60s"
            perSeriesAligner = "ALIGN_MAX"
          }
        }
      }
    }
  ]
}

resource "google_monitoring_dashboard" "principal" {
  project = var.projeto_id

  dashboard_json = jsonencode({
    displayName = "${var.prefixo}: validação"
    mosaicLayout = {
      columns = 12
      tiles = [
        {
          xPos = 0, yPos = 0, width = 6, height = 4
          widget = {
            title   = "Latência da função (p50, p95, p99)"
            xyChart = { dataSets = local.grafico_latencia, yAxis = { scale = "LINEAR" } }
          }
        },
        {
          xPos = 6, yPos = 0, width = 6, height = 4
          widget = {
            title = "Instâncias da função"
            xyChart = {
              dataSets = [{
                plotType = "LINE"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "metric.type=\"run.googleapis.com/container/instance_count\" ${local.filtro_funcao}"
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_MAX"
                      crossSeriesReducer = "REDUCE_SUM"
                    }
                  }
                }
              }]
              yAxis = { scale = "LINEAR" }
            }
          }
        },
        {
          xPos = 0, yPos = 4, width = 6, height = 4
          widget = {
            title   = "Mensagens não entregues (principal e DLQ)"
            xyChart = { dataSets = local.grafico_assinaturas, yAxis = { scale = "LINEAR" } }
          }
        },
        {
          xPos = 6, yPos = 4, width = 6, height = 4
          widget = {
            title = "Idade da mensagem mais antiga sem confirmação (s)"
            xyChart = {
              dataSets = [{
                plotType = "LINE"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "metric.type=\"pubsub.googleapis.com/subscription/oldest_unacked_message_age\" resource.type=\"pubsub_subscription\" resource.label.\"subscription_id\"=\"${var.assinatura_central_nome}\""
                    aggregation = {
                      alignmentPeriod  = "60s"
                      perSeriesAligner = "ALIGN_MAX"
                    }
                  }
                }
              }]
              yAxis = { scale = "LINEAR" }
            }
          }
        },
        {
          xPos = 0, yPos = 8, width = 12, height = 4
          widget = {
            title = "Erros do serviço por código"
            xyChart = {
              dataSets = [{
                plotType = "STACKED_BAR"
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "metric.type=\"logging.googleapis.com/user/${google_logging_metric.erros_por_codigo.name}\" resource.type=\"cloud_run_revision\""
                    aggregation = {
                      alignmentPeriod    = "60s"
                      perSeriesAligner   = "ALIGN_SUM"
                      crossSeriesReducer = "REDUCE_SUM"
                      groupByFields      = ["metric.label.\"codigo\""]
                    }
                  }
                }
              }]
              yAxis = { scale = "LINEAR" }
            }
          }
        },
      ]
    }
  })
}
