## Cold start — configuração aquecido (RNF-06, CE-18)

Executado em 2026-11-03 09:15:45 -03 (duração 2h4m39s). Resultado: **critérios não atendidos** (ver conclusão).

### Parâmetros

| Parâmetro | Valor |
|---|---|
| projeto | svv-dev |
| gateway | https://svv-dev-gateway-4mjof8z9.ue.gateway.dev/v1 |
| configuração da função | aquecido |
| amostras | 6 |
| inatividade antes de cada amostra | 20m0s |
| máquina | VM-E2E1 |
| agendador | jobs mantidos (reconciliação e aquecimento do gateway ativos) |

### Linha do tempo

```text
09:15:45  jobs do Scheduler mantidos durante a medição
09:15:45  amostra 1/6: aguardando 20m0s de inatividade
09:35:49  amostra 1: fria 2.191s, quente 494ms (instâncias antes: 1 (09:34))
09:35:49  amostra 2/6: aguardando 20m0s de inatividade
09:55:52  amostra 2: fria 1.078s, quente 514ms (instâncias antes: 1 (09:54))
09:55:52  amostra 3/6: aguardando 20m0s de inatividade
10:15:56  amostra 3: fria 2.078s, quente 543ms (instâncias antes: 1 (10:14))
10:15:56  amostra 4/6: aguardando 20m0s de inatividade
10:35:59  amostra 4: fria 1.135s, quente 445ms (instâncias antes: 1 (10:34))
10:35:59  amostra 5/6: aguardando 20m0s de inatividade
10:56:03  amostra 5: fria 2.059s, quente 525ms (instâncias antes: 1 (10:55))
10:56:03  amostra 6/6: aguardando 20m0s de inatividade
11:16:06  amostra 6: fria 1.209s, quente 615ms (instâncias antes: 1 (11:15))
11:16:06  aguardando 4 min pela ingestão das métricas e dos logs
11:20:24  ✅ RNF-06: mediana da primeira requisição após inatividade = 1208 ms (alvo ≤ 1,5 s)
11:20:24  ❌ todas as primeiras requisições ≤ 1,5 s (3 de 6 acima)
```

### Amostras

| # | Hora | Instâncias antes (minuto) | 1ª requisição (cliente) | 2ª requisição (cliente) | 1ª no gateway (log, agente) | 2ª no gateway (log, agente) | 1ª na função (log) | 2ª na função (log) | Partidas de instância (média) |
|---|---|---|---|---|---|---|---|---|---|
| 1 | 09:35:47 | 1 (09:34) | 201 em 2.191s | 494ms | 1.9s (ESPv2) | 350ms (cloudesf) | 1.875s | 323ms | 0 (0 ms) |
| 2 | 09:55:50 | 1 (09:54) | 201 em 1.078s | 514ms | 787ms (cloudesf) | 373ms (cloudesf) | 749ms | 346ms | 0 (0 ms) |
| 3 | 10:15:53 | 1 (10:14) | 201 em 2.078s | 543ms | 1.788s (ESPv2) | 406ms (cloudesf) | 1.762s | 370ms | 0 (0 ms) |
| 4 | 10:35:58 | 1 (10:34) | 201 em 1.135s | 445ms | 847ms (cloudesf) | 306ms (ESPv2) | 809ms | 293ms | 0 (0 ms) |
| 5 | 10:56:01 | 1 (10:55) | 201 em 2.059s | 525ms | 1.777s (cloudesf) | 386ms (cloudesf) | 1.729s | 359ms | 0 (0 ms) |
| 6 | 11:16:05 | 1 (11:15) | 201 em 1.209s | 615ms | 910ms (cloudesf) | 476ms (cloudesf) | 858ms | 456ms | 0 (0 ms) |

### Resumo das primeiras requisições

6 amostras: média 1625 ms, mediana 1208 ms, máximo 2191 ms.

### Conclusão

- ✅ RNF-06: mediana da primeira requisição após inatividade = 1208 ms (alvo ≤ 1,5 s)
- ❌ todas as primeiras requisições ≤ 1,5 s (3 de 6 acima)
