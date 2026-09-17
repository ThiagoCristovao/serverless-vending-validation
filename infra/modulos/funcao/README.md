# Módulo `funcao` — Sprint 5

**Responsabilidade.** Implantação do serviço de validação como Cloud Run function (2ª geração).

**Recursos previstos**

- `archive_file` (data) + `google_storage_bucket` + `google_storage_bucket_object` — empacota
  `servico/` e envia o código-fonte.
- `google_cloudfunctions2_function` — `svv-<amb>-validacao`, `runtime = "go127"`,
  `entry_point = "ValidarMaquina"`, `service_config { available_memory = "256Mi",
  timeout_seconds = 30, max_instance_count = <limite de custo>, min_instance_count = 0,
  ingress_settings = "ALLOW_ALL", service_account_email = <svv-funcao-validacao> }`, variáveis
  de ambiente e `secret_environment_variables` para a chave HMAC.
- `google_cloud_run_service_iam_member` — `roles/run.invoker` **somente** para a conta de serviço
  do gateway (nunca `allUsers`).

**Entradas previstas.** `projeto_id`, `regiao`, `prefixo`, contas de serviço, nome do tópico.

**Saídas previstas.** `url` da função (usada em `x-google-backend.address` e `jwt_audience`).

Para a Sprint 10, `min_instance_count = 1` é uma alternativa a medir contra o cenário de cold start.
