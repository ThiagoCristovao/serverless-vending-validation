# ferramentas — scripts de apoio

Utilitários que não fazem parte do produto, mas sustentam o desenvolvimento e a avaliação.

| Ferramenta | Sprint | Propósito |
|---|---|---|
| `gerar-qr/` | 4–5 | Gera o par de chaves Ed25519 de exemplo, assina payloads conforme `docs/payload-qr.md` e emite o QR em PNG para colar em uma máquina de testes |
| `semear-firestore/` | 5 | Carrega máquinas, operadores e chaves públicas de exemplo no Firestore (ou no emulador) |
| `injecao-falhas/` | 9 | Scripts que desligam o consumidor, simulam rede intermitente e indisponibilidade do provedor de autenticação, coletando evidências |
| `carga/` | 10 | Scripts k6 para os testes de carga e coleta de p50/p95/p99, cold start e regime contínuo |

Ferramentas em Go ficam em subdiretórios com `go.mod` próprio ou compartilham o módulo do serviço,
a decidir na Sprint 4.
