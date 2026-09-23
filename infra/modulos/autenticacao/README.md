# Módulo `autenticacao` — implementado na Sprint 3

Habilita o Firebase no projeto, ativa o Identity Platform com o provedor **e-mail e senha**
(sem cadastro anônimo) e registra o aplicativo Android, expondo o conteúdo do
`google-services.json` como saída sensível.

| Entrada | Descrição | Padrão |
|---|---|---|
| `projeto_id` | Projeto GCP | — |
| `pacote_android` | Pacote do app (`applicationId`) | — |
| `nome_app_android` | Nome de exibição no Firebase | `svv-aplicativo` |
| `dominios_autorizados` | Domínios extras para redirecionamento | `[]` |

Saídas: `projeto_firebase`, `app_android_id`, `google_services_nome_arquivo`,
`google_services_json` (sensível).

Operadores são criados por administração (script `ferramentas/semear-firestore` ou console), não
por autocadastro no aplicativo.

**Irreversibilidade.** Habilitar o Firebase e o Identity Platform não pode ser desfeito pela API.
No `terraform destroy` esses dois recursos apenas saem do estado; o `apply` seguinte os encontra e
volta a gerenciá-los. Isso não afeta o critério da Sprint 3, pois o estado resultante é idêntico.
