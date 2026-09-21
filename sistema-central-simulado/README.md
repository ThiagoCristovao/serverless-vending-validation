# sistema-central-simulado — consumidor do sistema central (Go)

Implementado na **Sprint 8**. Simula o sistema central da rede de vending como assinante do
tópico `svv-<ambiente>-validacoes`, para viabilizar os testes end-to-end e os cenários de
resiliência das Sprints 9 e 10.

## Comportamento previsto

- Consome a assinatura `svv-<ambiente>-sistema-central` (pull, com ordenação por `idMaquina`).
- Aplica **idempotência** por (`idValidacao`, `tipo`): mensagens duplicadas são reconhecidas e
  ignoradas.
- Persiste o que recebe em um armazenamento local simples (SQLite ou arquivo JSON) e expõe um
  endpoint de consulta para as asserções dos testes.
- Possui um **interruptor de indisponibilidade** (variável de ambiente ou endpoint administrativo)
  que faz o consumidor parar de confirmar mensagens ou falhar deliberadamente, para exercitar
  retenção, retentativa e fila de mensagens mortas.
- Roda localmente (Sprint 8) ou como serviço no Cloud Run (se a avaliação exigir).

## Estrutura prevista

```
sistema-central-simulado/
├── go.mod
├── cmd/central/main.go
└── internal/{consumidor,armazenamento,admin}/
```
