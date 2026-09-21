# servico — serviço de validação (Go)

Implementado a partir da **Sprint 4** (domínio) e **Sprint 5** (adaptadores e implantação).
Executa como Cloud Run function (2ª geração), runtime `go127`, invocada exclusivamente pelo
API Gateway.

## Estrutura prevista

```
servico/
├── go.mod                  módulo github.com/ThiagoCristovao/serverless-vending-validation/servico
├── funcao.go               único contato com o Functions Framework: init() registra
│                           functions.HTTP("ValidarMaquina", ...) e monta as dependências
├── cmd/local/main.go       executa a função localmente com funcframework (porta 8080)
├── internal/
│   ├── dominio/            entidades e regras: PayloadQr, Maquina, Validacao, Contrassenha (HMAC),
│   │                       erros de domínio; sem dependência de nuvem
│   ├── aplicacao/          casos de uso: IniciarValidacao, ConcluirValidacao, ConsultarValidacao
│   ├── portas/             interfaces: RepositorioMaquinas, RepositorioValidacoes, RepositorioOperadores,
│   │                       PublicadorEventos, ProvedorChavesQr, Relogio
│   └── adaptadores/
│       ├── http/           handlers conforme ../api/openapi.yaml, erros em Problem Details
│       ├── firestore/      implementação dos repositórios (transações)
│       ├── pubsub/         publicador de eventos
│       └── memoria/        implementações em memória para testes de unidade
└── test/integracao/        testes com os emuladores do Firestore e do Pub/Sub
```

## Princípios

- Arquitetura limpa: `dominio` não importa nada de `adaptadores`; `aplicacao` depende só de
  `portas`.
- Critério da Sprint 4: `go test ./...` roda sem rede e sem credenciais.
- Contrato primeiro: os handlers seguem `api/openapi.yaml`; geração de tipos com `oapi-codegen` é
  a alternativa preferida para evitar divergência manual.
- Logs estruturados em JSON com `idValidacao`, `idMaquina` e `uid` em todas as linhas de uma
  requisição.
