## Carga — configuração A, 50 operadores simultâneos (RNF-05, RNF-06)

Executado em 2026-10-29 16:00:12 -03 (duração 6m41s). Resultado: **todos os critérios atendidos**.

### Parâmetros

| Parâmetro | Valor |
|---|---|
| projeto | svv-dev |
| gateway | https://svv-dev-gateway-4mjof8z9.ue.gateway.dev/v1 |
| configuração da função | A |
| operadores simultâneos | 50 |
| rampa | 30s |
| duração em regime | 2m0s |
| pausa entre validações | 1s |
| fluxo de cada operador | POST /validacoes (chave nova) → POST /validacoes/{id}/conclusao → pausa |

### Linha do tempo

```text
16:00:12  obtendo tokens de 50 operadores
16:00:15  rampa de 30s; regime até 16:02:45
16:02:45  carga encerrada
16:02:45  aguardando 4 min pela ingestão das métricas do Monitoring
16:06:52  ✅ nenhuma requisição com erro em regime (0 de 7617)
16:06:52  ✅ RNF-05: p95 no gateway com 50 operadores = 256 ms (alvo ≤ 500 ms)
16:06:52  cliente em regime: p50 275 ms, p95 365 ms, p99 418 ms; 63.5 req/s
```

### Latência medida no cliente (todas as requisições, inclusive a rampa)

| Requisição | N | Erros | Média | p50 | p95 | p99 | Máx. |
|---|---|---|---|---|---|---|---|
| todas | 8587 | 0 | 289 ms | 277 ms | 367 ms | 426 ms | 3858 ms |
| abertura | 4298 | 0 | 295 ms | 282 ms | 378 ms | 443 ms | 863 ms |
| conclusão | 4289 | 0 | 283 ms | 270 ms | 356 ms | 410 ms | 3858 ms |

### Latência medida no cliente em regime (2m0s após a rampa)

| Requisição | N | Erros | Média | p50 | p95 | p99 | Máx. |
|---|---|---|---|---|---|---|---|
| todas | 7617 | 0 | 287 ms | 275 ms | 365 ms | 418 ms | 1456 ms |
| abertura | 3809 | 0 | 293 ms | 280 ms | 375 ms | 434 ms | 597 ms |
| conclusão | 3808 | 0 | 281 ms | 269 ms | 356 ms | 406 ms | 1456 ms |

### Vazão em regime

7617 requisições em 2m0s: **63.5 req/s** (1904.2 validações/min).

### Linha do tempo no cliente (janelas de 10 s desde o início da rampa)

| Janela | N | Erros | p50 | p95 | Máx. |
|---|---|---|---|---|---|
| 0s–10s | 108 | 0 | 310 ms | 498 ms | 3858 ms |
| 10s–20s | 328 | 0 | 285 ms | 364 ms | 471 ms |
| 20s–30s | 534 | 0 | 282 ms | 375 ms | 863 ms |
| 30s–40s | 624 | 0 | 288 ms | 384 ms | 528 ms |
| 40s–50s | 630 | 0 | 285 ms | 380 ms | 540 ms |
| 50s–60s | 624 | 0 | 280 ms | 378 ms | 521 ms |
| 60s–70s | 638 | 0 | 276 ms | 368 ms | 456 ms |
| 70s–80s | 645 | 0 | 269 ms | 347 ms | 436 ms |
| 80s–90s | 641 | 0 | 268 ms | 325 ms | 422 ms |
| 90s–100s | 644 | 0 | 269 ms | 328 ms | 412 ms |
| 100s–110s | 627 | 0 | 279 ms | 385 ms | 481 ms |
| 110s–120s | 639 | 0 | 272 ms | 352 ms | 498 ms |
| 120s–130s | 633 | 0 | 275 ms | 375 ms | 508 ms |
| 130s–140s | 630 | 0 | 275 ms | 378 ms | 1456 ms |
| 140s–150s | 642 | 0 | 272 ms | 355 ms | 498 ms |

### Latência no provedor (Monitoring, janela da rampa ao fim)

| Medida | p50 | p95 | p99 |
|---|---|---|---|
| gateway, total (`serviceruntime…/request_latencies`) | 157 ms | 256 ms | 410 ms |
| gateway, parcela do back-end (`…request_latencies_backend`) | 153 ms | 255 ms | 400 ms |
| gateway, sobrecarga própria (`…request_latencies_overhead`) | 0 ms | 1 ms | 1 ms |
| função (`run.googleapis.com/request_latencies`) | 121 ms | 190 ms | 248 ms |

### Instâncias da função por minuto (ativas + ociosas, máximo no minuto)

| Minuto | Instâncias |
|---|---|
| 16:00 | 7 |
| 16:01 | 8 |
| 16:02 | 10 |

### p95 da função por minuto (Monitoring)

| Minuto | p95 |
|---|---|
| 16:01 | 202 ms |
| 16:02 | 186 ms |

### Escala e partidas

Requisições contadas pela função na janela: **5713**. Máximo de instâncias alocadas: **10**. Partidas de instância (`startup_latencies`): **3**, média **69 ms**.

### Conclusão

- ✅ nenhuma requisição com erro em regime (0 de 7617)
- ✅ RNF-05: p95 no gateway com 50 operadores = 256 ms (alvo ≤ 500 ms)
