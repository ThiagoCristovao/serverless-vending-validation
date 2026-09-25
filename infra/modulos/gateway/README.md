# Módulo `gateway` — implementado na Sprint 5

**Responsabilidade.** Exposição do serviço via API Gateway com validação do JWT do Firebase.

**Recursos** (provider `google-beta`)

- `google_api_gateway_api` — `svv-<amb>-api`.
- `google_api_gateway_api_config` — gerado a partir de [`api/openapi.yaml`](../../../api/openapi.yaml)
  com `templatefile()` substituindo `PROJETO_ID` e a URL da função; conta de serviço
  `svv-<amb>-gateway` em `gateway_config.backend_config.google_service_account`.
- `google_api_gateway_gateway` — `svv-<amb>-gateway`, região `us-east1`.

**Verificação do JWT** (esquema de segurança `firebaseJwt` no contrato, forma OpenAPI 3.x):
`type: oauth2` com fluxo implícito e a extensão `x-google-auth` (`issuer =
https://securetoken.google.com/<PROJETO_ID>`, `jwksUri = .../securetoken@system.gserviceaccount.com`,
`audiences = [<PROJETO_ID>]`). As claims chegam à função no cabeçalho `X-Apigateway-Api-Userinfo`,
com o payload do JWT em base64url.

**Entradas previstas.** `projeto_id`, `regiao`, `prefixo`, URL da função, conta de serviço.

**Saídas previstas.** `hostname` do gateway (base do `servers[0].url` do contrato).

O API Gateway aceita OpenAPI 2.0, 3.0.x e 3.1.x (verificado em 2026-09-16), por isso o contrato
canônico em 3.1 é reaproveitado sem conversão (ADR-0006). A sintaxe das extensões de JWT em 3.x foi
confirmada em 2026-09-25 na documentação "Authenticating users with Firebase".
