# sistema-central-simulado — consumidor do sistema central (Go)

Simula o sistema central da rede de vending como assinante do tópico `svv-<ambiente>-validacoes`.
É um módulo Go **independente do serviço**: conhece só o contrato dos eventos
([docs/modelo-dados.md](../docs/modelo-dados.md), seção 6), como faria o sistema real (ADR-0013).
Sustenta a suíte de ponta a ponta da Sprint 8 e os cenários de resiliência das Sprints 9 e 10.

| Sprint | Estado | Conteúdo |
|---|---|---|
| 8 | **concluída** (suíte e2e executada contra `svv-dev` em 26/10/2026) | consumidor idempotente com registro em JSON Lines, interruptor de falhas, API administrativa, testes com Pub/Sub falso em processo e suíte e2e (fluxo principal, duplicatas, DLQ) |
| 9 | **concluída** (experimentos executados contra `svv-dev` em 27/10/2026) | `cmd/experimento`: cenários de injeção de falhas no lado do serviço (sistema central, Pub/Sub e Firestore indisponíveis) com relatório de evidências; pacote `laboratorio` compartilhado com a e2e |
| 10 | **concluída** (cargas e cold start executados contra `svv-dev` em 29 e 30/10/2026) | `cmd/carga`: operadores virtuais em laço fechado (10, 50 e 100) contra o gateway, comparação de configurações da função e cold start após inatividade, com relatório de evidências (ADR-0015) |

## Comportamento

- Consome `svv-<ambiente>-sistema-central` (pull em streaming, ordenação por `idMaquina`, uma
  goroutine de recepção).
- **Idempotência** por `(idValidacao, tipo)` (RF-24, CE-17): o evento é gravado em
  `dados/eventos.jsonl` e sincronizado em disco **antes** do `ack`; o índice é reconstruído na
  partida, então duplicatas são reconhecidas mesmo após reinício e confirmadas sem registrar.
- Corpo fora do contrato (JSON inválido, tipo desconhecido, versão diferente de 1, campos
  obrigatórios ausentes) é rejeitado com `nack`; após 5 tentativas o Pub/Sub o encaminha à DLQ
  (RF-25, CE-16) e o consumo das demais máquinas não é afetado (RNF-02).
- **Interruptor de falhas** (Sprints 8 e 9): `rejeitar` (nack antes de persistir),
  `confirmacao-perdida` (persiste e não confirma, como um ack perdido na rede) ou `nenhum`, por
  máquina ou geral; além de pausar e retomar o consumo (CE-15).
- Logs estruturados em JSON com `idMensagem`, `tentativa`, `idValidacao`, `tipo`, `idMaquina`.

## API administrativa (porta 8090)

| Rota | Uso |
|---|---|
| `GET /saude` | estado resumido: consumindo, total de eventos |
| `GET /eventos?idValidacao=&idMaquina=&tipo=` | eventos registrados, na ordem de chegada, com corpo e atributos originais |
| `GET /estatisticas` | recebidas, registradas, duplicadas, rejeitadas, inválidas, confirmações perdidas |
| `GET`/`PUT /admin/falhas` | interruptor: `{"modo":"rejeitar","idMaquina":"VM-2047"}` |
| `POST /admin/pausar` / `POST /admin/retomar` / `GET /admin/estado` | controle do consumo |

## Estrutura

```
cmd/central/main.go              flags/variáveis SVV_CENTRAL_*: projeto, assinatura, arquivo, porta, impersonar
cmd/experimento/                 Sprint 9: cenários de injeção de falhas e relatório Markdown (make experimento)
cmd/carga/                       Sprint 10: preparação de operadores virtuais, teste de carga e cold start (make carga-preparar, carga, carga-frio)
internal/
├── eventos/                     contrato dos eventos (decodificação e validação; campos extras tolerados)
├── armazenamento/               registro JSON Lines + índice por (idValidacao, tipo)
├── falhas/                      interruptor de indisponibilidade
├── consumidor/                  Processador (decisão ack/nack, estatísticas) e Consumidor (Pub/Sub, pausa/retomada)
├── admin/                       API HTTP
└── laboratorio/                 ambiente, operador de testes, adesivos assinados, gateway e consumidor em processo
e2e/                             suíte de ponta a ponta (tag `e2e`): fluxo principal, duplicatas, DLQ
```

## Executar e testar

```bash
make central-verificar     # vet + testes (Pub/Sub falso em processo: ordenação, DLQ, duplicatas)
make central-executar      # consome svv-dev-sistema-central; API em :8090
make e2e-preparar          # uma vez: semeia VM-E2E1 e VM-E2E2 no Firestore de dev
make e2e                   # suíte e2e contra svv-dev (~5 min); E2E_ARGS=-short pula o cenário de DLQ
make experimento CENARIO=central-indisponivel   # Sprint 9; também publicacao-indisponivel, firestore-indisponivel, restaurar
make carga-preparar OPERADORES=100              # Sprint 10, uma vez: contas e máquinas de carga
make carga OPERADORES=50 CONFIGURACAO=A         # teste de carga (relatório em docs/avaliacao/evidencias)
make carga-frio CONFIGURACAO=min0               # cold start após inatividade (~2 h)
```

Variáveis do comando (`SVV_CENTRAL_PROJETO`, `SVV_CENTRAL_ASSINATURA`, `SVV_CENTRAL_ARQUIVO`,
`SVV_CENTRAL_PORTA`, `SVV_CENTRAL_IMPERSONAR`, `SVV_CENTRAL_NIVEL_LOG`) têm equivalentes em
flags; `PUBSUB_EMULATOR_HOST` aponta para o emulador. Passo a passo e solução de problemas em
[docs/operacao/configuracao-ambiente.md](../docs/operacao/configuracao-ambiente.md), seção 7.

## Suíte e2e

Roda contra o ambiente `dev` real com o consumidor **dentro do processo de teste** (registro em
arquivo temporário, API administrativa em porta efêmera). Em cada cenário, a asserção final é
feita pela API administrativa, como faria um sistema externo:

| Teste | Cenário | Requisitos |
|---|---|---|
| `TestFluxoPrincipalAteOConsumo` | abertura, reenvio com a mesma `Idempotency-Key` (mesma validação), nova chave com validação aberta (`409`), conclusão; `iniciada` e `concluida` consumidas uma vez cada, em ordem, com leituras, rotação e operador | RF-14, RF-24, CE-02, CE-11 |
| `TestMensagemDuplicadaEhIgnorada` | cópia da mensagem republicada no tópico; `confirmacao-perdida` seguida da reentrega do Pub/Sub; nenhum registro extra | RF-24, RNF-13, CE-17 |
| `TestFalhasSucessivasEncaminhamParaDlq` | `rejeitar` em `VM-E2E2`: outra máquina segue consumida; após 5 tentativas a mensagem aparece na `validacoes-dlq-inspecao` com `CloudPubSubDeadLetterSourceDeliveryCount`; falha sanada, a conclusão da máquina é consumida | RF-25, RNF-02, CE-16 |

## Critério da Sprint 8

"A suíte executa contra o ambiente de desenvolvimento e cobre tanto o fluxo principal quanto os
cenários de duplicação e falha de processamento."

**Executada em 26/10/2026** contra `svv-dev` (`make e2e`, 3 cenários, 234 s):

| Cenário | Observado |
|---|---|
| Fluxo principal | `201` (5,3 s, partida a frio), reenvio com a mesma chave `201` (0,3 s) com a mesma validação, nova chave `409 validacao_em_andamento`, conclusão `200`; `iniciada` e `concluida` consumidas 2,5 s após a conclusão, uma vez cada, em ordem |
| Duplicatas | cópia republicada no tópico reconhecida em 6 s sem registro novo; com `confirmacao-perdida`, a reentrega do Pub/Sub chegou 14 s depois (espera mínima de 10 s) e foi reconhecida pela chave `(idValidacao, tipo)` |
| Falhas sucessivas | com `rejeitar` em `VM-E2E2`, a `VM-E2E1` continuou sendo consumida em 0,5 s; a mensagem rejeitada foi entregue 10 vezes (o contador de tentativas reiniciou após a 5ª) e chegou à DLQ 3 min 11 s após a publicação, com `CloudPubSubDeadLetterSourceDeliveryCount = 5`; falha sanada, a conclusão da máquina foi consumida de imediato |

Resumo do consumidor ao final: 21 recebidas, 9 registradas, 2 duplicadas, 10 rejeitadas,
0 inválidas, 1 confirmação perdida. O Pub/Sub descreve a contagem de tentativas como
aproximada; a observação do reinício do contador fica registrada para a análise da Sprint 9.
