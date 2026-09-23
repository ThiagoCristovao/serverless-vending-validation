# -----------------------------------------------------------------------------
# Habilitações irreversíveis do projeto: Firebase e Identity Platform.
#
# A API não permite desfazer nenhuma das duas. Por isso elas ficam no bootstrap,
# que é aplicado uma vez por projeto e nunca destruído, e não no ambiente, que
# precisa passar por `destroy` + `apply` sem intervenção manual (Sprint 3).
#
# O projeto Firebase é criado pelo console (ADR-0010) e adotado pelo Terraform
# com o bloco import abaixo, idempotente após a primeira adoção.
# -----------------------------------------------------------------------------

provider "google-beta" {
  project = var.projeto_id
  region  = var.regiao

  user_project_override = true
  billing_project       = var.projeto_id
}

import {
  to = google_firebase_project.principal
  id = var.projeto_id
}

resource "google_firebase_project" "principal" {
  provider = google-beta

  project = var.projeto_id

  depends_on = [google_project_service.apis]
}

# Firebase Authentication via Identity Platform: e-mail e senha, sem anônimos.
resource "google_identity_platform_config" "principal" {
  project = var.projeto_id

  autodelete_anonymous_users = false

  sign_in {
    allow_duplicate_emails = false

    email {
      enabled           = true
      password_required = true
    }

    anonymous {
      enabled = false
    }
  }

  authorized_domains = concat(
    ["localhost", "${var.projeto_id}.firebaseapp.com", "${var.projeto_id}.web.app"],
    var.dominios_autorizados,
  )

  depends_on = [google_firebase_project.principal]
}
