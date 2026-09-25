# Módulo `funcao` — implementado na Sprint 5

Empacota `servico/`, envia ao Cloud Storage e implanta a Cloud Run function (2ª geração)
`<prefixo>-validacao` com runtime `go127`, ponto de entrada `ValidarMaquina`, variáveis de
ambiente do serviço e a chave HMAC montada do Secret Manager. Só o gateway e o Cloud Scheduler
recebem `roles/run.invoker`; nada é público. O job `<prefixo>-reconciliar-publicacoes` chama
`POST /interno/reconciliar-publicacoes` a cada 5 minutos com token OIDC (ADR-0012).

| Entrada | Descrição | Padrão |
|---|---|---|
| `projeto_id`, `regiao`, `prefixo` | Projeto, região e prefixo | — |
| `codigo_fonte_dir` | Diretório do módulo Go | — |
| `conta_funcao_email`, `conta_gateway_email`, `conta_scheduler_email` | Identidades | — |
| `topico_validacoes`, `segredo_hmac_nome` | Dependências de dados | — |
| `memoria`, `tempo_limite_segundos`, `max_instancias`, `min_instancias` | Dimensionamento | `256Mi`, `30`, `10`, `0` |
| `prazo_validacao`, `nivel_log`, `reconciliacao_cron` | Comportamento | `15m`, `info`, `*/5 * * * *` |

Saídas: `nome`, `url`, `objeto_fonte`.

Qualquer mudança no código gera um novo objeto (hash no nome) e uma nova revisão. O arquivo zip
fica em `.build/` (ignorado pelo git). Para a Sprint 10, `min_instancias = 1` é a alternativa a
medir contra o cenário de cold start.
