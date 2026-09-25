# -----------------------------------------------------------------------------
# API Gateway: valida o JWT do Firebase e roteia para a função.
# O contrato canônico api/openapi.yaml (3.1) é usado diretamente, com os
# marcadores PROJETO_ID e URL_DA_FUNCAO substituídos (ADR-0006).
# -----------------------------------------------------------------------------

locals {
  contrato = replace(
    replace(file(var.contrato_caminho), "URL_DA_FUNCAO", "${var.url_funcao}${var.caminho_base_backend}"),
    "PROJETO_ID", var.projeto_id,
  )
}

resource "google_api_gateway_api" "api" {
  provider = google-beta

  project      = var.projeto_id
  api_id       = "${var.prefixo}-api"
  display_name = "Serviço de validação de máquinas (${var.prefixo})"
}

resource "google_api_gateway_api_config" "config" {
  provider = google-beta

  project              = var.projeto_id
  api                  = google_api_gateway_api.api.api_id
  api_config_id_prefix = "${var.prefixo}-cfg-"
  display_name         = "Contrato ${var.prefixo}"

  openapi_documents {
    document {
      path     = "openapi.yaml"
      contents = base64encode(local.contrato)
    }
  }

  gateway_config {
    backend_config {
      google_service_account = var.conta_gateway_email
    }
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "google_api_gateway_gateway" "gateway" {
  provider = google-beta

  project      = var.projeto_id
  region       = var.regiao
  gateway_id   = "${var.prefixo}-gateway"
  api_config   = google_api_gateway_api_config.config.id
  display_name = "Gateway ${var.prefixo}"
}
