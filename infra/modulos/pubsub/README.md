# Módulo `pubsub` — Sprint 3

**Responsabilidade.** Tópicos, assinaturas, fila de mensagens mortas e política de retentativa da
mensageria entre o serviço de validação e o sistema central.

**Recursos previstos**

- `google_pubsub_topic.validacoes` — `svv-<ambiente>-validacoes`, retenção de 7 dias, esquema opcional.
- `google_pubsub_topic.validacoes_dlq` — `svv-<ambiente>-validacoes-dlq`.
- `google_pubsub_subscription.sistema_central` — pull, `ack_deadline_seconds = 60`,
  `enable_message_ordering = true`, `retry_policy { minimum_backoff = 10s, maximum_backoff = 600s }`,
  `dead_letter_policy { max_delivery_attempts = 5 }`.
- `google_pubsub_subscription.dlq_inspecao` — pull sobre a DLQ, para inspeção manual e testes.
- `google_pubsub_topic_iam_member` / `google_pubsub_subscription_iam_member` — publicador (função) e
  assinante (sistema central simulado); o agente de serviço do Pub/Sub precisa de `publisher` na DLQ
  e `subscriber` na assinatura de origem.

**Entradas previstas.** `projeto_id`, `prefixo`, contas de serviço do módulo `iam`.

**Saídas previstas.** Nomes dos tópicos e assinaturas.

Detalhes de desenho em [docs/arquitetura.md](../../../docs/arquitetura.md), seção Mensageria.
