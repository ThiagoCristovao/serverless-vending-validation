# Convenções

## Idioma

Português em identificadores, nomes de recursos, mensagens de commit e documentação (ADR-0007).

- Identificadores, nomes de arquivo e de recursos **sem acentos ou cedilha**: `validacao`,
  `maquina`, `contrassenha`. URLs, Terraform e ferramentas Go não lidam bem com eles.
- Palavras impostas por ferramenta ficam como são: `main`, `init`, `internal/`, `go.mod`,
  `pubspec.yaml`, `README.md`, cabeçalho `Idempotency-Key`.
- Campos de JSON em `camelCase`: `idValidacao`, `proximaContrassenha`. Campos do QR em chaves curtas
  (`maq`, `mod`, `loc`) por limitação de tamanho.
- Nomes de recursos GCP em `kebab-case` com prefixo `svv-<ambiente>-`: `svv-dev-validacoes`.
- Pacotes Go em minúsculas, uma palavra: `dominio`, `aplicacao`, `portas`, `adaptadores`.

## Commits

[Conventional Commits](https://www.conventionalcommits.org/pt-br/) com os tipos da especificação e
descrição em português, no imperativo, sem ponto final.

```
<tipo>(<escopo>): <descrição>

[corpo opcional]
```

- **Tipos:** `feat`, `fix`, `docs`, `infra`, `test`, `refactor`, `chore`, `ci`.
- **Escopos:** `servico`, `aplicativo`, `infra`, `api`, `docs`, `central`, `ferramentas`.
- Exemplos: `docs(api): definir contrato inicial de validacoes`,
  `infra(pubsub): adicionar fila de mensagens mortas`, `feat(servico): derivar contrassenha por HMAC`.

## Branches e PRs

- `main` protegida; nada é commitado diretamente nela.
- Branches `sprint-N/descricao-curta` (ex.: `sprint-0/fundacao`, `sprint-4/dominio-contrassenha`).
- Um PR por entregável, usando o modelo em `.github/PULL_REQUEST_TEMPLATE.md`; merge por squash.
- Ao encerrar uma sprint, criar a tag `sprint-N` no commit de merge. As tags são a linha do tempo
  apresentada à banca.

## Registros de decisão (ADR)

- Um arquivo por decisão em `docs/adr/NNNN-titulo-curto.md`, a partir de `0000-modelo.md`.
- Status: `proposta` (aguarda validação do orientador), `aceita`, `substituída por ADR-XXXX`,
  `rejeitada`. Uma decisão não é editada depois de aceita; cria-se outra que a substitui.
- Toda escolha que a banca possa questionar merece um ADR: tecnologia, região, algoritmo,
  formato de dados, escopo cortado.

## Terraform

- Um módulo por serviço gerenciado em `infra/modulos/<nome>`; ambientes compõem módulos em
  `infra/ambientes/<ambiente>`.
- Variáveis e saídas em português; `description` obrigatória.
- `terraform fmt` antes de commitar (`make infra-formatar`). O `.terraform.lock.hcl` é versionado.
- Nenhum valor sensível em `.tf`; `tfvars` e `backend.hcl` ficam fora do git.

## Documentação

- Markdown com linhas de até 100 caracteres.
- Lacunas explícitas marcadas com `> **TODO:**` e, quando dependem do orientador, `(validar com
  orientador)`.
- Diagramas em Mermaid, com a fonte em `docs/diagramas/` e exportação para a monografia feita a
  partir dela.
