## Cenário A — sistema central indisponível durante validações (CE-15, RNF-01, RNF-03)

Executado em 2026-10-27 14:58:56 -03 (duração 5m21s). Resultado: **todos os critérios atendidos**.

### Parâmetros

| Parâmetro | Valor |
|---|---|
| projeto | svv-dev |
| gateway | https://svv-dev-gateway-4mjof8z9.ue.gateway.dev/v1 |
| validações durante a falha | 6 |
| registro do consumidor | /tmp/svv-laboratorio-762341102/eventos.jsonl |
| assinatura | svv-dev-sistema-central |
| duração da indisponibilidade | 3m0s |

### Linha do tempo

```text
14:58:57  consumidor iniciado; aguardando 10 s para esvaziar a assinatura
14:59:07  eventos antigos consumidos: 0
14:59:07  sistema central PAUSADO (indisponível)
14:59:20  6 validações executadas durante a indisponibilidade; 6 completas sem erro
14:59:51  backlog (num_undelivered_messages) = 0 em 14:59:00
15:00:22  backlog (num_undelivered_messages) = 0 em 14:59:00
15:00:52  backlog (num_undelivered_messages) = 6 em 15:00:00
15:01:23  backlog (num_undelivered_messages) = 6 em 15:00:00
15:01:53  backlog (num_undelivered_messages) = 12 em 15:01:00
15:02:08  backlog (num_undelivered_messages) = 12 em 15:01:00
15:02:08  sistema central RETOMADO após 3m0.651s
15:02:17  12 de 12 eventos das validações da falha consumidos 9.005s após a retomada
15:02:17  coletando métricas do Monitoring por 2 min após a retomada
15:04:17  ✅ RNF-03: 6/6 validações completas (201/200) com o sistema central indisponível
15:04:17  ✅ RNF-01: 12/12 eventos consumidos após o restabelecimento, sem intervenção manual, em 9.005s
15:04:17  ✅ ordem por máquina preservada (iniciada antes de concluida em cada validação)
15:04:17  ✅ nenhuma duplicata ou rejeição durante a recuperação (duplicadas=0, rejeitadas=0)
```

### Validações durante a indisponibilidade

| # | Máquina | idValidacao | Abertura | Tempo | Conclusão | Tempo | Erro |
|---|---|---|---|---|---|---|---|
| 1 | VM-E2E1 | 01M5Z889K4WW5NEKN12MA4JJ98 | 201 | 5.512s | 200 | 586ms | — |
| 2 | VM-E2E2 | 01M5Z88DYRNATVQ8TN31T5ANP2 | 201 | 3.609s | 200 | 513ms | — |
| 3 | VM-E2E1 | 01M5Z88EX827N5EBXW8AMJQTHY | 201 | 381ms | 200 | 331ms | — |
| 4 | VM-E2E2 | 01M5Z88FM1K0AP5FBQGM61FN99 | 201 | 382ms | 200 | 398ms | — |
| 5 | VM-E2E1 | 01M5Z88GBAJGDPHQXDQWCA7N8S | 201 | 317ms | 200 | 335ms | — |
| 6 | VM-E2E2 | 01M5Z88GZ6TFK41E3D1BJP95VJ | 201 | 358ms | 200 | 301ms | — |

### Métrica `subscription/num_undelivered_messages` (Cloud Monitoring)

| Instante | Mensagens não entregues |
|---|---|
| 14:58:00 | 0 |
| 14:59:00 | 0 |
| 15:00:00 | 6 |
| 15:01:00 | 12 |
| 15:02:00 | 12 |
| 15:03:00 | 6 |

### Métrica `subscription/oldest_unacked_message_age` (segundos)

| Instante | Idade (s) |
|---|---|
| 14:58:00 | 0 |
| 14:59:00 | 0 |
| 15:00:00 | 10 |
| 15:01:00 | 70 |
| 15:02:00 | 130 |
| 15:03:00 | 163 |

### Conclusão

- ✅ RNF-03: 6/6 validações completas (201/200) com o sistema central indisponível
- ✅ RNF-01: 12/12 eventos consumidos após o restabelecimento, sem intervenção manual, em 9.005s
- ✅ ordem por máquina preservada (iniciada antes de concluida em cada validação)
- ✅ nenhuma duplicata ou rejeição durante a recuperação (duplicadas=0, rejeitadas=0)
