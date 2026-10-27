## Cenário C — Firestore indisponível (CE-13, RF-23)

Executado em 2026-10-27 15:10:29 -03 (duração 3m44s). Resultado: **todos os critérios atendidos**.

### Parâmetros

| Parâmetro | Valor |
|---|---|
| projeto | svv-dev |
| gateway | https://svv-dev-gateway-4mjof8z9.ue.gateway.dev/v1 |
| validações durante a falha | 6 |
| registro do consumidor | /tmp/svv-laboratorio-3325144753/eventos.jsonl |
| falha injetada | remoção de roles/datastore.user da conta da função no projeto |

### Linha do tempo

```text
15:10:32  vínculo roles/datastore.user REMOVIDO da conta da função
15:10:33  sonda 01M5Z8X291N4NTM1H88XCF5JTR aberta normalmente (IAM ainda propagando); concluindo
15:10:43  sonda 01M5Z8XCWNV6H15CSCEMXYF1RX aberta normalmente (IAM ainda propagando); concluindo
15:10:54  sonda 01M5Z8XQGSYQXX7X5BKX6033T8 aberta normalmente (IAM ainda propagando); concluindo
15:11:05  sonda 01M5Z8Y22VHCYRXAK1AYECS6A0 aberta normalmente (IAM ainda propagando); concluindo
15:11:16  sonda 01M5Z8YCNHYAK15X6KDAE6GA5G aberta normalmente (IAM ainda propagando); concluindo
15:11:27  sonda 01M5Z8YQ2PDKKMPX7XPEKGE370 aberta normalmente (IAM ainda propagando); concluindo
15:11:37  sonda 01M5Z8Z1M5WH0F2G41V3V9M4ZH aberta normalmente (IAM ainda propagando); concluindo
15:11:48  sonda 01M5Z8ZC4WTH15NRYP65EHGMV1 aberta normalmente (IAM ainda propagando); concluindo
15:11:59  sonda 01M5Z8ZPK143Q7BXX84T65AMS5 aberta normalmente (IAM ainda propagando); concluindo
15:12:09  falha ATIVA: 503 dependencia_indisponivel (1m37.372s após a remoção)
15:12:56  24 respostas 503 dependencia_indisponivel e 0 de outro tipo
15:12:58  vínculo roles/datastore.user RESTAURADO; sondando até o serviço voltar
15:12:58  ainda 503; nova sonda em 10 s
15:13:08  ainda 503; nova sonda em 10 s
15:13:19  ainda 503; nova sonda em 10 s
15:13:29  ainda 503; nova sonda em 10 s
15:13:39  ainda 503; nova sonda em 10 s
15:13:49  ainda 503; nova sonda em 10 s
15:13:59  ainda 503; nova sonda em 10 s
15:14:10  serviço RECUPERADO: 201 com a chave da 1ª requisição falha, validação 01M5Z93PFGEYJP6MYND25NCH48 (1m12.001s após restaurar)
15:14:11  validação pós-recuperação consumida pelo sistema central: true (1.001s)
15:14:13  ✅ CE-13: todas as 24 tentativas receberam 503 dependencia_indisponivel (falha rápida, sem estado parcial)
15:14:13  ✅ RF-23: após os 3 reenvios, o aplicativo exibiria "Serviço indisponível" ao operador (fluxo coberto por teste de widget)
15:14:13  ✅ recuperação sem intervenção: a chave de uma requisição falha foi reutilizada e criou a validação (1m12.001s após restaurar)
15:14:13  ✅ fluxo completo após a recuperação, inclusive o consumo pelo sistema central
```

### Requisições durante a falha, com os reenvios do aplicativo (1 s, 2 s, 4 s)

| # | Chave | 1ª | 2ª | 3ª | 4ª |
|---|---|---|---|---|---|
| 1 | 7a3d598e | 503 (217ms) | 503 (169ms) | 503 (206ms) | 503 (191ms) |
| 2 | 84212e1a | 503 (198ms) | 503 (167ms) | 503 (238ms) | 503 (167ms) |
| 3 | 097650c1 | 503 (169ms) | 503 (169ms) | 503 (161ms) | 503 (256ms) |
| 4 | c0cde1e8 | 503 (187ms) | 503 (186ms) | 503 (197ms) | 503 (174ms) |
| 5 | c43ee114 | 503 (193ms) | 503 (176ms) | 503 (166ms) | 503 (179ms) |
| 6 | 74d9f9a2 | 503 (177ms) | 503 (178ms) | 503 (164ms) | 503 (200ms) |

### Logs da função (Cloud Logging)

32 entradas com `codigo = dependencia_indisponivel` entre 15:10:32 e 15:14:13 (métrica derivada de log `svv-dev-erros-por-codigo`).

### Conclusão

- ✅ CE-13: todas as 24 tentativas receberam 503 dependencia_indisponivel (falha rápida, sem estado parcial)
- ✅ RF-23: após os 3 reenvios, o aplicativo exibiria "Serviço indisponível" ao operador (fluxo coberto por teste de widget)
- ✅ recuperação sem intervenção: a chave de uma requisição falha foi reutilizada e criou a validação (1m12.001s após restaurar)
- ✅ fluxo completo após a recuperação, inclusive o consumo pelo sistema central
