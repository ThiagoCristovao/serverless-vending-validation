# servico — serviço de validação (Go)

Cloud Run function (2ª geração), runtime `go127`, invocada exclusivamente pelo API Gateway.
Arquitetura limpa: o domínio não conhece nuvem, transporte nem persistência.

| Sprint | Estado | Conteúdo |
|---|---|---|
| 4 | **implementada** | `internal/dominio`, `internal/portas`, `internal/aplicacao`, `internal/adaptadores/memoria` e testes de unidade; só biblioteca padrão |
| 5 | pendente | `funcao.go` (registro no Functions Framework), `cmd/local`, adaptadores `http`, `firestore` e `pubsub`, testes de integração com emuladores |

## Estrutura

```
servico/
├── go.mod                          github.com/ThiagoCristovao/serverless-vending-validation/servico
├── internal/
│   ├── dominio/                    entidades e regras (ADR-0002, docs/requisitos.md)
│   │   ├── payloadqr.go            decodificação, validação, assinatura Ed25519 e hash do adesivo
│   │   ├── contrassenha.go         HMAC-SHA256 → código supervisor de 4 dígitos
│   │   ├── identificador.go        ULID sem dependências
│   │   ├── maquina.go              Maquina, Operador, efeitos da conclusão (rotação)
│   │   ├── validacao.go            ciclo de vida: aberta → concluida | expirada
│   │   ├── evento.go               validacao.iniciada / validacao.concluida
│   │   └── erros.go                códigos estáveis (espelham api/openapi.yaml)
│   ├── portas/                     interfaces: repositórios, publicador, relógio, identificadores
│   ├── aplicacao/                  casos de uso: IniciarValidacao, ConcluirValidacao, ConsultarValidacao
│   └── adaptadores/
│       └── memoria/                implementação em memória das portas (testes e execução local)
└── (Sprint 5) funcao.go, cmd/local/, internal/adaptadores/{http,firestore,pubsub}/
```

## Regras implementadas

- **Adesivo (QR):** esquema fechado (campos desconhecidos rejeitados), padrões de formato, modelo
  suportado, validade até o fim do dia `exp` (UTC), assinatura Ed25519 sobre
  `v|maq|mod|loc|exp|kid`, chave identificada por `kid` e conferência com o cadastro da máquina.
- **Contrassenha:** `HMAC-SHA256(chave, idMaquina:contador)` → 4 dígitos; a resposta traz o código
  corrente e o próximo; a rotação (`contador + 1`) acontece na conclusão com `codigoRotacionado`.
- **Validação:** uma aberta por máquina (RN-03); expira em 15 min (RN-04), avaliada de forma
  preguiçosa; conclusão idempotente; leituras não negativas; observações até 500 caracteres.
- **Idempotência:** `Idempotency-Key` + hash do conteúdo; mesma chave com conteúdo diferente ou
  outro operador é conflito (ADR-0009).
- **Eventos:** publicados após persistir; falha de publicação não afeta o operador e deixa o
  marcador de pendência para reconciliação (CE-14).
- **Erros:** `dominio.Erro` com `Codigo` estável, mapeado para HTTP e Problem Details pelo adaptador
  HTTP (Sprint 5).

## Como verificar

```bash
make servico-verificar      # go vet + go test ./... (sem rede, sem credenciais)
make servico-cobertura      # cobertura por pacote
```

Critério da Sprint 4: a suíte executa integralmente em ambiente local, sem acesso à rede.
