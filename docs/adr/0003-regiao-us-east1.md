# ADR-0003: Provisionar todos os recursos regionais em `us-east1`

- **Status:** aceita (informar ao orientador)
- **Data:** 2026-09-16
- **Decisores:** Thiago Cristovão de Souza
- **Sprint:** 0

## Contexto

A escolha natural para um trabalho brasileiro seria `southamerica-east1` (São Paulo). Em
2026-09-16 a documentação do API Gateway listava como regiões suportadas apenas `us-central1`,
`us-east1`, `us-east4`, `us-west2`, `us-west3`, `us-west4`, `europe-west1`, `europe-west2`,
`asia-northeast1` e `australia-southeast1`. Não há API Gateway em São Paulo.

## Decisão

Todos os recursos regionais (API Gateway, Cloud Run function, Firestore, bucket de estado) em
**`us-east1`** (Carolina do Sul), a região suportada pelo gateway mais próxima do Brasil. A região
é uma única variável (`regiao`) nos ambientes Terraform.

## Alternativas consideradas

- **Gateway em `us-east1` e função/Firestore em `southamerica-east1`** — cada requisição faria um
  salto entre regiões (gateway → função) e outro de volta, somando latência variável que
  contaminaria as medições de p50/p95/p99 da Sprint 10. Descartada.
- **`us-central1`** — igualmente suportada, porém mais distante do Brasil. Descartada.
- **Substituir o API Gateway por outro mecanismo (ex.: Cloud Run com validação de JWT na própria
  função)** — mudaria a arquitetura aprovada no TCC 1 e removeria a separação de responsabilidades
  defendida na Seção 2.7. Descartada.

## Consequências

### Positivas

- Todos os saltos internos ficam na mesma região; latência mais estável e medições limpas.
- Arquitetura do TCC 1 preservada.

### Negativas e riscos

- Latência adicional dispositivo → nuvem para operadores no Brasil (ordem de 100–150 ms de ida e
  volta). A avaliação deve medir e discutir esse componente separadamente da latência do serviço.
- Dados em jurisdição estrangeira; irrelevante para o escopo acadêmico com dados fictícios, mas
  deve ser mencionado como consideração de uma implantação real.

## Referências

- Google Cloud. *API Gateway deployment model* (lista de regiões).
  <https://docs.cloud.google.com/api-gateway/docs/deployment-model>. Acesso em 16 set. 2026.
