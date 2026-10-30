## Carga — configuração A, 100 operadores simultâneos (RNF-05, RNF-06)

Executado em 2026-10-29 16:07:14 -03 (duração 6m42s). Resultado: **critérios não atendidos** (ver conclusão).

### Parâmetros

| Parâmetro | Valor |
|---|---|
| projeto | svv-dev |
| gateway | https://svv-dev-gateway-4mjof8z9.ue.gateway.dev/v1 |
| configuração da função | A |
| operadores simultâneos | 100 |
| rampa | 30s |
| duração em regime | 2m0s |
| pausa entre validações | 1s |
| fluxo de cada operador | POST /validacoes (chave nova) → POST /validacoes/{id}/conclusao → pausa |

### Linha do tempo

```text
16:07:14  obtendo tokens de 100 operadores
16:07:18  rampa de 30s; regime até 16:09:48
16:09:48  carga encerrada
16:09:48  aguardando 4 min pela ingestão das métricas do Monitoring
16:13:56  ✅ nenhuma requisição com erro em regime (0 de 9393)
16:13:56  ❌ RNF-05: p95 no gateway com 100 operadores = 1834 ms (alvo ≤ 500 ms)
16:13:56  cliente em regime: p50 624 ms, p95 1592 ms, p99 1880 ms; 78.3 req/s
```

### Latência medida no cliente (todas as requisições, inclusive a rampa)

| Requisição | N | Erros | Média | p50 | p95 | p99 | Máx. |
|---|---|---|---|---|---|---|---|
| todas | 10940 | 1 | 734 ms | 566 ms | 1549 ms | 1871 ms | 2385 ms |
| abertura | 5485 | 1 | 743 ms | 573 ms | 1555 ms | 1860 ms | 2353 ms |
| conclusão | 5455 | 0 | 725 ms | 560 ms | 1543 ms | 1872 ms | 2385 ms |

### Latência medida no cliente em regime (2m0s após a rampa)

| Requisição | N | Erros | Média | p50 | p95 | p99 | Máx. |
|---|---|---|---|---|---|---|---|
| todas | 9393 | 0 | 778 ms | 624 ms | 1592 ms | 1880 ms | 2353 ms |
| abertura | 4688 | 0 | 788 ms | 637 ms | 1599 ms | 1890 ms | 2353 ms |
| conclusão | 4705 | 0 | 768 ms | 613 ms | 1585 ms | 1876 ms | 2209 ms |

### Vazão em regime

9393 requisições em 2m0s: **78.3 req/s** (2348.2 validações/min).

### Linha do tempo no cliente (janelas de 10 s desde o início da rampa)

| Janela | N | Erros | p50 | p95 | Máx. |
|---|---|---|---|---|---|
| 0s–10s | 217 | 0 | 292 ms | 486 ms | 916 ms |
| 10s–20s | 624 | 1 | 291 ms | 437 ms | 2129 ms |
| 20s–30s | 706 | 0 | 516 ms | 1434 ms | 2385 ms |
| 30s–40s | 718 | 0 | 670 ms | 1920 ms | 2353 ms |
| 40s–50s | 785 | 0 | 379 ms | 1903 ms | 2087 ms |
| 50s–60s | 832 | 0 | 564 ms | 1432 ms | 1720 ms |
| 60s–70s | 810 | 0 | 411 ms | 1630 ms | 1789 ms |
| 70s–80s | 819 | 0 | 550 ms | 1372 ms | 1580 ms |
| 80s–90s | 790 | 0 | 689 ms | 1434 ms | 1671 ms |
| 90s–100s | 785 | 0 | 665 ms | 1414 ms | 1650 ms |
| 100s–110s | 743 | 0 | 889 ms | 1323 ms | 1659 ms |
| 110s–120s | 759 | 0 | 820 ms | 1387 ms | 1504 ms |
| 120s–130s | 776 | 0 | 615 ms | 1461 ms | 1863 ms |
| 130s–140s | 792 | 0 | 772 ms | 1413 ms | 1879 ms |
| 140s–150s | 784 | 0 | 706 ms | 1440 ms | 1706 ms |

### Requisições com erro (até 20)

| Hora | Operador | Tipo | Status | Duração |
|---|---|---|---|---|
| 16:07:34.760 | 9 | abertura | 503 | 2129 ms |

### Latência no provedor (Monitoring, janela da rampa ao fim)

| Medida | p50 | p95 | p99 |
|---|---|---|---|
| gateway, total (`serviceruntime…/request_latencies`) | 386 ms | 1834 ms | 2046 ms |
| gateway, parcela do back-end (`…request_latencies_backend`) | 368 ms | 1837 ms | 2047 ms |
| gateway, sobrecarga própria (`…request_latencies_overhead`) | 0 ms | 1 ms | 3 ms |
| função (`run.googleapis.com/request_latencies`) | 119 ms | 188 ms | 235 ms |

### Instâncias da função por minuto (ativas + ociosas, máximo no minuto)

| Minuto | Instâncias |
|---|---|
| 16:07 | 9 |
| 16:08 | 9 |
| 16:09 | 10 |

### p95 da função por minuto (Monitoring)

| Minuto | p95 |
|---|---|
| 16:08 | 203 ms |
| 16:09 | 180 ms |

### Escala e partidas

Requisições contadas pela função na janela: **7220**. Máximo de instâncias alocadas: **10**. Partidas de instância (`startup_latencies`): **1**, média **49 ms**.

### Conclusão

- ✅ nenhuma requisição com erro em regime (0 de 9393)
- ❌ RNF-05: p95 no gateway com 100 operadores = 1834 ms (alvo ≤ 500 ms)
