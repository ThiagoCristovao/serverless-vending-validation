# Modelo de dados

**Sprint 1.** Coleções e documentos do Firestore (modo nativo, região `us-east1`), índices, regras
de segurança e esquema dos eventos publicados no Pub/Sub. Complementa
[payload-qr.md](payload-qr.md) e [`api/openapi.yaml`](../api/openapi.yaml).

Convenções: nomes de coleção no plural e em português sem acento; campos em `camelCase`;
instantes como `Timestamp` do Firestore (serializados em ISO 8601 UTC na API); identificadores de
validação em **ULID** (26 caracteres, ordenáveis por tempo).

## 1. Coleções

### 1.1 `maquinas/{idMaquina}`

Uma por máquina física. Criada pelo administrador (script `semear-firestore`).

| Campo | Tipo | Obrig. | Origem | Descrição |
|---|---|---|---|---|
| `modelo` | string | sim | cadastro | Enum `CN168` |
| `localizacao` | map | sim | cadastro | `{ id: string, nome: string }`; `id` deve coincidir com `loc` do QR |
| `ativa` | boolean | sim | cadastro | Máquinas inativas não podem ser validadas |
| `contadorCodigo` | integer | sim | serviço | Contador da rotação da contrassenha; começa em 0 |
| `validacaoAbertaId` | string ou null | sim | serviço | ULID da validação `aberta`, se houver (trava de RN-03) |
| `ultimaValidacaoId` | string ou null | sim | serviço | ULID da última validação concluída |
| `ultimaValidacaoEm` | timestamp ou null | sim | serviço | Instante da última conclusão |
| `ultimoMedidor` | integer ou null | sim | serviço | Medidor informado na última conclusão (RN-07) |
| `criadaEm`, `atualizadaEm` | timestamp | sim | serviço/script | Auditoria |

### 1.2 `validacoes/{idValidacao}`

Uma por visita iniciada. Nunca armazena a contrassenha em claro: guarda o `contadorCodigo` da
época, do qual o código é derivável (ADR-0002).

| Campo | Tipo | Obrig. | Origem | Descrição |
|---|---|---|---|---|
| `idMaquina` | string | sim | payload QR | Referência a `maquinas` |
| `idOperador` | string | sim | JWT (`uid`) | Referência a `operadores` |
| `status` | string | sim | serviço | `aberta`, `concluida`, `expirada` |
| `iniciadaEm` | timestamp | sim | serviço | |
| `expiraEm` | timestamp | sim | serviço | `iniciadaEm + 15 min` (RN-04) |
| `concluidaEm` | timestamp ou null | sim | serviço | |
| `contadorCodigo` | integer | sim | serviço | Snapshot de `maquinas.contadorCodigo` na abertura |
| `chaveIdempotencia` | string | sim | cabeçalho `Idempotency-Key` | UUID |
| `hashRequisicao` | string | sim | serviço | SHA-256 do corpo, para detectar mesma chave com corpo diferente (422) |
| `qr` | map | sim | payload QR | `{ kid, exp, loc, mod }` para auditoria |
| `leituras` | map ou null | sim | conclusão | `{ medidor: integer, unidadesVendidas: integer }` |
| `codigoRotacionado` | boolean ou null | sim | conclusão | |
| `observacoes` | string ou null | não | conclusão | Até 500 caracteres |
| `avisos` | array de string | não | serviço | Condições aceitas mas sinalizadas na conclusão, ex.: `medidor_menor_que_anterior` (RN-07, ADR-0011) |
| `eventos` | map | sim | serviço | `{ iniciadaPublicadaEm: timestamp\|null, concluidaPublicadaEm: timestamp\|null }` — marcador de outbox (CE-14) |
| `dispositivo` | map ou null | não | requisição | `{ plataforma, versaoApp }` para diagnóstico |

### 1.3 `operadores/{uid}`

Chave = `uid` do Firebase Authentication. Criada pelo administrador.

| Campo | Tipo | Obrig. | Descrição |
|---|---|---|---|
| `nome` | string | sim | Exibido no rodapé do aplicativo |
| `email` | string | sim | Igual ao do Firebase Auth |
| `rota` | string ou null | sim | Identificador da rota (futuro: autorização por rota) |
| `ativo` | boolean | sim | RN-06 |
| `criadoEm` | timestamp | sim | |

### 1.4 `chavesQr/{kid}`

| Campo | Tipo | Obrig. | Descrição |
|---|---|---|---|
| `algoritmo` | string | sim | `Ed25519` |
| `chavePublica` | string | sim | 32 bytes em base64url |
| `ativa` | boolean | sim | Chaves inativas rejeitam qualquer adesivo |
| `validaDe`, `validaAte` | timestamp | sim | Janela de emissão de adesivos com esta chave |

## 2. Transações e invariantes

- **Abertura** (`POST /v1/validacoes`), em uma transação: ler `maquinas/{id}` e `operadores/{uid}`;
  se `validacaoAbertaId` apontar para validação `aberta` e não expirada → 409; se apontar para uma
  expirada → marcá-la `expirada` e seguir; criar `validacoes/{ulid}`; gravar `validacaoAbertaId`.
- **Idempotência**: antes da transação, consultar `validacoes` por `chaveIdempotencia`; se existir
  com o mesmo `hashRequisicao` → devolver o mesmo resultado; se diferente → 422.
- **Conclusão**, em uma transação: validação deve estar `aberta` e dentro de `expiraEm`; gravar
  leituras e `status = concluida`; se `codigoRotacionado` → `maquinas.contadorCodigo += 1`;
  atualizar `ultimaValidacao*`, `ultimoMedidor`; limpar `validacaoAbertaId`.
- **Expiração** é avaliada de forma preguiçosa (na leitura ou na próxima abertura). Um job
  periódico é opcional.
- **Publicação** ocorre **depois** da transação; falha de publicação deixa
  `eventos.*PublicadaEm = null` para reconciliação (CE-14) e não altera a resposta ao operador.

## 3. Índices

| Coleção | Campos | Uso |
|---|---|---|
| `validacoes` | `chaveIdempotencia` (simples, automático) | Idempotência |
| `validacoes` | `idMaquina` asc, `iniciadaEm` desc | Histórico por máquina |
| `validacoes` | `idOperador` asc, `iniciadaEm` desc | Histórico por operador |
| `validacoes` | `status` asc, `expiraEm` asc | Job opcional de expiração |
| `validacoes` | `eventos.iniciadaPublicadaEm` asc (simples) e `eventos.concluidaPublicadaEm` asc | Reconciliação de publicação |

## 4. Regras de segurança

O aplicativo nunca acessa o Firestore. Regras:

```
rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /{document=**} {
      allow read, write: if false;
    }
  }
}
```

O serviço acessa com sua conta de serviço (`roles/datastore.user`), que ignora as regras.

## 5. Retenção

Durante o trabalho nada é excluído: o volume é pequeno e o histórico completo interessa à avaliação.
Para uma implantação real, recomenda-se a política TTL do Firestore em `validacoes` com um campo
`expurgarEm = concluidaEm + 2 anos`, que o próprio banco aplica sem código adicional. Backups
diários ficam retidos por 7 dias (módulo `firestore`).

## 6. Eventos publicados no Pub/Sub

Tópico `svv-<ambiente>-validacoes`. Estilo *event-carried state transfer*: a mensagem traz o que o
sistema central precisa, sem consulta de volta.

**Atributos da mensagem** (usados para filtro e ordenação sem abrir o corpo):

| Atributo | Valor |
|---|---|
| `tipo` | `validacao.iniciada` ou `validacao.concluida` |
| `idValidacao` | ULID |
| `idMaquina` | id da máquina |
| `versao` | `1` |
| chave de ordenação | `idMaquina` |

**Corpo** (JSON):

```json
{
  "tipo": "validacao.concluida",
  "versao": 1,
  "idValidacao": "01J8ZK3V9Q7XW2N4M6P8R0T2Y4",
  "ocorridoEm": "2026-09-17T13:19:44Z",
  "maquina": { "id": "VM-2047", "modelo": "CN168", "localizacao": { "id": "BLA-T", "nome": "Bloco A - Térreo" } },
  "operador": { "id": "uid-do-firebase", "nome": "Operador Exemplo" },
  "validacao": {
    "iniciadaEm": "2026-09-17T13:05:12Z",
    "concluidaEm": "2026-09-17T13:19:44Z",
    "leituras": { "medidor": 14832, "unidadesVendidas": 137 },
    "codigoRotacionado": true,
    "avisos": []
  }
}
```

Para `validacao.iniciada`, `validacao` traz apenas `iniciadaEm` e `expiraEm`. A contrassenha
**nunca** trafega no evento.

**Consumidor:** idempotência por `(idValidacao, tipo)`; confirmação (`ack`) somente após persistir.
