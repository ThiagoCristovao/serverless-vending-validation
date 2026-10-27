# Relatório de resiliência e independência (Sprint 9)

Avaliação técnica de dois dos três atributos de qualidade do trabalho (TCC 1, Seção 3.4.7):
**resiliência operacional** e **independência em relação ao sistema central**. Cada cenário de
exceção documentado na Sprint 2 ([arquitetura.md, seção 9](../arquitetura.md)) que envolve uma
falha de ambiente foi induzido deliberadamente no ambiente `svv-dev` e teve o comportamento
registrado com as ferramentas de observabilidade do provedor (Cloud Monitoring, Cloud Logging,
Firestore) e com os registros do consumidor e do dispositivo. Os mecanismos de injeção estão no
[ADR-0014](../adr/0014-injecao-de-falhas-por-iam-proxy-e-dns.md); os relatórios brutos gerados
pelas ferramentas ficam em [evidencias/](evidencias/).

## 1. Método

| Lado | Como a falha é induzida | Como se observa | Ferramenta |
|---|---|---|---|
| Serviço | pausa do consumidor; remoção temporária de um vínculo IAM da conta da função (Pub/Sub, Firestore) | respostas do gateway, marcadores no Firestore, métricas da assinatura, logs da função, registro do consumidor | `make experimento CENARIO=…` (`sistema-central-simulado/cmd/experimento`) |
| Dispositivo | proxy HTTP entre o aplicativo e o gateway (corte, atraso, 503, resposta descartada, intermitência); DNS que não resolve o Firebase Authentication | tela do aplicativo, log do proxy, `logcat`, Cloud Logging | `make injecao-proxy`, `make injecao-dns`, `make app-apk-injecao` (`ferramentas/injecao-falhas`) |

Os cenários do lado do serviço são automatizados e reproduzíveis; cada execução grava um
relatório com parâmetros, linha do tempo, tabelas e critérios verificados. Os cenários do
dispositivo seguem o roteiro de `ferramentas/injecao-falhas/README.md` em um Motorola Edge 50 Pro
(Android 15).

## 2. Cenários do lado do serviço

### A. Sistema central indisponível durante validações (CE-15; RNF-01, RNF-03)

Executado em 27/10/2026 ([evidência](evidencias/central-indisponivel.md)). O consumidor simulado
foi pausado por 3 min; durante a pausa, 6 validações completas (abertura e conclusão) foram
feitas pelo gateway, alternando as máquinas `VM-E2E1` e `VM-E2E2`.

| Observação | Valor |
|---|---|
| Validações completas durante a indisponibilidade | 6 de 6, todas `201`/`200` |
| Tempo de resposta (abertura) | 5,5 s e 3,6 s nas duas primeiras (partida a frio de instâncias), 0,3 a 0,4 s nas demais |
| Backlog da assinatura (`num_undelivered_messages`, Monitoring) | 0 → 6 → 12 → 12 → 6 (amostras por minuto) |
| Idade da mensagem mais antiga (`oldest_unacked_message_age`) | 10 s → 70 s → 130 s durante a pausa |
| Eventos consumidos após a retomada | 12 de 12, em 9,0 s, na ordem por máquina, sem duplicatas |

**Conclusão.** A indisponibilidade do sistema central é invisível ao operador (RNF-03); as
mensagens ficam retidas na camada de mensageria e são consumidas integralmente, em ordem e sem
intervenção manual ao restabelecer (RNF-01). O backlog e a idade das mensagens no Monitoring são
os sinais operacionais da falha, com o alerta de backlog configurado para 10 min.

### B. Pub/Sub indisponível ao publicar (CE-14; RNF-04, ADR-0012)

Executado em 27/10/2026 ([evidência](evidencias/publicacao-indisponivel.md)). O vínculo
`roles/pubsub.publisher` da conta da função foi removido do tópico; a falha ficou ativa
1 min 19 s depois (a primeira validação cujo marcador `eventos.iniciadaPublicadaEm` ficou nulo).
Com a publicação indisponível, foram feitas 6 validações completas; o vínculo foi restaurado após
1 min 28 s de falha e a reconciliação periódica do Cloud Scheduler (ADR-0012) republicou as
pendências.

| Observação | Valor |
|---|---|
| Validações completas com a publicação indisponível | 6 de 6, todas `201`/`200`, entre 0,49 e 0,61 s |
| Publicações pendentes durante a falha (Firestore) | 12 de 12 marcadores `null`, com `status = concluida` |
| Logs da função durante a falha | 14 avisos `falha ao publicar evento; publicação pendente`, um por evento |
| Reconciliação | chamada do Scheduler em 15:25:03 (1,97 s) republicou tudo; 12 marcadores gravados |
| Eventos consumidos após restaurar | 28 de 28 (sondas e validações da falha), em 4 min 15 s, na ordem por máquina |

**Conclusão.** A resposta ao operador não depende da publicação (RNF-04): a validação é
persistida antes, e a falha de publicação vira uma pendência silenciosa (CE-14). A reconciliação
periódica republica sem intervenção e o consumidor idempotente absorve repetições; o tempo de
recuperação é limitado pelo período do Scheduler (5 min) e pela idade mínima das pendências
(2 min).

### C. Firestore indisponível (CE-13; RF-23)

Executado em 27/10/2026 ([evidência](evidencias/firestore-indisponivel.md)). O vínculo
`roles/datastore.user` da conta da função foi removido do projeto; o serviço passou a responder
`503 dependencia_indisponivel` 1 min 37 s depois. Foram feitas 6 requisições de abertura, cada
uma com os três reenvios automáticos do aplicativo (1 s, 2 s, 4 s).

| Observação | Valor |
|---|---|
| Respostas durante a falha | 24 de 24 `503 dependencia_indisponivel`, entre 161 e 256 ms (falha rápida) |
| Estado parcial | nenhum: nada é gravado quando o armazenamento falha; as mesmas chaves de idempotência puderam ser reutilizadas |
| Recuperação após restaurar o vínculo | primeiro `201` 1 min 12 s depois, com a chave da primeira requisição que havia falhado; conclusão e consumo pelo sistema central normais |
| Logs da função | 32 entradas com `codigo = dependencia_indisponivel`, visíveis na métrica derivada de log `svv-dev-erros-por-codigo` |

**Conclusão.** A indisponibilidade do armazenamento é convertida em falha rápida e explícita
(`503` com código estável), que o aplicativo trata com reenvios e, por fim, com a mensagem
"Serviço indisponível" ao operador (RF-22, RF-23). A recuperação não exige intervenção: o reenvio
com a mesma `Idempotency-Key` cria a validação normalmente. A propagação do IAM após a
restauração não foi monotônica: houve um `201` 1 min 12 s após restaurar, um `503` isolado 3 min
45 s depois e respostas consistentes a partir de cerca de 5 min (Cloud Logging, 18:12:58 a
18:17:51 UTC), o que reforça a necessidade dos reenvios no cliente.

## 3. Cenários do lado do dispositivo

Roteiro em [ferramentas/injecao-falhas/README.md](../../ferramentas/injecao-falhas/README.md).
Execução pendente no aparelho; a tabela registra o esperado e será completada com o observado.

| # | Cenário | Esperado | Observado |
|---|---|---|---|
| D1 | Sem rede ao validar (CE-01, CE-02) | três reenvios com a mesma `Idempotency-Key`, mensagem "Sem conexão" e "Tentar novamente"; ao voltar a rede, `201` sem duplicar | *pendente* |
| D2 | Rede lenta (CE-02) | tempo esgotado em 20 s e reenvios | *pendente* |
| D3 | Resposta da conclusão perdida (CE-03) | o serviço conclui; na retentativa o aplicativo consulta `GET` e exibe a conclusão sem reenviar | *pendente* |
| D4 | Rede intermitente (RF-23) | fluxo completo com os reenvios absorvendo as falhas | *pendente* |
| E1 | Provedor de autenticação indisponível no login (CE-05) | "Autenticação indisponível no momento" (rede no ar, provedor sem resposta) | *pendente* |
| E2 | Sessão iniciada com o provedor indisponível (CE-05) | validação e conclusão normais com o token em cache | *pendente* |
| E3 | Restabelecimento do provedor | login volta a funcionar | *pendente* |

A diferença entre "sem conexão" e "autenticação indisponível" no login foi introduzida nesta
sprint: o SDK do Firebase devolve o mesmo erro nos dois casos, e o aplicativo passou a
distingui-los pelo estado de conectividade do aparelho.

## 4. Síntese por requisito

| Requisito / cenário | Resultado | Onde |
|---|---|---|
| RNF-01 — nenhum evento perdido com o consumidor indisponível | atendido: 12/12 eventos consumidos após 3 min de pausa | A |
| RNF-02 — falhas repetidas não bloqueiam as demais | atendido na Sprint 8 (DLQ após 5 tentativas; outra máquina seguiu) | [e2e](../../sistema-central-simulado/README.md) |
| RNF-03 — validação completa com o sistema central indisponível | atendido: 6/6 validações `201`/`200` | A |
| RNF-04 — resposta não depende da publicação | atendido: 6/6 validações `201`/`200` com 12/12 publicações pendentes | B |
| RNF-13 — reenvios não duplicam validações, rotações nem eventos | atendido: reenvio idempotente e duplicatas ignoradas (Sprint 8); chaves reutilizadas após `503` (C) | e2e, C |
| CE-13 — Firestore indisponível | `503` rápido, recuperação sem intervenção | C |
| CE-14 — Pub/Sub indisponível ao publicar | pendências reconciliadas em 4 min 15 s, 28/28 eventos consumidos | B |
| CE-15 — sistema central indisponível | mensagens retidas e consumidas em ordem | A |
| CE-16, CE-17 — DLQ e duplicatas | Sprint 8 | e2e |
| CE-01, CE-02, CE-03, CE-05 — rede e autenticação no dispositivo | pendentes de execução no aparelho | seção 3 |

## 5. Observações e limitações

- **Propagação de IAM.** A remoção de um vínculo levou de 59 s a 1 min 19 s (tópico, duas
  execuções) e 1 min 37 s (projeto) para surtir efeito; a restauração não foi monotônica (ver C).
  Os experimentos sondam o serviço até observar o efeito e registram esses tempos, que fazem
  parte da evidência.
- **Granularidade das métricas.** As métricas do Pub/Sub no Monitoring têm um ponto por minuto e
  atraso de cerca de 1 min; por isso o backlog aparece em patamares (6, 12) e a amostra posterior à
  retomada ainda mostra 6.
- **Partida a frio.** As primeiras requisições de cada experimento levaram 3 a 5 s (instâncias
  novas da função); o tema é objeto da Sprint 10 (RNF-06, CE-18).
- **Contador de tentativas do Pub/Sub.** Na Sprint 8 a mensagem rejeitada foi entregue 10 vezes
  antes da DLQ, com o contador reiniciando após a 5ª; a documentação descreve a contagem como
  aproximada.
- **Escopo.** A indisponibilidade do provedor de autenticação é simulada por DNS no dispositivo;
  o gateway continua validando os JWT já emitidos, como ocorreria em uma falha real do provedor.

## 6. Como reproduzir

```bash
make e2e-preparar                                   # uma vez: máquinas VM-E2E1 e VM-E2E2
make experimento CENARIO=central-indisponivel       # ~6 min
make experimento CENARIO=publicacao-indisponivel    # ~10 min (espera a reconciliação)
make experimento CENARIO=firestore-indisponivel     # ~5 min
make experimento CENARIO=restaurar                  # se um cenário for interrompido
make app-apk-injecao && make injecao-proxy          # cenários D no celular
make injecao-dns                                    # cenários E no celular
```

Entre um cenário de IAM e o seguinte, aguarde o serviço responder `201` de forma consistente
(cerca de 5 min após a restauração), ou o próximo experimento registrará sondas com `503`.
