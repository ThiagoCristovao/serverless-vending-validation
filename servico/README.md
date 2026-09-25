# servico — serviço de validação (Go)

Cloud Run function (2ª geração), runtime `go127`, invocada exclusivamente pelo API Gateway (contrato
público) e pelo Cloud Scheduler (endpoint interno de reconciliação). Arquitetura limpa: o domínio
não conhece nuvem, transporte nem persistência.

| Sprint | Estado | Conteúdo |
|---|---|---|
| 4 | implementada | `internal/dominio`, `internal/portas`, `internal/aplicacao`, `internal/adaptadores/memoria` e testes de unidade; só biblioteca padrão |
| 5 | **implementada** | `funcao.go`, adaptadores `apihttp`, `armazenamento` (Firestore), `mensageria` (Pub/Sub), `sistema`, `registro`, `config`, reconciliação de publicações, ferramentas em `cmd/`, testes de integração com emulador |

## Estrutura

```
servico/
├── go.mod                       github.com/ThiagoCristovao/serverless-vending-validation/servico
├── funcao.go                    entry point ValidarMaquina (Functions Framework); monta as dependências na 1ª requisição
├── cmd/
│   ├── local/                   executa o serviço com dados em memória, sem credenciais (porta 8080)
│   ├── gerar-qr/                gera o par de chaves Ed25519 e assina adesivos (JSON e PNG)
│   ├── semear-firestore/        cadastra máquina, operador e chave no Firestore (ou emulador)
│   └── token-operador/          obtém um ID token do Firebase Auth para chamar o gateway com curl
├── internal/
│   ├── dominio/                 entidades e regras (ADR-0002, docs/requisitos.md)
│   ├── portas/                  interfaces: repositórios, publicador, relógio, identificadores
│   ├── aplicacao/               casos de uso: iniciar, concluir, consultar, reconciliar publicações
│   ├── config/                  variáveis de ambiente (SVV_*)
│   ├── registro/                slog em JSON no formato do Cloud Logging
│   └── adaptadores/
│       ├── apihttp/             rotas do contrato, Problem Details, identidade do gateway, log por requisição
│       ├── armazenamento/       repositórios no Firestore com transações; testes com emulador (tag integracao)
│       ├── mensageria/          publicador Pub/Sub v2 com chave de ordenação e tempo limite curto
│       ├── sistema/             relógio real e ULID com entropia criptográfica
│       └── memoria/             portas em memória para testes e execução local
```

## Rotas

| Método e caminho | Uso |
|---|---|
| `POST /v1/validacoes` | inicia a validação (contrato) |
| `GET /v1/validacoes/{id}` | consulta (contrato) |
| `POST /v1/validacoes/{id}/conclusao` | conclui (contrato) |
| `POST /interno/reconciliar-publicacoes` | Cloud Scheduler, fora do gateway (ADR-0012) |
| `GET /saude` | verificação local |

A identidade do operador vem do cabeçalho `X-Apigateway-Api-Userinfo` (claims do JWT já validado
pelo gateway). Em execução local, `X-Operador-Teste: <uid>` substitui o cabeçalho.

## Configuração (variáveis de ambiente, definidas pelo Terraform)

`SVV_PROJETO_ID`, `SVV_TOPICO_VALIDACOES`, `SVV_CHAVE_HMAC` (hex, do Secret Manager),
`SVV_FIRESTORE_BANCO`, `SVV_PRAZO_VALIDACAO`, `SVV_ATRASO_RECONCILIACAO`, `SVV_LOTE_RECONCILIACAO`,
`SVV_NIVEL_LOG`, `SVV_TEMPO_LIMITE_PUBLICACAO`. Detalhes em `internal/config`.

## Como verificar

```bash
make servico-verificar      # go vet + testes de unidade (sem rede)
make servico-cobertura
make emuladores-subir && make servico-integracao && make emuladores-parar   # Firestore emulado
make servico-local          # serviço em memória; imprime um QR assinado e um curl de exemplo
```

Critério da Sprint 5 (verificado em `svv-dev`): uma requisição autenticada com payload válido
devolve a contrassenha, persiste o registro no Firestore e publica a mensagem no tópico.
