# Módulo `observabilidade` — implementado na Sprint 3

Canal de notificação por e-mail, métrica derivada de log (erros do serviço por `codigo`), duas
políticas de alerta e o painel usado na avaliação técnica.

| Alerta | Condição | Cenário |
|---|---|---|
| Mensagens na DLQ | `num_undelivered_messages` > 0 por 5 min na assinatura de inspeção | CE-16 |
| Backlog no sistema central | `num_undelivered_messages` > `limiar_backlog` por 10 min | CE-15 |

Painel: latência p50/p95/p99 e instâncias da função (métricas do Cloud Run), mensagens não entregues
nas duas assinaturas, idade da mensagem mais antiga e erros por código.

| Entrada | Descrição | Padrão |
|---|---|---|
| `projeto_id`, `prefixo` | Projeto e prefixo | — |
| `email_alertas` | Destinatário dos alertas | — |
| `nome_funcao` | Serviço Cloud Run da função (filtros) | — |
| `assinatura_central_nome`, `assinatura_dlq_nome` | Assinaturas monitoradas | — |
| `limiar_backlog` | Limiar do alerta de backlog | `100` |

Os gráficos da função ficam vazios até a Sprint 5, quando a função passa a existir.
