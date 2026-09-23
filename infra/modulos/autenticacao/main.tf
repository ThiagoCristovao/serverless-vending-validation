# -----------------------------------------------------------------------------
# Firebase Authentication (via Identity Platform) e registro do app Android.
#
# Observação: habilitar o Firebase e o Identity Platform em um projeto são ações
# que não podem ser desfeitas pela API. No `terraform destroy` esses dois
# recursos apenas saem do estado; o `apply` seguinte os reencontra.
# -----------------------------------------------------------------------------

resource "google_firebase_project" "principal" {
  provider = google-beta

  project = var.projeto_id
}

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

resource "google_firebase_android_app" "operador" {
  provider = google-beta

  project      = var.projeto_id
  display_name = var.nome_app_android
  package_name = var.pacote_android

  deletion_policy = "DELETE"

  depends_on = [google_firebase_project.principal]
}

# Conteúdo do google-services.json, consumido pelo aplicativo (Sprint 6).
# Nunca versionado: gerado a partir da saída do Terraform.
data "google_firebase_android_app_config" "operador" {
  provider = google-beta

  project = var.projeto_id
  app_id  = google_firebase_android_app.operador.app_id
}
