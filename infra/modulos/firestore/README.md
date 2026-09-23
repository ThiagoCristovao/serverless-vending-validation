# Módulo `firestore` — implementado na Sprint 3

Banco Firestore em modo nativo, backups diários, regras de segurança que negam todo acesso de
cliente e os índices compostos de [docs/modelo-dados.md](../../../docs/modelo-dados.md).

| Entrada | Descrição | Padrão |
|---|---|---|
| `projeto_id` | Projeto GCP | — |
| `regiao` | Região do banco | — |
| `protecao_exclusao` | Protege contra exclusão (falso em dev para `destroy` + `apply`) | `false` |
| `backup_diario` | Agenda backup diário | `true` |
| `retencao_backup` | Retenção dos backups | `604800s` |

Saídas: `banco_nome`, `banco_id`.

As regras ficam em `firestore.rules` e são publicadas como release `cloud.firestore`; alterar o
arquivo gera um novo ruleset e a release é substituída automaticamente.
