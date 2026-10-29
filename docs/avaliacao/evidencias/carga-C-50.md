## Carga — configuração C, 50 operadores simultâneos (RNF-05, RNF-06)

Executado em 2026-10-29 17:05:05 -03 (duração 6m45s). Resultado: **todos os critérios atendidos**.

### Parâmetros

| Parâmetro | Valor |
|---|---|
| projeto | svv-dev |
| gateway | https://svv-dev-gateway-4mjof8z9.ue.gateway.dev/v1 |
| configuração da função | C |
| operadores simultâneos | 50 |
| rampa | 30s |
| duração em regime | 2m0s |
| pausa entre validações | 1s |
| fluxo de cada operador | POST /validacoes (chave nova) → POST /validacoes/{id}/conclusao → pausa |

### Linha do tempo

```text
17:05:05  obtendo tokens de 50 operadores
17:05:11  rampa de 30s; regime até 17:07:41
17:07:41  carga encerrada
17:07:41  aguardando 4 min pela ingestão das métricas do Monitoring
17:11:50  ✅ nenhuma requisição com erro em regime (0 de 6913)
17:11:50  ✅ RNF-05: p95 no gateway com 50 operadores = 247 ms (alvo ≤ 500 ms)
17:11:50  cliente em regime: p50 363 ms, p95 407 ms, p99 454 ms; 57.6 req/s
```

### Latência medida no cliente (todas as requisições, inclusive a rampa)

| Requisição | N | Erros | Média | p50 | p95 | p99 | Máx. |
|---|---|---|---|---|---|---|---|
| todas | 7783 | 0 | 371 ms | 363 ms | 409 ms | 463 ms | 5547 ms |
| abertura | 3897 | 0 | 380 ms | 369 ms | 418 ms | 482 ms | 5547 ms |
| conclusão | 3886 | 0 | 361 ms | 357 ms | 398 ms | 454 ms | 893 ms |

### Latência medida no cliente em regime (2m0s após a rampa)

| Requisição | N | Erros | Média | p50 | p95 | p99 | Máx. |
|---|---|---|---|---|---|---|---|
| todas | 6913 | 0 | 367 ms | 363 ms | 407 ms | 454 ms | 739 ms |
| abertura | 3454 | 0 | 374 ms | 369 ms | 414 ms | 458 ms | 739 ms |
| conclusão | 3459 | 0 | 361 ms | 357 ms | 397 ms | 445 ms | 545 ms |

### Vazão em regime

6913 requisições em 2m0s: **57.6 req/s** (1728.2 validações/min).

### Linha do tempo no cliente (janelas de 10 s desde o início da rampa)

| Janela | N | Erros | p50 | p95 | Máx. |
|---|---|---|---|---|---|
| 0s–10s | 75 | 0 | 413 ms | 3388 ms | 5547 ms |
| 10s–20s | 304 | 0 | 365 ms | 421 ms | 498 ms |
| 20s–30s | 491 | 0 | 358 ms | 389 ms | 474 ms |
| 30s–40s | 577 | 0 | 360 ms | 393 ms | 478 ms |
| 40s–50s | 586 | 0 | 360 ms | 390 ms | 426 ms |
| 50s–60s | 572 | 0 | 366 ms | 402 ms | 554 ms |
| 60s–70s | 572 | 0 | 369 ms | 414 ms | 498 ms |
| 70s–80s | 576 | 0 | 363 ms | 401 ms | 495 ms |
| 80s–90s | 574 | 0 | 369 ms | 437 ms | 580 ms |
| 90s–100s | 575 | 0 | 360 ms | 392 ms | 470 ms |
| 100s–110s | 576 | 0 | 363 ms | 427 ms | 530 ms |
| 110s–120s | 582 | 0 | 362 ms | 397 ms | 739 ms |
| 120s–130s | 563 | 0 | 368 ms | 433 ms | 538 ms |
| 130s–140s | 583 | 0 | 360 ms | 402 ms | 547 ms |
| 140s–150s | 577 | 0 | 362 ms | 397 ms | 491 ms |

### Latência no provedor (Monitoring, janela da rampa ao fim)

| Medida | p50 | p95 | p99 |
|---|---|---|---|
| gateway, total (`serviceruntime…/request_latencies`) | 120 ms | 247 ms | 260 ms |
| gateway, parcela do back-end (`…request_latencies_backend`) | 119 ms | 246 ms | 260 ms |
| gateway, sobrecarga própria (`…request_latencies_overhead`) | 0 ms | 1 ms | 1 ms |
| função (`run.googleapis.com/request_latencies`) | 117 ms | 161 ms | 215 ms |

### Instâncias da função por minuto (ativas + ociosas, máximo no minuto)

| Minuto | Instâncias |
|---|---|
| 17:05 | 1 |
| 17:06 | 1 |
| 17:07 | 1 |

### p95 da função por minuto (Monitoring)

| Minuto | p95 |
|---|---|
| 17:06 | 159 ms |
| 17:07 | 162 ms |

### Escala e partidas

Requisições contadas pela função na janela: **5383**. Máximo de instâncias alocadas: **1**. Partidas de instância (`startup_latencies`): **0**, média **0 ms**.

### Conclusão

- ✅ nenhuma requisição com erro em regime (0 de 6913)
- ✅ RNF-05: p95 no gateway com 50 operadores = 247 ms (alvo ≤ 500 ms)
