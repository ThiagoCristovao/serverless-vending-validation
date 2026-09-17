# ADR-0006: Um único contrato OpenAPI 3.1 para documentação, geração de código e API Gateway

- **Status:** aceita
- **Data:** 2026-09-16
- **Decisores:** Thiago Cristovão de Souza
- **Sprint:** 1

## Contexto

A Sprint 1 exige o contrato da API em OpenAPI. Historicamente o API Gateway aceitava apenas
OpenAPI 2.0, o que forçaria manter dois arquivos (um 3.x para documentação e geração de código, um
2.0 para o gateway). Em 2026-09-16 a documentação do API Gateway passou a listar suporte a
OpenAPI 2.0, 3.0.x e 3.1.x.

## Decisão

`api/openapi.yaml` em **OpenAPI 3.1** é o contrato canônico e o único. Ele carrega as extensões
`x-google-backend` e as extensões de JWT do esquema de segurança `firebaseJwt`, com marcadores
(`PROJETO_ID`, URL da função) substituídos por `templatefile()` no módulo Terraform `gateway`.
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

- A sintaxe exata das extensões de JWT em documentos 3.x deve ser confirmada na documentação
  "Autenticar usuários com Firebase" do API Gateway ao implantar (Sprint 5). Se algum recurso 3.1
  não for aceito pelo gateway, a saída é rebaixar o arquivo para 3.0.x, não voltar a 2.0.

## Referências

- Google Cloud. *OpenAPI overview — API Gateway*.
  <https://docs.cloud.google.com/api-gateway/docs/openapi-overview>. Acesso em 16 set. 2026.
- RFC 9457, *Problem Details for HTTP APIs*.
