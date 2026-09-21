# Módulo `autenticacao` — Sprint 3

**Responsabilidade.** Firebase Authentication (via Identity Platform) e registro do aplicativo
Android no Firebase.

**Recursos previstos** (provider `google-beta`)

- `google_firebase_project` — habilita o Firebase no projeto GCP.
- `google_identity_platform_config` — ativa o Identity Platform e o provedor e-mail/senha
  (`sign_in { email { enabled = true, password_required = true } }`), desabilitando cadastro
  aberto (operadores são criados por administração).
- `google_firebase_android_app` — registro do app (`package_name`), com
  `google_firebase_android_app_config` (data source) para gerar o `google-services.json`, que
  **não** é versionado.
- `google_identity_platform_default_supported_idp_config` — não previsto (sem login social).

**Entradas previstas.** `projeto_id`, `pacote_android`.

**Saídas previstas.** `google_services_json` (sensível), `projeto_firebase`.

**Observação.** A primeira ativação do Firebase pode exigir aceite de termos no console; registrar
no ADR se acontecer, pois afeta o critério "sem intervenção manual" da Sprint 3.
