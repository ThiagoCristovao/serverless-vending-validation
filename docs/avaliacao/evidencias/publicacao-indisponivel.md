## Cenário B — Pub/Sub indisponível ao publicar (CE-14, RNF-04, ADR-0012)

Executado em 2026-10-27 15:19:23 -03 (duração 5m49s). Resultado: **todos os critérios atendidos**.

### Parâmetros

| Parâmetro | Valor |
|---|---|
| projeto | svv-dev |
| gateway | https://svv-dev-gateway-4mjof8z9.ue.gateway.dev/v1 |
| validações durante a falha | 6 |
| registro do consumidor | /tmp/svv-laboratorio-1042900806/eventos.jsonl |
| tópico | svv-dev-validacoes |
| falha injetada | remoção de roles/pubsub.publisher da conta da função no tópico |

### Linha do tempo

```text
15:19:25  vínculo pubsub.publisher REMOVIDO da conta da função
15:19:27  sonda 01M5Z9DB3F3JRQVVYZJCVEYZFV publicada normalmente (IAM ainda propagando)
15:19:38  sonda 01M5Z9DPQMHW3Z97HX9J8BY65W publicada normalmente (IAM ainda propagando)
15:19:49  sonda 01M5Z9E19X017JV72DDNKHE83W publicada normalmente (IAM ainda propagando)
15:20:00  sonda 01M5Z9EBZZHW8RH0G4QK6J1605 publicada normalmente (IAM ainda propagando)
15:20:11  sonda 01M5Z9EPPH36T1RYS0CEG4FGVD publicada normalmente (IAM ainda propagando)
15:20:22  sonda 01M5Z9F1D9A7EGGEZ2WZ5CFY2F publicada normalmente (IAM ainda propagando)
15:20:33  sonda 01M5Z9FC3H97N78D9H8HPNTN1C publicada normalmente (IAM ainda propagando)
15:20:44  falha ATIVA: validação 01M5Z9FPTC8HNT3C23YYSMRFK1 ficou com eventos.iniciadaPublicadaEm nulo (1m18.982s após a remoção)
15:20:50  6 validações executadas com a publicação indisponível; 6 completas sem erro
15:20:51  12 de 12 publicações pendentes (marcador nulo)
15:20:53  vínculo pubsub.publisher RESTAURADO após 1m27.7s de falha; aguardando a reconciliação (Cloud Scheduler a cada 5 min, pendências com mais de 2 min)
15:25:08  28 de 28 eventos consumidos 4m15.032s após restaurar
15:25:12  ✅ RNF-04: 6/6 validações completas (201/200) com a publicação indisponível
15:25:12  ✅ CE-14: 12/12 publicações ficaram pendentes (marcador nulo) sem afetar a resposta ao operador
15:25:12  ✅ ADR-0012: reconciliação republicou e o consumidor recebeu 28/28 eventos em 4m15.032s após restaurar (12 marcadores gravados)
15:25:12  ✅ ordem por máquina preservada após a reconciliação
```

### Validações com a publicação indisponível

| # | Máquina | idValidacao | Abertura | Tempo | Conclusão | Tempo | Erro |
|---|---|---|---|---|---|---|---|
| 1 | VM-E2E1 | 01M5Z9FR2Q9HZ0Z1VFPMRES0SM | 201 | 526ms | 200 | 500ms | — |
| 2 | VM-E2E2 | 01M5Z9FS03Q9R9XCRTXC8FVPTJ | 201 | 498ms | 200 | 509ms | — |
| 3 | VM-E2E1 | 01M5Z9FSZNG4KZ6NQ4M82EVBTD | 201 | 488ms | 200 | 505ms | — |
| 4 | VM-E2E2 | 01M5Z9FTY7TM96ZN11Q5PCQWF0 | 201 | 508ms | 200 | 491ms | — |
| 5 | VM-E2E1 | 01M5Z9FVYDAREAXQ334W97Q19K | 201 | 613ms | 200 | 488ms | — |
| 6 | VM-E2E2 | 01M5Z9FWZMH9ZS0XZQPE1JMG8Q | 201 | 500ms | 200 | 498ms | — |

### Marcadores de publicação no Firestore durante a falha

| idValidacao | status | iniciadaPublicadaEm | concluidaPublicadaEm |
|---|---|---|---|
| 01M5Z9FR2Q9HZ0Z1VFPMRES0SM | concluida | **null** | **null** |
| 01M5Z9FS03Q9R9XCRTXC8FVPTJ | concluida | **null** | **null** |
| 01M5Z9FSZNG4KZ6NQ4M82EVBTD | concluida | **null** | **null** |
| 01M5Z9FTY7TM96ZN11Q5PCQWF0 | concluida | **null** | **null** |
| 01M5Z9FVYDAREAXQ334W97Q19K | concluida | **null** | **null** |
| 01M5Z9FWZMH9ZS0XZQPE1JMG8Q | concluida | **null** | **null** |

### Marcadores de publicação no Firestore após a reconciliação

| idValidacao | status | iniciadaPublicadaEm | concluidaPublicadaEm |
|---|---|---|---|
| 01M5Z9FR2Q9HZ0Z1VFPMRES0SM | concluida | 15:25:03 | 15:25:03 |
| 01M5Z9FS03Q9R9XCRTXC8FVPTJ | concluida | 15:25:03 | 15:25:03 |
| 01M5Z9FSZNG4KZ6NQ4M82EVBTD | concluida | 15:25:03 | 15:25:03 |
| 01M5Z9FTY7TM96ZN11Q5PCQWF0 | concluida | 15:25:03 | 15:25:03 |
| 01M5Z9FVYDAREAXQ334W97Q19K | concluida | 15:25:03 | 15:25:03 |
| 01M5Z9FWZMH9ZS0XZQPE1JMG8Q | concluida | 15:25:03 | 15:25:03 |

### Logs da função (Cloud Logging): publicações pendentes e chamadas de reconciliação

| Instante | Severidade | Mensagem | Rota | idValidacao | Status |
|---|---|---|---|---|---|
| 15:20:04 | INFO | log de acesso: POST /interno/reconciliar-publicacoes → 200 em 0.063538267s | — | — | — |
| 15:20:04 | INFO | requisição atendida | /interno/reconciliar-publicacoes | — | 200 |
| 15:20:44 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FPTC8HNT3C23YYSMRFK1 | — |
| 15:20:44 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FPTC8HNT3C23YYSMRFK1 | — |
| 15:20:45 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FR2Q9HZ0Z1VFPMRES0SM | — |
| 15:20:45 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FR2Q9HZ0Z1VFPMRES0SM | — |
| 15:20:46 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FS03Q9R9XCRTXC8FVPTJ | — |
| 15:20:46 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FS03Q9R9XCRTXC8FVPTJ | — |
| 15:20:47 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FSZNG4KZ6NQ4M82EVBTD | — |
| 15:20:47 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FSZNG4KZ6NQ4M82EVBTD | — |
| 15:20:48 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FTY7TM96ZN11Q5PCQWF0 | — |
| 15:20:48 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FTY7TM96ZN11Q5PCQWF0 | — |
| 15:20:49 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FVYDAREAXQ334W97Q19K | — |
| 15:20:49 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FVYDAREAXQ334W97Q19K | — |
| 15:20:50 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FWZMH9ZS0XZQPE1JMG8Q | — |
| 15:20:50 | WARNING | falha ao publicar evento; publicação pendente | — | 01M5Z9FWZMH9ZS0XZQPE1JMG8Q | — |
| 15:25:03 | INFO | log de acesso: POST /interno/reconciliar-publicacoes → 200 em 1.973482662s | — | — | — |

### Conclusão

- ✅ RNF-04: 6/6 validações completas (201/200) com a publicação indisponível
- ✅ CE-14: 12/12 publicações ficaram pendentes (marcador nulo) sem afetar a resposta ao operador
- ✅ ADR-0012: reconciliação republicou e o consumidor recebeu 28/28 eventos em 4m15.032s após restaurar (12 marcadores gravados)
- ✅ ordem por máquina preservada após a reconciliação
