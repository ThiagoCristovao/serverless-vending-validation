# ADR-0012: Reconciliar publicações pendentes com Cloud Scheduler e endpoint interno

- **Status:** aceita
- **Data:** 2026-09-25
- **Decisores:** Thiago Cristovão de Souza
- **Sprint:** 2 (arquitetura) e 5 (implementação)

## Contexto

A validação é persistida antes da publicação no Pub/Sub, e uma falha de publicação não pode
afetar o operador (RNF-04, CE-14). O documento fica então com o marcador de publicação nulo e
precisa ser republicado depois, sem intervenção manual.

## Decisão

Um job do **Cloud Scheduler** chama a cada 5 minutos o endpoint interno
`POST /interno/reconciliar-publicacoes` da própria função, autenticado com token OIDC da conta de
serviço do Scheduler e **fora** do API Gateway. A função consulta as validações com marcador nulo há
mais de 2 minutos (índices já previstos em `docs/modelo-dados.md`), republica os eventos e grava o
marcador ao conseguir. O consumidor já é idempotente, então uma republicação duplicada é inócua.

## Alternativas consideradas

- **Segunda função disparada por Eventarc a cada escrita no Firestore** — mais componentes, mais
  custo por invocação e nenhuma vantagem para um volume pequeno. Descartada.
- **Retentativa síncrona longa na própria requisição** — aumentaria a latência percebida pelo
  operador justamente quando o Pub/Sub está instável. Descartada; mantém-se uma retentativa curta
  antes de marcar a pendência.

## Consequências

### Positivas

- Um recurso a mais (o job) e nenhum serviço novo; tudo em Terraform.
- Comportamento verificável na Sprint 9: derrubar a publicação, observar a pendência, ver a
  reconciliação acontecer.

### Negativas e riscos

- Janela de até 5 minutos entre a falha e a republicação; aceitável, pois o sistema central é
  assíncrono por natureza.

## Referências

- `docs/arquitetura.md`, seção 6; `docs/modelo-dados.md`, seções 2 e 3.
