# Módulo `pubsub` — implementado na Sprint 3

Tópico `<prefixo>-validacoes`, assinatura ordenada `<prefixo>-sistema-central` com retentativa
exponencial e fila de mensagens mortas `<prefixo>-validacoes-dlq` (com assinatura de inspeção).
Concede publicação à função e consumo ao sistema central simulado, além dos papéis que o agente de
serviço do Pub/Sub precisa para encaminhar mensagens mortas.

| Entrada | Descrição | Padrão |
|---|---|---|
| `projeto_id`, `prefixo` | Projeto e prefixo dos nomes | — |
| `conta_publicador_email` | Conta de serviço da função | — |
| `conta_assinante_email` | Conta de serviço do sistema central | — |
| `retencao_mensagens` | Retenção no tópico e nas assinaturas | `604800s` |
| `prazo_confirmacao_segundos` | `ack_deadline` | `60` |
| `espera_minima` / `espera_maxima` | Retentativa exponencial | `10s` / `600s` |
| `max_tentativas_entrega` | Tentativas antes da DLQ | `5` |

Saídas: nomes e IDs dos tópicos e assinaturas. Desenho em
[docs/arquitetura.md](../../../docs/arquitetura.md), seção 6.
