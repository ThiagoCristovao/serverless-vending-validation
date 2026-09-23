# ADR-0010: Criar o projeto GCP pelo console do Firebase e adotá-lo no Terraform por `import`

- **Status:** aceita
- **Data:** 2026-09-23
- **Decisores:** Thiago Cristovão de Souza
- **Sprint:** 3

## Contexto

O guia original criava o projeto com `gcloud projects create` e deixava o Terraform habilitar o
Firebase (`google_firebase_project`, que chama `projects.addFirebase`). Na prática, com a conta
`alunos.utfpr.edu.br` (Google Workspace for Education, organização `alunos.utfpr.edu.br`):

- `addFirebase` respondeu `403 PERMISSION_DENIED: The caller does not have permission`, sem
  detalhes, para o projeto `svv-dev-1419`, mesmo com `roles/owner`, `firebase.projects.update`
  confirmado por `testIamPermissions` e o projeto de cota configurado no provider.
- A lista `availableProjects` da API do Firebase (e a lista "Adicionar projeto" do console) não
  incluía o projeto, nem após 80 minutos, nem um segundo projeto de teste criado sem nenhuma API.
  O projeto criado pelo assistente da avaliação gratuita aparecia normalmente.
- As políticas de organização efetivas (`gcp.restrictServiceUsage`, `serviceuser.services`,
  `iam.allowedPolicyMemberDomains`, `gcp.resourceLocations`, criação de contas de serviço) estavam
  todas liberadas.

A causa não é documentada pelo Google. Criar o projeto **pelo console do Firebase** funcionou de
imediato (`svv-dev`).

## Decisão

1. O projeto GCP de cada ambiente é criado pelo console do Firebase (com Google Analytics
   desativado), e o faturamento é vinculado depois pelo `gcloud`.
2. O Terraform **adota** o recurso `google_firebase_project` com um bloco `import` idempotente no
   **bootstrap**, junto com o `google_identity_platform_config`. Ambos são irreversíveis pela API
   (o teste de `destroy` + `apply` do ambiente falhou com "Identity Platform has already been
   enabled"), então vivem no bootstrap, que nunca é destruído, e o ambiente fica inteiramente
   recriável, que é o critério da Sprint 3.
3. Tudo o mais (app Android, Firestore, Pub/Sub, IAM, observabilidade, função, gateway) permanece
   criado exclusivamente por Terraform, no ambiente.

## Alternativas consideradas

- **Esperar a elegibilidade** — 80 minutos sem mudança; prazo do TCC não permite apostar em
  propagação. Descartada.
- **Conta Google pessoal como Owner para habilitar o Firebase** — mistura identidades e transfere
  parte da configuração para fora da conta institucional. Descartada enquanto o console funcionar.
- **Usar o projeto da avaliação gratuita ("My First Project")** — ID sem significado e sem
  controle sobre o que o assistente habilitou. Descartada.

## Consequências

### Positivas

- Desbloqueou a Sprint 3 no mesmo dia; o ID `svv-dev` ficou limpo.
- O restante da infraestrutura segue 100 % declarativo.

### Negativas e riscos

- Um passo manual (criar o projeto) precede o bootstrap; está documentado em
  `docs/operacao/configuracao-ambiente.md` e é feito uma vez por ambiente.
- A monografia deve relatar o desvio como limitação prática da abordagem "tudo como código":
  a criação do próprio projeto ficou fora do Terraform.
- O console do Firebase habilita APIs extras (App Engine, Hosting, FCM, Remote Config) que o
  projeto não usa; são gratuitas e inofensivas, mas não gerenciadas pelo Terraform.

## Referências

- Firebase Management API, `projects.availableProjects` e `projects.addFirebase`.
- Diagnóstico registrado no histórico do PR da Sprint 3.
