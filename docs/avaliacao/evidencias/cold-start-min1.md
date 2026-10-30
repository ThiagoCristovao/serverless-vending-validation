## Cold start — configuração min1 (RNF-06, CE-18)

Executado em 2026-10-30 11:34:03 -03 (duração 2h5m23s). Resultado: **critérios não atendidos** (ver conclusão).

### Parâmetros

| Parâmetro | Valor |
|---|---|
| projeto | svv-dev |
| gateway | https://svv-dev-gateway-4mjof8z9.ue.gateway.dev/v1 |
| configuração da função | min1 |
| amostras | 6 |
| inatividade antes de cada amostra | 20m0s |
| máquina | VM-E2E1 |
| agendador pausado | svv-dev-reconciliar-publicacoes |

### Linha do tempo

```text
11:34:05  job svv-dev-reconciliar-publicacoes pausado
11:34:05  amostra 1/6: aguardando 20m0s de inatividade
11:54:15  amostra 1: fria 8.04s, quente 505ms (instâncias antes: 1 (11:53))
11:54:15  amostra 2/6: aguardando 20m0s de inatividade
12:14:22  amostra 2: fria 6.182s, quente 580ms (instâncias antes: 1 (12:13))
12:14:22  amostra 3/6: aguardando 20m0s de inatividade
12:34:33  amostra 3: fria 5.132s, quente 4.619s (instâncias antes: 1 (12:33))
12:34:33  amostra 4/6: aguardando 20m0s de inatividade
12:54:42  amostra 4: fria 6.07s, quente 471ms (instâncias antes: 1 (12:53))
12:54:42  amostra 5/6: aguardando 20m0s de inatividade
13:14:53  amostra 5: fria 6.389s, quente 3.398s (instâncias antes: 1 (13:13))
13:14:53  amostra 6/6: aguardando 20m0s de inatividade
13:35:04  amostra 6: fria 6.362s, quente 3.462s (instâncias antes: 1 (13:33))
13:35:04  aguardando 4 min pela ingestão das métricas e dos logs
13:39:23  ❌ RNF-06: mediana da primeira requisição após inatividade = 6182 ms (alvo ≤ 1,5 s)
13:39:23  ❌ todas as primeiras requisições ≤ 1,5 s (6 de 6 acima)
13:39:26  job svv-dev-reconciliar-publicacoes retomado
```

### Amostras

| # | Hora | Instâncias antes (minuto) | 1ª requisição (cliente) | 2ª requisição (cliente) | 1ª no gateway (log, agente) | 2ª no gateway (log, agente) | 1ª na função (log) | 2ª na função (log) | Partidas de instância (média) |
|---|---|---|---|---|---|---|---|---|---|
| 1 | 11:54:06 | 1 (11:53) | 201 em 8.04s | 505ms | 6.631s (cloudesf) | 366ms (cloudesf) | 3.557s | 353ms | 0 (0 ms) |
| 2 | 12:14:16 | 1 (12:13) | 201 em 6.182s | 580ms | 4.665s (cloudesf) | 423ms (cloudesf) | 1.608s | 396ms | 0 (0 ms) |
| 3 | 12:34:24 | 1 (12:33) | 201 em 5.132s | 4.619s | 1.871s (ESPv2) | 3.484s (cloudesf) | 1.828s | 392ms | 0 (0 ms) |
| 4 | 12:54:35 | 1 (12:53) | 201 em 6.07s | 471ms | 4.728s (cloudesf) | 329ms (cloudesf) | 1.639s | 316ms | 0 (0 ms) |
| 5 | 13:14:43 | 1 (13:13) | 201 em 6.389s | 3.398s | 4.927s (cloudesf) | 379ms (ESPv2) | 1.877s | 357ms | 0 (0 ms) |
| 6 | 13:34:54 | 1 (13:33) | 201 em 6.362s | 3.462s | 4.955s (cloudesf) | 362ms (ESPv2) | 1.864s | 323ms | 0 (0 ms) |

As colunas do gateway vêm do log de acesso `endpoints_log` (`resource.type="api"`): latência medida
pelo próprio gateway e agente que atendeu (`service_agent`). Nesta execução elas foram extraídas do
log depois do término, porque a versão da ferramenta que rodou trazia no lugar o máximo da métrica
`api/request_latencies` do Monitoring, cujos valores (2 076, 4 152 e 8 305 ms) eram limites de
faixa do histograma e não latências reais; a ferramenta passou a registrar as colunas do log.

### Resumo das primeiras requisições

6 amostras: média 6362 ms, mediana 6182 ms, máximo 8039 ms.

### Conclusão

- ❌ RNF-06: mediana da primeira requisição após inatividade = 6182 ms (alvo ≤ 1,5 s)
- ❌ todas as primeiras requisições ≤ 1,5 s (6 de 6 acima)
