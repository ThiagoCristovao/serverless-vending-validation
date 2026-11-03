# ADR-0017 — Aquecimento do gateway por tráfego periódico (`GET /v1/saude` pelo Cloud Scheduler)

- **Status:** aceita
- **Data:** 2026-10-30 (medição em 2026-11-03)
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

Medição de 03/11/2026 (`docs/avaliacao/evidencias/cold-start-aquecido.md`; seis amostras, 20 min
de inatividade, os dois jobs ativos, configuração A com `min_instancias = 0`):

- **Primeira requisição no cliente: mediana 1,21 s, média 1,63 s, máximo 2,19 s**, contra 6,0 s
  de mediana e 8,0 s de máximo sem o aquecimento. Três amostras ficaram entre 1,08 e 1,21 s e três
  entre 2,06 e 2,19 s.
- **A parcela do gateway desapareceu.** O gateway cronometrou 0,02 a 0,05 s além da função nas
  seis amostras, com os dois agentes (`ESPv2` e `cloudesf`), e a diferença entre cliente e gateway
  caiu para 0,29 s (rede e conexão TLS do cliente; 0,14 s na segunda requisição, que reaproveita a
  conexão), contra 1,1 a 3,3 s antes de o gateway começar a cronometrar.
- **O que sobra é a função**, na mesma instância nas seis amostras e sem partida de contêiner:
  0,75 a 0,86 s em metade delas e 1,73 a 1,88 s na outra metade, alternando a cada amostra (a
  segunda requisição levou 0,29 a 0,46 s). A alternância estrita (lenta aos 0, 40 e 80 min da
  medição; rápida aos 20, 60 e 100 min) é compatível com uma conexão usada só pelas validações,
  provavelmente o canal do publicador do Pub/Sub, que a reconciliação não exercita quando não há
  pendências, e que sobrevive a 20 min de ociosidade mas não a 40; confirmar exigiria
  rastreamento, fora do escopo.

A meta original do RNF-06 (1,5 s) passa a ser atendida pela mediana; metade das amostras ainda
fica acima dela, por conta da função. O critério de aceitação continua o do ADR-0016 (≤ 10 s de
ponta a ponta), agora com folga. Se a parcela restante importar, o próximo passo é um aquecimento
que exercite as dependências da validação (Firestore e publicador do Pub/Sub), ao custo de uma
operação de verificação a cada 5 min; `min_instancias = 1` não ajudaria, porque a penalidade é de
conexões ociosas, não de partida (seção 3.2 do relatório).

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

- Em uso normal, a primeira requisição após inatividade cai de 6 s para 1,2 s de mediana
  (máximo 2,2 s); a parcela restante é da função.
- Mais um job no Cloud Scheduler e 288 requisições por dia pelo gateway; custo dentro da camada
  gratuita.
- O contrato passa a ter uma operação pública; a documentação de segurança do contrato e o
  módulo `gateway` registram isso.
- Para medir o pior caso (sem aquecimento), a ferramenta pausa os dois jobs; um `carga-frio`
  interrompido exige retomá-los manualmente.

## Referências

- ADR-0012, ADR-0015, ADR-0016; `docs/avaliacao/escalabilidade.md` (seção 3);
  `docs/avaliacao/evidencias/cold-start-aquecido.md`; `infra/modulos/gateway/README.md`.
