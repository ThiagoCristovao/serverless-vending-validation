## Carga — configuração C, 100 operadores simultâneos (RNF-05, RNF-06)

Executado em 2026-10-29 17:12:05 -03 (duração 6m42s). Resultado: **todos os critérios atendidos**.

### Parâmetros

| Parâmetro | Valor |
|---|---|
| projeto | svv-dev |
| gateway | https://svv-dev-gateway-4mjof8z9.ue.gateway.dev/v1 |
| configuração da função | C |
| operadores simultâneos | 100 |
| rampa | 30s |
| duração em regime | 2m0s |
| pausa entre validações | 1s |
| fluxo de cada operador | POST /validacoes (chave nova) → POST /validacoes/{id}/conclusao → pausa |

### Linha do tempo

```text
17:12:05  obtendo tokens de 100 operadores
17:12:10  rampa de 30s; regime até 17:14:40
17:14:40  carga encerrada
17:14:40  aguardando 4 min pela ingestão das métricas do Monitoring
17:18:48  ✅ nenhuma requisição com erro em regime (0 de 14468)
17:18:48  ✅ RNF-05: p95 no gateway com 100 operadores = 452 ms (alvo ≤ 500 ms)
17:18:48  cliente em regime: p50 305 ms, p95 467 ms, p99 543 ms; 120.6 req/s
```

### Latência medida no cliente (todas as requisições, inclusive a rampa)

| Requisição | N | Erros | Média | p50 | p95 | p99 | Máx. |
|---|---|---|---|---|---|---|---|
| todas | 16326 | 0 | 329 ms | 306 ms | 462 ms | 536 ms | 903 ms |
| abertura | 8169 | 0 | 337 ms | 311 ms | 480 ms | 556 ms | 903 ms |
| conclusão | 8157 | 0 | 320 ms | 298 ms | 443 ms | 502 ms | 642 ms |

### Latência medida no cliente em regime (2m0s após a rampa)

| Requisição | N | Erros | Média | p50 | p95 | p99 | Máx. |
|---|---|---|---|---|---|---|---|
| todas | 14468 | 0 | 329 ms | 305 ms | 467 ms | 543 ms | 710 ms |
| abertura | 7228 | 0 | 338 ms | 311 ms | 486 ms | 563 ms | 710 ms |
| conclusão | 7240 | 0 | 320 ms | 297 ms | 449 ms | 506 ms | 642 ms |

### Vazão em regime

14468 requisições em 2m0s: **120.6 req/s** (3617.0 validações/min).

### Linha do tempo no cliente (janelas de 10 s desde o início da rampa)

| Janela | N | Erros | p50 | p95 | Máx. |
|---|---|---|---|---|---|
| 0s–10s | 227 | 0 | 290 ms | 338 ms | 903 ms |
| 10s–20s | 639 | 0 | 295 ms | 353 ms | 491 ms |
| 20s–30s | 992 | 0 | 345 ms | 429 ms | 501 ms |
| 30s–40s | 1072 | 0 | 438 ms | 586 ms | 710 ms |
| 40s–50s | 1076 | 0 | 420 ms | 511 ms | 601 ms |
| 50s–60s | 1141 | 0 | 389 ms | 472 ms | 628 ms |
| 60s–70s | 1231 | 0 | 306 ms | 404 ms | 549 ms |
| 70s–80s | 1242 | 0 | 296 ms | 372 ms | 466 ms |
| 80s–90s | 1221 | 0 | 307 ms | 394 ms | 467 ms |
| 90s–100s | 1269 | 0 | 286 ms | 341 ms | 429 ms |
| 100s–110s | 1224 | 0 | 303 ms | 377 ms | 593 ms |
| 110s–120s | 1260 | 0 | 288 ms | 338 ms | 409 ms |
| 120s–130s | 1265 | 0 | 290 ms | 337 ms | 429 ms |
| 130s–140s | 1237 | 0 | 294 ms | 343 ms | 630 ms |
| 140s–150s | 1230 | 0 | 303 ms | 376 ms | 571 ms |

### Latência no provedor (Monitoring, janela da rampa ao fim)

| Medida | p50 | p95 | p99 |
|---|---|---|---|
| gateway, total (`serviceruntime…/request_latencies`) | 197 ms | 452 ms | 510 ms |
| gateway, parcela do back-end (`…request_latencies_backend`) | 197 ms | 449 ms | 509 ms |
| gateway, sobrecarga própria (`…request_latencies_overhead`) | 0 ms | 1 ms | 1 ms |
| função (`run.googleapis.com/request_latencies`) | 147 ms | 317 ms | 393 ms |

### Instâncias da função por minuto (ativas + ociosas, máximo no minuto)

| Minuto | Instâncias |
|---|---|
| 17:12 | 1 |
| 17:13 | 1 |
| 17:14 | 2 |

### p95 da função por minuto (Monitoring)

| Minuto | p95 |
|---|---|
| 17:13 | 364 ms |
| 17:14 | 259 ms |

### Escala e partidas

Requisições contadas pela função na janela: **11269**. Máximo de instâncias alocadas: **2**. Partidas de instância (`startup_latencies`): **2**, média **444 ms**.

### Conclusão

- ✅ nenhuma requisição com erro em regime (0 de 14468)
- ✅ RNF-05: p95 no gateway com 100 operadores = 452 ms (alvo ≤ 500 ms)
