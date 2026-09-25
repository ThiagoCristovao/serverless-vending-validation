# Módulo `iam` — implementado na Sprint 3

Contas de serviço e papéis de projeto segundo o menor privilégio, mais o segredo da chave HMAC da
contrassenha (só o contêiner; o valor é criado fora do Terraform com `make segredo-hmac-gerar`).

| Conta | Papéis de projeto (aqui) | Papéis por recurso (em outros módulos) |
|---|---|---|
| `<prefixo>-funcao-validacao` | `datastore.user`, `logging.logWriter`, `monitoring.metricWriter`, `cloudtrace.agent` | `pubsub.publisher` no tópico (`pubsub`); `secretmanager.secretAccessor` no segredo (aqui) |
| `<prefixo>-gateway` | — | `run.invoker` na função (`funcao`, Sprint 5) |
| `<prefixo>-scheduler` | — | `run.invoker` na função, para o endpoint interno de reconciliação (`funcao`) |
| `<prefixo>-central-simulado` | — | `pubsub.subscriber` nas assinaturas (`pubsub`) |

| Entrada | Descrição | Padrão |
|---|---|---|
| `projeto_id`, `prefixo` | Projeto e prefixo | — |
| `nome_segredo_hmac` | Sufixo do nome do segredo | `chave-hmac-contrassenha` |

Saídas: e-mails das três contas, `segredo_hmac_nome`, `segredo_hmac_id`.

Nesta fase o Terraform roda com a identidade do desenvolvedor (ADC); uma conta de serviço de
implantação para CI fica como evolução.
