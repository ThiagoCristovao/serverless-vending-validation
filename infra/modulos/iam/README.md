# Módulo `iam` — Sprint 3

**Responsabilidade.** Contas de serviço e papéis segundo o princípio do menor privilégio.

**Contas de serviço previstas**

| Conta | Usada por | Papéis |
|---|---|---|
| `svv-<amb>-funcao-validacao` | Cloud Run function | `roles/datastore.user`, `roles/pubsub.publisher` (no tópico), `roles/secretmanager.secretAccessor` (no segredo da chave HMAC), `roles/logging.logWriter`, `roles/monitoring.metricWriter`, `roles/cloudtrace.agent` |
| `svv-<amb>-gateway` | API Gateway | `roles/run.invoker` na função (concedido no módulo `funcao`) |
| `svv-<amb>-central-simulado` | Sistema central simulado | `roles/pubsub.subscriber` na assinatura |

**Recursos previstos.** `google_service_account`, `google_project_iam_member` (apenas quando o papel
não puder ser concedido no recurso), `google_secret_manager_secret` da chave HMAC da contrassenha
(valor definido fora do Terraform).

**Saídas previstas.** E-mails das contas de serviço.

Nesta fase o Terraform roda com a identidade do desenvolvedor (ADC); uma conta de serviço de
implantação para CI fica como evolução.
