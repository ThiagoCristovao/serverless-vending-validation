# ADR-0001: Organizar serviço, aplicativo, infraestrutura e documentação em um único repositório

- **Status:** aceita
- **Data:** 2026-09-16
- **Decisores:** Thiago Cristovão de Souza
- **Sprint:** 0

## Contexto

O TCC 1 (Seção 3.4.3) menciona que a infraestrutura seria "versionada em repositório dedicado",
enquanto a Sprint 0 do planejamento descreve um único repositório com diretórios para o serviço em
Go, o aplicativo Flutter, os módulos Terraform e a documentação. Era preciso escolher.

## Decisão

Monorepo: `servico/`, `aplicativo/`, `sistema-central-simulado/`, `infra/`, `api/`, `docs/` e
`ferramentas/` no mesmo repositório. A fonte LaTeX da monografia fica **fora** (Overleaf ou
repositório próprio) e referencia este repositório.

## Alternativas consideradas

- **Multi-repo (um por stack)** — exigiria sincronizar o contrato OpenAPI entre três repositórios
  e fragmentaria o histórico apresentado à banca. Descartada.
- **Monorepo incluindo a monografia** — misturaria artefatos de compilação LaTeX com código e
  aumentaria o repositório; o autor prefere editar a monografia em ferramenta própria. Descartada.

## Consequências

### Positivas

- O contrato `api/openapi.yaml` é versionado junto de cliente e servidor; uma mudança de contrato
  e suas implementações cabem em um único PR.
- A linha do tempo (tags `sprint-N`) mostra o projeto inteiro evoluindo.
- Um só `make verificar` e uma só CI.

### Negativas e riscos

- A CI precisa distinguir o que testar por diretório para não ficar lenta quando as três stacks
  tiverem código; mitigado com jobs separados por caminho.
- O texto do TCC 1 diverge; a monografia deve explicar a mudança em uma frase.

## Referências

- TCC 1, Seção 3.4.3; `docs/planejamento-sprints.md`, Sprint 0.
