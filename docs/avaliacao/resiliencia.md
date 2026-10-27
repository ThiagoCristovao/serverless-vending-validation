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
| Dispositivo | proxy HTTP entre o aplicativo e o gateway (corte, atraso, 503, resposta descartada, intermitência); DNS que não resolve o Firebase Authentication | tela do aplicativo, log do proxy, log do `dnsmasq`, Cloud Logging, Firestore | `make injecao-proxy`, `make injecao-dns`, `make app-apk-injecao` (`ferramentas/injecao-falhas`) |

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

Executados em 27/10/2026, das 17:24 às 18:14, em um Motorola Edge 50 Pro (Android 15) na mesma
rede Wi-Fi do computador, seguindo o roteiro de
[ferramentas/injecao-falhas/README.md](../../ferramentas/injecao-falhas/README.md)
([evidência](evidencias/dispositivo.md)). Os cenários D usam o APK de injeção, que aponta para
o proxy na porta 8099, e são observados pelo log do proxy (método, caminho, modo aplicado,
`Idempotency-Key`, status e duração), pelo Cloud Logging da função e pelo Firestore; os cenários
E usam o APK normal e o log de consultas do `dnsmasq`. O que o operador viu foi relatado a cada
passo. O aplicativo não escreve no `logcat`, que não serviu de evidência.

| # | Cenário | Esperado | Observado |
|---|---|---|---|
| D1 | Sem rede ao validar (CE-01, CE-02) | três reenvios com a mesma `Idempotency-Key`, mensagem "Sem conexão" e "Tentar novamente"; ao voltar a rede, `201` sem duplicar | **atendido.** Quatro `POST` cortados às 17:25:22, :23, :25 e :29 (esperas de 1, 2 e 4 s), nenhum chegou ao serviço; "Sem conexão" com "Tentar novamente"; com a rede de volta, `201` em 5,6 s (instância fria) e uma única validação nova no Firestore |
| D2 | Rede lenta (CE-02) | tempo esgotado em 20 s e reenvios | **atendido.** Quatro `POST` com a mesma chave a cada 21–24 s (20 s de tempo esgotado mais as esperas); o proxy os encaminhou após 25 s, quando o aplicativo já havia desistido, e o serviço respondeu `201` aos quatro (o 1º em 0,57 s criando a validação; os demais em 0,06–0,16 s, por idempotência): um único documento; "Sem conexão" após 1,5 min; "Tentar novamente" reaproveitou a chave e abriu a mesma validação |
| D3 | Resposta da conclusão perdida (CE-03) | o serviço conclui; na retentativa o aplicativo consulta `GET` e exibe a conclusão sem reenviar | **atendido.** `POST …/conclusao` processado pela função (`200`, 0,53 s) e resposta descartada às 17:30:55; o aplicativo consultou `GET` (descartado) e, 1 s depois, de novo (`200`): a conclusão apareceu sozinha; o serviço recebeu um único `POST` de conclusão |
| D4 | Rede intermitente (RF-23) | fluxo completo com os reenvios absorvendo as falhas | **atendido após ajuste.** Abertura: `POST` cortado, reenvio em 1 s, `201`. Conclusão, 1ª execução: `POST` cortado, `GET` de recuperação `200` (aberta) e a faixa "Sem conexão"; a conclusão não tinha reenvio automático e, com a alternância estrita do proxy, cada toque repetia o par. Após o ajuste (3.1): 4 envios e 4 consultas em 8,5 s antes de desistir, sem duplicar, sob alternância estrita; com cortes aleatórios em 50 %, conclusão `200` |
| E1 | Provedor de autenticação indisponível no login (CE-05) | "Autenticação indisponível no momento" (rede no ar, provedor sem resposta) | **atendido após dois ajustes no roteiro** (3.2). 1ª tentativa: o login passou porque o SDK Android entra por `www.googleapis.com`, host que não estava bloqueado; 2ª: passou pelo cache do resolvedor e pela conexão já aberta; 3ª, após desligar e ligar o Wi-Fi: consulta a `www.googleapis.com` respondida com `NXDOMAIN` às 18:12:16 e a mensagem "Autenticação indisponível no momento. Tente em instantes" |
| E2 | Sessão iniciada com o provedor indisponível (CE-05) | validação e conclusão normais com o token em cache | **atendido.** Com o bloqueio ativo, o celular só consultou o nome do gateway (18:04:50); validação `201` (1,5 s) e conclusão `200` (0,38 s) sem nenhuma chamada ao Firebase |
| E3 | Restabelecimento do provedor | login volta a funcionar | **atendido.** Com o `dnsmasq` parado e o DNS do celular de volta ao automático, o login funcionou |

A diferença entre "sem conexão" e "autenticação indisponível" no login foi introduzida nesta
sprint: o SDK do Firebase devolve o mesmo erro nos dois casos, e o aplicativo passou a
distingui-los pelo estado de conectividade do aparelho (E1).

### 3.1 Ajuste decorrente do D4: reenvio da conclusão

O ADR-0009 prevê que o aplicativo consulte `GET /validacoes/{id}` antes de reenviar uma
conclusão. O controlador da Sprint 7 consultava, mas nunca reenviava: quando o `POST` era cortado
antes de chegar ao serviço, a consulta devolvia "aberta" e a tela mostrava a faixa de erro,
exigindo um novo toque. Sob a alternância estrita do proxy, o par "`POST` cortado, `GET` ok" se
repetia indefinidamente. O controlador passou a reenviar a conclusão com as mesmas esperas
(1 s, 2 s, 4 s) quando a consulta confirma a validação aberta, o que é seguro porque a consulta
provou que nada foi aplicado, e a tratar `validacao_ja_concluida` consultando e exibindo a
conclusão. Quatro testes de widget cobrem os casos (reenvio após consulta, reenvios esgotados,
"já concluída" por envio anterior e erro de negócio sem consulta).

A alternância estrita 1:1 é um pior caso adversarial que não ocorre em uma rede real: como cada
envio é seguido de uma consulta, o `POST` fica sempre do lado cortado. Mesmo assim o aplicativo
faz o correto: reenvia, consulta, desiste com retorno ao operador e não duplica. Com cortes
aleatórios (modo `cortar` com `proporcao` 0,5) o fluxo completa.

### 3.2 Ajustes no roteiro dos cenários E

- O SDK Android do Firebase Authentication entra com e-mail e senha por
  `www.googleapis.com/identitytoolkit/v3`, não por `identitytoolkit.googleapis.com`; o bloqueio
  passou a incluir os três hosts (`www.googleapis.com`, `identitytoolkit.googleapis.com`,
  `securetoken.googleapis.com`).
- O resolvedor do Android guarda as respostas pelo TTL e o SDK reaproveita conexões abertas: depois
  de ativar o bloqueio é preciso desligar e ligar o Wi-Fi. Os dados móveis ficam desligados para o
  sistema não contornar o DNS do Wi-Fi.
- Com a sessão já iniciada, a ordem E2, E1, E3 usa uma única execução do `dnsmasq`.

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
| CE-01, CE-02 — rede ausente ou lenta ao validar | atendido: reenvios com a mesma chave; nenhuma validação duplicada (4 chegadas atrasadas, 1 documento) | D1, D2 |
| CE-03 — resposta da conclusão perdida | atendido: consulta reconhece a conclusão sem reenviar o `POST` | D3 |
| RF-23 — rede intermitente no dispositivo | atendido após o ajuste do reenvio da conclusão (3.1) | D4 |
| CE-05 — provedor de autenticação indisponível | atendido: mensagem específica no login; sessão iniciada valida e conclui sem o provedor; login volta ao restabelecer | E1, E2, E3 |

## 5. Observações e limitações

- **Propagação de IAM.** A remoção de um vínculo levou de 59 s a 1 min 19 s (tópico, duas
  execuções) e 1 min 37 s (projeto) para surtir efeito; a restauração não foi monotônica (ver C).
  Os experimentos sondam o serviço até observar o efeito e registram esses tempos, que fazem
  parte da evidência.
- **Granularidade das métricas.** As métricas do Pub/Sub no Monitoring têm um ponto por minuto e
  atraso de cerca de 1 min; por isso o backlog aparece em patamares (6, 12) e a amostra posterior à
  retomada ainda mostra 6.
- **Partida a frio.** As primeiras requisições de cada experimento levaram 3 a 5 s (instâncias
  novas da função), e o "Tentar novamente" do D1 levou 5,6 s; o tema é objeto da Sprint 10
  (RNF-06, CE-18).
- **Contador de tentativas do Pub/Sub.** Na Sprint 8 a mensagem rejeitada foi entregue 10 vezes
  antes da DLQ, com o contador reiniciando após a 5ª; a documentação descreve a contagem como
  aproximada.
- **Escopo.** A indisponibilidade do provedor de autenticação é simulada por DNS no dispositivo;
  o gateway continua validando os JWT já emitidos, como ocorreria em uma falha real do provedor.
- **Atraso no proxy.** Em `atrasar`, a requisição é encaminhada depois do atraso mesmo que o
  aplicativo já tenha desistido (o proxy só lê o corpo depois da espera e não percebe a conexão
  fechada). O efeito é realista, um pedido que chega atrasado ao serviço, e tornou o D2 uma
  demonstração mais forte da idempotência do que a prevista: quatro chegadas, uma validação.
- **Intermitência estrita.** O modo `intermitente` alterna corte e passagem a cada requisição; é
  um pior caso determinístico, útil para expor a falta de reenvio da conclusão (3.1), mas não
  representa uma rede real. Para o fluxo completo usou-se `cortar` com `proporcao` 0,5.
- **Evidência do dispositivo.** O aplicativo não escreve no `logcat`; a evidência do lado do
  aparelho é o relato do operador a cada passo, casado com o log do proxy ou do `dnsmasq`, o Cloud
  Logging e o Firestore. Os cenários foram conduzidos interativamente e são reproduzíveis pelo
  roteiro, não automatizados.
- **Hosts do Firebase Authentication.** O bloqueio DNS precisou de `www.googleapis.com` (3.2); a
  lista de hosts depende do SDK e da plataforma e deve ser conferida pelo log do `dnsmasq`.

## 6. Como reproduzir

```bash
make e2e-preparar                                   # uma vez: máquinas VM-E2E1 e VM-E2E2
make experimento CENARIO=central-indisponivel       # ~6 min
make experimento CENARIO=publicacao-indisponivel    # ~10 min (espera a reconciliação)
make experimento CENARIO=firestore-indisponivel     # ~5 min
make experimento CENARIO=restaurar                  # se um cenário for interrompido
make app-apk-injecao && make injecao-proxy          # cenários D no celular (roteiro no README da ferramenta)
make injecao-dns                                    # cenários E; ordem E2, E1, E3 com uma única execução
```

Entre um cenário de IAM e o seguinte, aguarde o serviço responder `201` de forma consistente
(cerca de 5 min após a restauração), ou o próximo experimento registrará sondas com `503`.
