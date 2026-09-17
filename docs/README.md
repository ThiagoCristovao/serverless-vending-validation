# Documentação

| Documento | Conteúdo | Sprint |
|---|---|---|
| [planejamento-sprints.md](planejamento-sprints.md) | Sprints 0–10: objetivo, backlog, entregável e critério de conclusão | — |
| [requisitos.md](requisitos.md) | Atores, fluxo, requisitos funcionais e não funcionais, regras de negócio | 1 |
| [payload-qr.md](payload-qr.md) | Esquema do QR code fixado na máquina, assinatura e validação | 1 |
| [modelo-dados.md](modelo-dados.md) | Coleções do Firestore, índices, regras e esquema dos eventos Pub/Sub | 1 |
| [arquitetura.md](arquitetura.md) | Componentes, fluxos, mensageria, segurança, cenários de exceção, telas | 2 |
| [diagramas/](diagramas/) | Fontes Mermaid dos diagramas usados na arquitetura e na monografia | 2 |
| [adr/](adr/) | Registros de decisão de arquitetura (ADR) | todas |
| [operacao/configuracao-ambiente.md](operacao/configuracao-ambiente.md) | Toolchain, projeto GCP, bootstrap, ambiente dev | 0 |
| [operacao/convencoes.md](operacao/convencoes.md) | Idioma, branches, commits, ADRs, nomes de recursos | 0 |
| [referencias/TCC1-proposta.pdf](referencias/TCC1-proposta.pdf) | Proposta aprovada no TCC 1 | — |

O contrato da API fica em [`../api/openapi.yaml`](../api/openapi.yaml), com exemplos em
[`../api/exemplos`](../api/exemplos).

## Redação da monografia

A fonte LaTeX da monografia fica fora deste repositório (ADR-0001). Ao encerrar cada sprint, o
capítulo correspondente deve ser atualizado com as decisões e os resultados registrados aqui, como
prevê o planejamento. Os ADRs são a fonte primária para a seção de decisões arquiteturais.
