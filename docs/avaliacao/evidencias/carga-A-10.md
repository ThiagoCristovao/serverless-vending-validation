## Carga — configuração A, 10 operadores simultâneos (RNF-05, RNF-06)

Executado em 2026-10-29 15:53:12 -03 (duração 6m39s). Resultado: **todos os critérios atendidos**.

### Parâmetros

| Parâmetro | Valor |
|---|---|
| projeto | svv-dev |
| gateway | https://svv-dev-gateway-4mjof8z9.ue.gateway.dev/v1 |
| configuração da função | A |
| operadores simultâneos | 10 |
| rampa | 30s |
| duração em regime | 2m0s |
| pausa entre validações | 1s |
| fluxo de cada operador | POST /validacoes (chave nova) → POST /validacoes/{id}/conclusao → pausa |

### Linha do tempo

```text
15:53:12  obtendo tokens de 10 operadores
15:53:12  rampa de 30s; regime até 15:55:42
15:55:42  carga encerrada
15:55:42  aguardando 4 min pela ingestão das métricas do Monitoring
15:59:51  ✅ nenhuma requisição com erro em regime (0 de 1504)
15:59:51  ✅ RNF-05: p95 no gateway com 10 operadores = 304 ms (alvo ≤ 500 ms)
15:59:51  ✅ RNF-06 (regime contínuo): função p50 = 136 ms, p95 = 250 ms (alvos ≤ 200 ms e ≤ 500 ms)
15:59:51  cliente em regime: p50 278 ms, p95 379 ms, p99 442 ms; 12.5 req/s
```

### Latência medida no cliente (todas as requisições, inclusive a rampa)

| Requisição | N | Erros | Média | p50 | p95 | p99 | Máx. |
|---|---|---|---|---|---|---|---|
| todas | 1704 | 0 | 303 ms | 281 ms | 394 ms | 511 ms | 4206 ms |
| abertura | 856 | 0 | 315 ms | 299 ms | 409 ms | 522 ms | 1722 ms |
| conclusão | 848 | 0 | 289 ms | 270 ms | 361 ms | 474 ms | 4206 ms |

### Latência medida no cliente em regime (2m0s após a rampa)

| Requisição | N | Erros | Média | p50 | p95 | p99 | Máx. |
|---|---|---|---|---|---|---|---|
| todas | 1504 | 0 | 298 ms | 278 ms | 379 ms | 442 ms | 4206 ms |
| abertura | 756 | 0 | 310 ms | 295 ms | 394 ms | 442 ms | 1722 ms |
| conclusão | 748 | 0 | 286 ms | 268 ms | 340 ms | 446 ms | 4206 ms |

### Vazão em regime

1504 requisições em 2m0s: **12.5 req/s** (376.0 validações/min).

### Linha do tempo no cliente (janelas de 10 s desde o início da rampa)

| Janela | N | Erros | p50 | p95 | Máx. |
|---|---|---|---|---|---|
| 0s–10s | 24 | 0 | 372 ms | 605 ms | 747 ms |
| 10s–20s | 65 | 0 | 325 ms | 511 ms | 1428 ms |
| 20s–30s | 111 | 0 | 291 ms | 433 ms | 525 ms |
| 30s–40s | 121 | 0 | 279 ms | 399 ms | 1722 ms |
| 40s–50s | 126 | 0 | 291 ms | 397 ms | 608 ms |
| 50s–60s | 124 | 0 | 301 ms | 409 ms | 1385 ms |
| 60s–70s | 126 | 0 | 281 ms | 348 ms | 400 ms |
| 70s–80s | 122 | 0 | 277 ms | 351 ms | 397 ms |
| 80s–90s | 133 | 0 | 276 ms | 385 ms | 559 ms |
| 90s–100s | 126 | 0 | 273 ms | 372 ms | 417 ms |
| 100s–110s | 120 | 0 | 279 ms | 386 ms | 1108 ms |
| 110s–120s | 127 | 0 | 286 ms | 379 ms | 464 ms |
| 120s–130s | 132 | 0 | 270 ms | 347 ms | 472 ms |
| 130s–140s | 115 | 0 | 272 ms | 383 ms | 4206 ms |
| 140s–150s | 132 | 0 | 273 ms | 347 ms | 841 ms |

### Latência no provedor (Monitoring, janela da rampa ao fim)

| Medida | p50 | p95 | p99 |
|---|---|---|---|
| gateway, total (`serviceruntime…/request_latencies`) | 176 ms | 304 ms | 504 ms |
| gateway, parcela do back-end (`…request_latencies_backend`) | 177 ms | 280 ms | 497 ms |
| gateway, sobrecarga própria (`…request_latencies_overhead`) | 0 ms | 1 ms | 3 ms |
| função (`run.googleapis.com/request_latencies`) | 136 ms | 250 ms | 386 ms |

### Instâncias da função por minuto (ativas + ociosas, máximo no minuto)

| Minuto | Instâncias |
|---|---|
| 15:53 | 3 |
| 15:54 | 3 |
| 15:55 | 9 |

### p95 da função por minuto (Monitoring)

| Minuto | p95 |
|---|---|
| 15:54 | 282 ms |
| 15:55 | 229 ms |

### Escala e partidas

Requisições contadas pela função na janela: **1137**. Máximo de instâncias alocadas: **9**. Partidas de instância (`startup_latencies`): **6**, média **249 ms**.

### Conclusão

- ✅ nenhuma requisição com erro em regime (0 de 1504)
- ✅ RNF-05: p95 no gateway com 10 operadores = 304 ms (alvo ≤ 500 ms)
- ✅ RNF-06 (regime contínuo): função p50 = 136 ms, p95 = 250 ms (alvos ≤ 200 ms e ≤ 500 ms)
