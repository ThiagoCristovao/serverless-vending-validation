# Módulo `gateway` — Sprint 5

**Responsabilidade.** Exposição do serviço via API Gateway com validação do JWT do Firebase.

**Recursos previstos** (provider `google-beta`)

- `google_api_gateway_api` — `svv-<amb>-api`.
- `google_api_gateway_api_config` — gerado a partir de [`api/openapi.yaml`](../../../api/openapi.yaml)
  com `templatefile()` substituindo `PROJETO_ID` e a URL da função; conta de serviço
  `svv-<amb>-gateway` em `gateway_config.backend_config.google_service_account`.
- `google_api_gateway_gateway` — `svv-<amb>-gateway`, região `us-east1`.

**Verificação do JWT** (esquema de segurança `firebaseJwt` no contrato):
`x-google-issuer = https://securetoken.google.com/<PROJETO_ID>`,
`x-google-jwks_uri = https://www.googleapis.com/service_accounts/v1/metadata/x509/securetoken@system.gserviceaccount.com`,
`x-google-audiences = <PROJETO_ID>`. As claims chegam à função no cabeçalho `X-Apigw-Api-Userinfo`.

**Entradas previstas.** `projeto_id`, `regiao`, `prefixo`, URL da função, conta de serviço.

**Saídas previstas.** `hostname` do gateway (base do `servers[0].url` do contrato).

O API Gateway aceita OpenAPI 2.0, 3.0.x e 3.1.x (verificado em 2026-09-16), por isso o contrato
canônico em 3.1 é reaproveitado sem conversão (ADR-0006). Confirmar a sintaxe exata das extensões
de JWT em 3.x na documentação "Autenticar usuários com Firebase" ao implementar.
