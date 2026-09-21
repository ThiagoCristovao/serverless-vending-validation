# Módulo `observabilidade` — Sprint 3 (base) e Sprint 5 (métricas do serviço)

**Responsabilidade.** Logs, métricas, painéis e alertas usados na avaliação técnica (Sprints 9 e 10).

**Recursos previstos**

- `google_logging_metric` — métricas baseadas em log: contagem por `codigo` de erro do serviço.
- `google_monitoring_dashboard` — painel com latência p50/p95/p99 e contagem de instâncias da
  função, backlog da assinatura (`subscription/num_undelivered_messages`), idade da mensagem mais
  antiga (`subscription/oldest_unacked_message_age`) e tamanho da DLQ.
- `google_monitoring_alert_policy` — backlog acima do limiar por 10 min; qualquer mensagem na DLQ.
- `google_monitoring_notification_channel` — e-mail do autor.
- Retenção de logs padrão (30 dias) é suficiente para o trabalho.

**Entradas previstas.** `projeto_id`, nomes da função e das assinaturas.
