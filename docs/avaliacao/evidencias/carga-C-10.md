## Carga — configuração C, 10 operadores simultâneos (RNF-05, RNF-06)

Executado em 2026-10-29 16:18:26 -03 (duração 6m38s). Resultado: **todos os critérios atendidos**.

### Parâmetros

| Parâmetro | Valor |
|---|---|
| projeto | svv-dev |
| gateway | https://svv-dev-gateway-4mjof8z9.ue.gateway.dev/v1 |
| configuração da função | C |
| operadores simultâneos | 10 |
| rampa | 30s |
| duração em regime | 2m0s |
| pausa entre validações | 1s |
| fluxo de cada operador | POST /validacoes (chave nova) → POST /validacoes/{id}/conclusao → pausa |

### Linha do tempo

```text
16:18:26  obtendo tokens de 10 operadores
16:18:26  rampa de 30s; regime até 16:20:56
16:20:56  carga encerrada
16:20:56  aguardando 4 min pela ingestão das métricas do Monitoring
16:25:04  ✅ nenhuma requisição com erro em regime (0 de 1400)
16:25:04  ✅ RNF-05: p95 no gateway com 10 operadores = 249 ms (alvo ≤ 500 ms)
16:25:04  ✅ RNF-06 (regime contínuo): função p50 = 115 ms, p95 = 161 ms (alvos ≤ 200 ms e ≤ 500 ms)
16:25:04  cliente em regime: p50 353 ms, p95 387 ms, p99 423 ms; 11.7 req/s
```

### Latência medida no cliente (todas as requisições, inclusive a rampa)

| Requisição | N | Erros | Média | p50 | p95 | p99 | Máx. |
|---|---|---|---|---|---|---|---|
| todas | 1586 | 0 | 361 ms | 354 ms | 398 ms | 475 ms | 2336 ms |
| abertura | 794 | 0 | 370 ms | 360 ms | 408 ms | 511 ms | 2336 ms |
| conclusão | 792 | 0 | 352 ms | 348 ms | 385 ms | 445 ms | 647 ms |

### Latência medida no cliente em regime (2m0s após a rampa)

| Requisição | N | Erros | Média | p50 | p95 | p99 | Máx. |
|---|---|---|---|---|---|---|---|
| todas | 1400 | 0 | 356 ms | 353 ms | 387 ms | 423 ms | 510 ms |
| abertura | 700 | 0 | 362 ms | 359 ms | 393 ms | 423 ms | 469 ms |
| conclusão | 700 | 0 | 349 ms | 346 ms | 376 ms | 414 ms | 510 ms |

### Vazão em regime

1400 requisições em 2m0s: **11.7 req/s** (350.0 validações/min).

### Linha do tempo no cliente (janelas de 10 s desde o início da rampa)

| Janela | N | Erros | p50 | p95 | Máx. |
|---|---|---|---|---|---|
| 0s–10s | 20 | 0 | 475 ms | 1499 ms | 2336 ms |
| 10s–20s | 63 | 0 | 381 ms | 440 ms | 539 ms |
| 20s–30s | 103 | 0 | 361 ms | 412 ms | 537 ms |
| 30s–40s | 116 | 0 | 356 ms | 391 ms | 406 ms |
| 40s–50s | 116 | 0 | 348 ms | 373 ms | 510 ms |
| 50s–60s | 116 | 0 | 346 ms | 375 ms | 423 ms |
| 60s–70s | 120 | 0 | 347 ms | 380 ms | 424 ms |
| 70s–80s | 118 | 0 | 352 ms | 376 ms | 408 ms |
| 80s–90s | 116 | 0 | 355 ms | 383 ms | 414 ms |
| 90s–100s | 115 | 0 | 355 ms | 396 ms | 426 ms |
| 100s–110s | 117 | 0 | 362 ms | 413 ms | 440 ms |
| 110s–120s | 114 | 0 | 357 ms | 395 ms | 469 ms |
| 120s–130s | 118 | 0 | 353 ms | 389 ms | 445 ms |
| 130s–140s | 116 | 0 | 352 ms | 384 ms | 448 ms |
| 140s–150s | 118 | 0 | 352 ms | 385 ms | 443 ms |

### Latência no provedor (Monitoring, janela da rampa ao fim)

| Medida | p50 | p95 | p99 |
|---|---|---|---|
| gateway, total (`serviceruntime…/request_latencies`) | 119 ms | 249 ms | 359 ms |
| gateway, parcela do back-end (`…request_latencies_backend`) | 116 ms | 247 ms | 310 ms |
| gateway, sobrecarga própria (`…request_latencies_overhead`) | 0 ms | 1 ms | 3 ms |
| função (`run.googleapis.com/request_latencies`) | 115 ms | 161 ms | 245 ms |

### Instâncias da função por minuto (ativas + ociosas, máximo no minuto)

| Minuto | Instâncias |
|---|---|
| 16:18 | 9 |
| 16:19 | 1 |
| 16:20 | 2 |

### p95 da função por minuto (Monitoring)

| Minuto | p95 |
|---|---|
| 16:19 | 227 ms |
| 16:20 | 142 ms |

### Escala e partidas

Requisições contadas pela função na janela: **937**. Máximo de instâncias alocadas: **9**. Partidas de instância (`startup_latencies`): **2**, média **363 ms**.

### Conclusão

- ✅ nenhuma requisição com erro em regime (0 de 1400)
- ✅ RNF-05: p95 no gateway com 10 operadores = 249 ms (alvo ≤ 500 ms)
- ✅ RNF-06 (regime contínuo): função p50 = 115 ms, p95 = 161 ms (alvos ≤ 200 ms e ≤ 500 ms)
