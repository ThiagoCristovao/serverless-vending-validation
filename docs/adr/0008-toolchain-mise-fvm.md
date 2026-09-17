# ADR-0008: Versões do toolchain fixadas com mise (Go, Terraform) e FVM (Flutter)

- **Status:** aceita
- **Data:** 2026-09-16
- **Decisores:** Thiago Cristovão de Souza
- **Sprint:** 0

## Contexto

A máquina de desenvolvimento não tinha Go, Flutter, Terraform nem gcloud instalados. O critério
da Sprint 0 exige que qualquer pessoa clone o repositório e reproduza o ambiente seguindo o README,
o que pede versões fixadas e versionadas.

## Decisão

- **mise** gerencia Go e Terraform a partir de `.mise.toml` na raiz.
- **FVM** gerencia o Flutter a partir de `aplicativo/.fvmrc`.
- **gcloud CLI** é instalado pelo repositório apt oficial (fora de gerenciadores de versão, como
  recomenda o Google).
- O Makefile usa imagens Docker (`hashicorp/terraform`, `redocly/cli`) como alternativa **apenas
  para verificação** quando as ferramentas locais não existem.

Versões fixadas em 2026-09-16: Go 1.27.1 (runtime `go127` das Cloud Run functions), Terraform
1.16.3, Flutter 3.47.4 (Dart 3.13.3), provider `hashicorp/google` `~> 8.3`.

## Alternativas consideradas

- **Devcontainer (Docker)** — ambiente totalmente reproduzível, mas a imagem com Flutter e Android
  SDK é pesada e o acesso ao dispositivo Android físico via USB dentro do container exige
  configuração extra. Descartada.
- **Instalação manual documentada** — simples, porém sem garantia de versão entre máquinas.
  Descartada.
- **mise também para Flutter** — o plugin comunitário é menos estável que o FVM, que é a
  ferramenta consagrada no ecossistema. Descartada.

## Consequências

### Positivas

- Versões visíveis no repositório e trocadas por PR.
- Instalação nativa: depuração USB e `flutter doctor` funcionam sem atrito.

### Negativas e riscos

- Três mecanismos de instalação (mise, FVM, apt) em vez de um; documentado em
  `docs/operacao/configuracao-ambiente.md`.
- Atualizar o Go exige conferir o runtime GA correspondente nas Cloud Run functions.

## Referências

- <https://mise.jdx.dev>; <https://fvm.app>; Google Cloud, *Cloud Run functions runtimes*
  (<https://docs.cloud.google.com/run/docs/runtimes/function-runtimes>). Acesso em 16 set. 2026.
