# ADR-0017 — Aquecimento do gateway por tráfego periódico (`GET /v1/saude` pelo Cloud Scheduler)

- **Status:** aceita
- **Data:** 2026-10-30
- **Sprint:** 10 (trabalho futuro do ADR-0016, executado após o fechamento da sprint)

## Contexto

O ADR-0016 registrou que a primeira requisição após 20 min de inatividade leva 6 s, dos quais
3,0 a 4,6 s são a partida da instância do API Gateway e 1,6 a 2,7 s a parcela da função, e que
`min_instancias = 1` não muda a mediana. O job de reconciliação (ADR-0012) mantém a função
aquecida a cada 5 min, mas chama o serviço Cloud Run diretamente, sem atravessar o gateway. A
mitigação preferida pelo ADR-0016 era tráfego periódico que atravessasse o gateway.

## Decisão

1. **Rota pública `GET /v1/saude` no contrato.** É a mesma verificação que a função já expunha
   em `/saude` para o Cloud Run, agora também sob `/v1` e declarada em `api/openapi.yaml` com
   `security: []`: a única operação sem autenticação. Responde `200 {"status":"ok"}` sem tocar
   nas dependências e sem devolver dados; o custo de abuso é o de uma requisição vazia, limitada
   pelo teto de instâncias da função.
2. **Job `svv-<amb>-aquecer-gateway` no módulo `gateway`.** Cloud Scheduler chama
   `https://<gateway>/v1/saude` a cada 5 min (`gateway_aquecimento_cron`; `null` desabilita),
   sem autenticação e sem retentativa. Cada chamada atravessa o gateway e chega à função, o que
   mantém aquecidos o gateway e a instância da função (e suas conexões), em uso normal.
3. **A ferramenta de cold start distingue os dois regimes.** `make carga-frio` com
   `-agendador pausar` (padrão) pausa a reconciliação e o aquecimento e mede o pior caso, como
   nas evidências `min0` e `min1`; com `-agendador manter` mede o uso normal, com os dois jobs
   ativos (`CONFIGURACAO=aquecido`).

## Resultado

*(a preencher com a medição `cold-start-aquecido.md`)*

## Alternativas consideradas

- **Chamar uma rota autenticada pelo Scheduler.** Exigiria um segundo esquema de segurança no
  contrato (token OIDC da conta do Scheduler, emissor `accounts.google.com`) só para essa
  operação; mais superfície no contrato do que uma rota de saúde sem dados.
- **Apontar o job de reconciliação para o gateway.** A reconciliação é interna (fora do contrato
  público, autenticada por OIDC) e não deve ganhar exposição pública só para aquecer o gateway.
- **Chamada direta do aplicativo ao Cloud Run.** Dispensaria o gateway, mas é mudança de
  arquitetura (autenticação, contrato, chaves) sem ganho além do cold start; mantida como
  alternativa no ADR-0016.

## Consequências

- Mais um job no Cloud Scheduler e 288 requisições por dia pelo gateway; custo dentro da camada
  gratuita.
- O contrato passa a ter uma operação pública; a documentação de segurança do contrato e o
  módulo `gateway` registram isso.
- Para medir o pior caso (sem aquecimento), a ferramenta pausa os dois jobs; um `carga-frio`
  interrompido exige retomá-los manualmente.

## Referências

- ADR-0012, ADR-0015, ADR-0016; `docs/avaliacao/escalabilidade.md` (seção 3);
  `docs/avaliacao/evidencias/cold-start-aquecido.md`; `infra/modulos/gateway/README.md`.
