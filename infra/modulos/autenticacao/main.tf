# -----------------------------------------------------------------------------
# Registro do aplicativo Android no Firebase e conteúdo do google-services.json.
#
# Firebase e Identity Platform (habilitações irreversíveis) são geridos no
# bootstrap; aqui fica só o que pode ser destruído e recriado livremente.
# -----------------------------------------------------------------------------

resource "google_firebase_android_app" "operador" {
  provider = google-beta

  project      = var.projeto_id
  display_name = var.nome_app_android
  package_name = var.pacote_android

  deletion_policy = "DELETE"
}

# Consumido pelo aplicativo (Sprint 6). Nunca versionado: gerado a partir da
# saída do Terraform com `make app-google-services`.
data "google_firebase_android_app_config" "operador" {
  provider = google-beta

  project = var.projeto_id
  app_id  = google_firebase_android_app.operador.app_id
}
