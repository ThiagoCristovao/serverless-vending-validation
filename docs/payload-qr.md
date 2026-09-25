# Payload do QR code

**Sprint 1.** Esquema definitivo do QR fixado em cada máquina, fechando a proposta preliminar do
TCC 1 (Seção 3.2). Decisão de desenho em [ADR-0002](adr/0002-contrassenha-codigo-supervisor-e-qr-assinado.md).

## 1. Princípios

- **Estático.** É um adesivo impresso; a máquina é offline e não tem tela. Nada no payload muda
  com o tempo.
- **Autossuficiente para a validação local.** O aplicativo consegue rejeitar payloads malformados,
  vencidos ou (desejável) forjados sem rede.
- **Assinado, não secreto.** Os campos são legíveis; a integridade vem da assinatura Ed25519. Não
  há segredo no adesivo nem no aplicativo.
- **Pequeno.** Cabe em um QR de versão baixa, legível a 20–40 cm com câmera de celular comum.

## 2. Esquema

Codificação: JSON compacto (sem espaços), UTF-8, chaves na ordem abaixo.

| Campo | Tipo | Obrig. | Origem | Restrição | Descrição |
|---|---|---|---|---|---|
| `v` | inteiro | sim | fixo | `= 1` | Versão do esquema do payload |
| `maq` | string | sim | cadastro da máquina | `^[A-Z0-9-]{3,32}$` | Identificador único da máquina na rede (`idMaquina`) |
| `mod` | string | sim | cadastro da máquina | enum `CN168` | Modelo do equipamento |
| `loc` | string | sim | cadastro da localização | `^[A-Z0-9-]{1,32}$` | Identificador da localização física |
| `exp` | string | sim | emissão do adesivo | data ISO 8601 `AAAA-MM-DD` | Último dia de validade do adesivo |
| `kid` | string | sim | chave de assinatura | `^[a-z0-9]{1,8}$` | Identificador da chave pública que valida `sig` |
| `sig` | string | sim | emissão do adesivo | `^[A-Za-z0-9_-]{86}$` | Assinatura Ed25519 (64 bytes) em base64url sem preenchimento |

### Exemplo

```json
{"v":1,"maq":"VM-2047","mod":"CN168","loc":"BLA-T","exp":"2027-12-31","kid":"k1","sig":"01234567890123456789012345678901234567890123456789012345678901234567890123456789abcdef"}
```

A assinatura acima é ilustrativa (formato correto, valor fictício). Exemplos versionados em
[`api/exemplos`](../api/exemplos).

## 3. Assinatura

**Mensagem canônica:** valores de `v`, `maq`, `mod`, `loc`, `exp` e `kid`, nesta ordem, unidos por
`|`, em UTF-8. Para o exemplo: `1|VM-2047|CN168|BLA-T|2027-12-31|k1`.

**Algoritmo:** Ed25519 (RFC 8032). `sig = base64url_sem_padding(Ed25519.assinar(chave_privada[kid], canonica))`.

**Chaves.**

- A chave privada pertence ao administrador da rede e fica **fora da nuvem** (gerada e usada pela
  ferramenta `ferramentas/gerar-qr`). Para o trabalho, um par de exemplo é gerado e sua parte
  pública é versionada; a privada de exemplo pode ser versionada com aviso explícito de que não
  serve para produção.
- A chave pública de cada `kid` fica na coleção `chavesQr` do Firestore (usada pelo serviço) e
  embutida no aplicativo (usada na verificação local, RF-05).
- Rotação: novo `kid`, novos adesivos; o `kid` antigo fica ativo até o último adesivo vencer.

## 4. Validação

| Verificação | Aplicativo (local) | Serviço | Erro do serviço |
|---|---|---|---|
| JSON válido e apenas os campos do esquema | sim | sim | 400 `payload_invalido` |
| Tipos, padrões e `v = 1` | sim | sim | 400 `payload_invalido` |
| `mod` suportado | sim | sim | 422 `modelo_nao_suportado` |
| `exp` maior ou igual à data atual | sim (relógio do dispositivo) | sim (relógio do servidor, autoritativo) | 410 `qr_expirado` |
| `kid` conhecido e ativo | desejável | sim | 422 `chave_qr_desconhecida` |
| `sig` válida para a mensagem canônica | desejável (RF-05) | sim | 422 `assinatura_invalida` |
| Máquina cadastrada e ativa | não | sim | 404 `maquina_desconhecida` / 409 `maquina_inativa` |
| `mod` e `loc` coincidem com o cadastro | não | sim (divergência indica adesivo trocado de máquina) | 422 `assinatura_invalida` (`detail` explica) |

## 5. Tamanho e legibilidade

Payload do exemplo: cerca de 150 caracteres antes de `sig` e 240 no total. Em modo byte com
correção de erro **M**, cabe em um QR versão 11 (61×61 módulos). Com correção **Q** (recomendada para
adesivos sujeitos a desgaste), versão 13. Impresso a 4 cm de lado, é lido confortavelmente a
20–40 cm.

Verificação prevista na Sprint 5, com a ferramenta `gerar-qr`: medir a leitura em dispositivo físico
com correção M e Q e registrar o resultado nesta seção.

## 6. Evolução

- `v = 2` poderá adotar codificação binária compacta (CBOR + base45) se o tamanho virar problema.
- Novos modelos entram na enumeração de `mod` por ADR, junto com as regras de contrassenha do
  modelo.
