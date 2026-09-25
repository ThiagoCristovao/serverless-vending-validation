# ADR-0006: Um único contrato OpenAPI 3.0 para documentação, geração de código e API Gateway

- **Status:** aceita (revisada em 2026-09-25: versão 3.0.3 em vez de 3.1)
- **Data:** 2026-09-16
- **Decisores:** Thiago Cristovão de Souza
- **Sprint:** 1

## Contexto

A Sprint 1 exige o contrato da API em OpenAPI. Historicamente o API Gateway aceitava apenas
OpenAPI 2.0, o que forçaria manter dois arquivos (um 3.x para documentação e geração de código, um
2.0 para o gateway). Em 2026-09-16 a documentação do API Gateway passou a listar suporte a
OpenAPI 2.0, 3.0.x e 3.1.x.

## Decisão

`api/openapi.yaml` em **OpenAPI 3.0.3** é o contrato canônico e o único. Ele carrega a declaração
de backend `x-google-api-management` (referenciada por `x-google-backend`) e o esquema de segurança
`firebaseJwt` com `x-google-auth`, com marcadores (`PROJETO_ID`, `URL_DA_FUNCAO`) substituídos por
`replace()` no módulo Terraform `gateway`.

**Revisão de 2026-09-25.** O contrato nasceu em 3.1, mas o conversor do API Gateway rejeitou duas
construções: tipos múltiplos (`type: [string, 'null']`, próprios do 3.1) e o `x-google-backend`
em forma de objeto (em 3.x o backend é declarado em `x-google-api-management.backends` e referenciado
por nome). Rebaixar para 3.0.3 (`nullable: true`, `enum` no lugar de `const`, `example` no lugar de
`examples`) resolveu sem perder expressividade relevante e mantém um único arquivo.
Erros seguem RFC 9457 (`application/problem+json`). O lint (`redocly lint`) roda na CI.

## Alternativas consideradas

- **Dois arquivos (3.1 canônico + 2.0 para o gateway)** — necessário apenas se o suporte a 3.x não
  existisse; custo de sincronização. Descartada.
- **Autorar em 2.0** — perderia recursos de esquema (`oneOf`, `const`, `examples`) e ferramentas
  modernas de geração para Go e Dart. Descartada.

## Consequências

### Positivas

- Uma fonte de verdade para servidor (`oapi-codegen`), cliente (geração Dart) e gateway.
- O critério da Sprint 1 ("contrato suficiente para implementar cliente e servidor de forma
  independente") fica verificável pelo lint.

### Negativas e riscos

- O conversor do gateway é mais restritivo que o lint: toda mudança de contrato deve ser aplicada
  no ambiente dev antes do merge, porque só a criação do `api_config` valida as extensões.

## Referências

- Google Cloud. *OpenAPI overview — API Gateway*.
  <https://docs.cloud.google.com/api-gateway/docs/openapi-overview>. Acesso em 16 set. 2026.
- RFC 9457, *Problem Details for HTTP APIs*.
