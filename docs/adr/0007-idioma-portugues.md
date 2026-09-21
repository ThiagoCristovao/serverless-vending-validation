# ADR-0007: Português em código, commits e documentação

- **Status:** aceita
- **Data:** 2026-09-16
- **Decisores:** Thiago Cristovão de Souza
- **Sprint:** 0

## Contexto

O trabalho é apresentado em português a uma banca brasileira, e os documentos técnicos do
repositório alimentam diretamente a monografia. Era preciso decidir o idioma de identificadores,
mensagens de commit e documentação, e como conviver com palavras impostas por ferramentas.

## Decisão

Tudo em português, com regras práticas:

- Identificadores, nomes de arquivo e de recursos **sem acentos ou cedilha** (`validacao`,
  `maquina`), porque URLs, Terraform, Go e Dart não lidam bem com eles.
- Palavras impostas por ferramenta permanecem (`main`, `init`, `internal/`, `go.mod`,
  `pubspec.yaml`, cabeçalho `Idempotency-Key`).
- Conventional Commits mantém os **tipos** da especificação (`feat`, `fix`, `docs`, `infra`,
  `test`, `refactor`, `chore`, `ci`) por serem tokens padronizados; a descrição é em português.
- Campos de JSON em `camelCase` português (`idValidacao`); campos do QR abreviados por tamanho.

## Alternativas consideradas

- **Código e commits em inglês, documentação em português** — padrão de mercado; descartada por
  preferência do autor e pela coerência entre código e texto da monografia.
- **Tudo em inglês** — exigiria traduzir a documentação para a monografia. Descartada.

## Consequências

### Positivas

- Nomes do código aparecem tal como na monografia; menos tradução mental para a banca.

### Negativas e riscos

- Mistura inevitável com nomes de bibliotecas em inglês (`functions.HTTP`, `AsyncNotifier`).
- Contribuidores externos futuros podem estranhar; aceitável para o escopo.

## Referências

- `docs/operacao/convencoes.md`.
