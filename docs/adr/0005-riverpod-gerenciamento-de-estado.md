# ADR-0005: Riverpod para gerenciamento de estado e injeção de dependências no Flutter

- **Status:** aceita
- **Data:** 2026-09-16
- **Decisores:** Thiago Cristovão de Souza
- **Sprint:** 0 (aplicada na Sprint 6)

## Contexto

A Sprint 6 pede que a abordagem de gerenciamento de estado seja definida antes de estruturar o
projeto Flutter. O aplicativo tem três telas e um fluxo linear (login → leitura → resultado), com
estados assíncronos (autenticação, requisição HTTP, conectividade) e necessidade de testes de
widget.

## Decisão

**Riverpod** (`flutter_riverpod`, com `riverpod_annotation` para geração de código), usando
`AsyncNotifier` para os fluxos assíncronos e provedores como mecanismo de injeção de dependências
(cliente HTTP, repositórios, relógio), o que permite substituí-los em testes.

## Alternativas consideradas

- **Bloc** — separação explícita entre eventos e estados, mais cerimônia; vantajoso em fluxos
  complexos, desnecessário para três telas. Descartada.
- **Provider** — precursor do Riverpod, sem segurança em tempo de compilação e com dependência da
  árvore de widgets. Descartada.
- **setState puro** — não resolve injeção de dependências para testes. Descartada.

## Consequências

### Positivas

- Pouco código para muito comportamento; testes de unidade e de widget com `ProviderScope` e
  `overrides`.
- Estados assíncronos (`AsyncValue`) mapeiam diretamente para os estados de tela definidos na
  arquitetura (carregando, erro, sucesso).

### Negativas e riscos

- Geração de código (`build_runner`) adiciona um passo ao fluxo de desenvolvimento.

## Referências

- <https://riverpod.dev>; `docs/planejamento-sprints.md`, Sprint 6.
