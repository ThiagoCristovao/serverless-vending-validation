# Módulo `firestore` — Sprint 3

**Responsabilidade.** Banco Firestore (modo nativo) e regras de segurança.

**Recursos previstos**

- `google_firestore_database` — banco `(default)`, `location_id = us-east1`, `type = FIRESTORE_NATIVE`,
  proteção contra exclusão habilitada, backups diários (`google_firestore_backup_schedule`).
- `google_firebaserules_ruleset` + `google_firebaserules_release` — regras que **negam todo acesso de
  cliente**: o aplicativo nunca fala com o Firestore; só o serviço, via conta de serviço.
- `google_firestore_index` — índices compostos de [docs/modelo-dados.md](../../../docs/modelo-dados.md).

**Entradas previstas.** `projeto_id`, `regiao`.

**Saídas previstas.** `banco_nome`.
