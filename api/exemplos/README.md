# Exemplos do contrato

Arquivos JSON que ilustram [`../openapi.yaml`](../openapi.yaml). Servem de fixture para os testes
do serviço (Sprint 4) e do aplicativo (Sprint 7). Os mesmos exemplos estão embutidos no contrato
para serem validados pelo lint.

| Arquivo | Descrição |
|---|---|
| `qr-valido.json` | Payload do QR de uma máquina ativa, adesivo dentro da validade |
| `qr-expirado.json` | Payload com `exp` no passado (rejeitado localmente e pelo serviço com 410) |
| `validacao-requisicao.json` | Corpo de `POST /v1/validacoes` |
| `validacao-resposta.json` | Resposta 201: contrassenha corrente, próxima contrassenha e dados da máquina |
| `conclusao-requisicao.json` | Corpo de `POST /v1/validacoes/{id}/conclusao` |
| `conclusao-resposta.json` | Resposta 200 da conclusão |
| `consulta-resposta.json` | Resposta 200 de `GET /v1/validacoes/{id}` para uma validação concluída |
| `erro-409.json` | Erro em Problem Details: validação em andamento para a máquina |

O campo `sig` dos payloads de QR é **ilustrativo** até que a ferramenta `ferramentas/gerar-qr`
produza assinaturas reais com a chave de exemplo; ele respeita apenas o formato (86 caracteres
base64url).
