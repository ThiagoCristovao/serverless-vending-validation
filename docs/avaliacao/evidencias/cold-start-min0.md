## Cold start — configuração min0 (RNF-06, CE-18)

Executado em 2026-10-30 09:23:18 -03 (duração 2h5m35s). Resultado: **critérios não atendidos** (ver conclusão).

### Parâmetros

| Parâmetro | Valor |
|---|---|
| projeto | svv-dev |
| gateway | https://svv-dev-gateway-4mjof8z9.ue.gateway.dev/v1 |
| configuração da função | min0 |
| amostras | 6 |
| inatividade antes de cada amostra | 20m0s |
| máquina | VM-E2E1 |
| agendador pausado | svv-dev-reconciliar-publicacoes |

### Linha do tempo

```text
09:23:20  job svv-dev-reconciliar-publicacoes pausado
09:23:20  amostra 1/6: aguardando 20m0s de inatividade
09:43:32  amostra 1: fria 4.995s, quente 5.802s (instâncias antes: 0 (09:41))
09:43:32  amostra 2/6: aguardando 20m0s de inatividade
10:03:40  amostra 2: fria 6.024s, quente 469ms (instâncias antes: 0 (10:02))
10:03:40  amostra 3/6: aguardando 20m0s de inatividade
10:23:52  amostra 3: fria 5.752s, quente 4.741s (instâncias antes: 0 (10:22))
10:23:52  amostra 4/6: aguardando 20m0s de inatividade
10:44:01  amostra 4: fria 7.205s, quente 603ms (instâncias antes: 0 (10:42))
10:44:01  amostra 5/6: aguardando 20m0s de inatividade
11:04:11  amostra 5: fria 7.262s, quente 761ms (instâncias antes: 0 (11:03))
11:04:11  amostra 6/6: aguardando 20m0s de inatividade
11:24:23  amostra 6: fria 6.972s, quente 3.18s (instâncias antes: 0 (11:23))
11:24:23  aguardando 4 min pela ingestão das métricas e dos logs
11:28:51  ❌ RNF-06: mediana da primeira requisição após inatividade = 6023 ms (alvo ≤ 1,5 s)
11:28:51  ❌ todas as primeiras requisições ≤ 1,5 s (6 de 6 acima)
11:28:53  job svv-dev-reconciliar-publicacoes retomado
```

### Amostras

| # | Hora | Instâncias antes (minuto) | 1ª requisição (cliente) | 2ª requisição (cliente) | 1ª no gateway (log, agente) | 2ª no gateway (log, agente) | 1ª na função (log) | 2ª na função (log) | Partidas de instância (média) |
|---|---|---|---|---|---|---|---|---|---|
| 1 | 09:43:21 | 0 (09:41) | 201 em 4.995s | 5.802s | 1.758s (ESPv2) | 3.52s (cloudesf) | 1.457s | 440ms | 1 (302 ms) |
| 2 | 10:03:33 | 0 (10:02) | 201 em 6.024s | 469ms | 2.807s (ESPv2) | 323ms (ESPv2) | 2.526s | 304ms | 1 (581 ms) |
| 3 | 10:23:41 | 0 (10:22) | 201 em 5.752s | 4.741s | 2.816s (ESPv2) | 3.428s (cloudesf) | 2.535s | 363ms | 1 (648 ms) |
| 4 | 10:43:53 | 0 (10:42) | 201 em 7.205s | 603ms | 5.962s (cloudesf) | 424ms (cloudesf) | 2.654s | 391ms | 1 (547 ms) |
| 5 | 11:04:03 | 0 (11:03) | 201 em 7.262s | 761ms | 5.981s (cloudesf) | 586ms (cloudesf) | 2.552s | 568ms | 1 (681 ms) |
| 6 | 11:24:13 | 0 (11:23) | 201 em 6.972s | 3.18s | 5.852s (cloudesf) | 303ms (ESPv2) | 2.478s | 278ms | 1 (382 ms) |

As colunas do gateway vêm do log de acesso `endpoints_log` (`resource.type="api"`): latência medida
pelo próprio gateway e agente que atendeu (`service_agent`). Nesta execução elas foram extraídas do
log depois do término, porque a versão da ferramenta que rodou trazia no lugar o máximo da métrica
`api/request_latencies` do Monitoring, cujos valores (4 152 e 8 305 ms) eram limites de faixa do
histograma e não latências reais; a ferramenta passou a registrar as colunas do log diretamente.

### Resumo das primeiras requisições

6 amostras: média 6368 ms, mediana 6023 ms, máximo 7262 ms.

### Conclusão

- ❌ RNF-06: mediana da primeira requisição após inatividade = 6023 ms (alvo ≤ 1,5 s)
- ❌ todas as primeiras requisições ≤ 1,5 s (6 de 6 acima)
