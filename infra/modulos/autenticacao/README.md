# Módulo `autenticacao` — implementado na Sprint 3

Registra o aplicativo Android no Firebase e expõe o conteúdo do `google-services.json` como saída
sensível (gravado com `make app-google-services`, nunca versionado).

As duas habilitações **irreversíveis** do projeto, Firebase e Identity Platform (provedor e-mail e
senha), não ficam aqui: estão em [`infra/bootstrap/firebase.tf`](../../bootstrap/firebase.tf),
porque a API não permite desfazê-las e o ambiente precisa passar por `destroy` + `apply` sem
intervenção manual (ADR-0010).

| Entrada | Descrição | Padrão |
|---|---|---|
| `projeto_id` | Projeto GCP já habilitado no Firebase | — |
| `pacote_android` | Pacote do app (`applicationId`) | — |
| `nome_app_android` | Nome de exibição no Firebase | `svv-aplicativo` |

Saídas: `app_android_id`, `google_services_nome_arquivo`, `google_services_json` (sensível).

Operadores são criados por administração (script `ferramentas/semear-firestore` ou console), não
por autocadastro no aplicativo.
