# infra/bootstrap — preparação do projeto GCP

Provisiona o mínimo que precisa existir antes de qualquer outro módulo Terraform:

1. habilitação das APIs usadas ao longo de todas as sprints;
2. bucket do Cloud Storage para o estado remoto dos ambientes (versionado, acesso uniforme,
   prevenção de acesso público);
3. orçamento mensal com alertas em 50 %, 90 % e 100 % (opcional);
4. as habilitações **irreversíveis** do projeto: adoção do projeto Firebase (criado pelo console,
   ADR-0010, adotado por bloco `import`) e Identity Platform com o provedor e-mail e senha.

## Quando executar

Uma vez por projeto GCP, logo após criar o projeto **pelo console do Firebase** e vinculá-lo ao
faturamento
(passo a passo em [docs/operacao/configuracao-ambiente.md](../../docs/operacao/configuracao-ambiente.md)).

## Estado

Este diretório usa **estado local** porque o bucket de estado ainda não existe quando ele roda.
O arquivo `terraform.tfstate` gerado fica fora do versionamento (`.gitignore`). Após cada `apply`,
copie-o para o bucket recém-criado com `make infra-bootstrap-salvar-estado`; o bucket é versionado,
então cada cópia fica preservada. Sem o estado, alterações futuras no bootstrap exigem
`terraform import`.

## Como executar

```bash
cp infra/bootstrap/terraform.tfvars.example infra/bootstrap/terraform.tfvars
# editar terraform.tfvars
make infra-bootstrap-init
make infra-bootstrap-plan
make infra-bootstrap-apply
make infra-bootstrap-salvar-estado
```

Pré-requisito: `gcloud auth application-default login` já executado e a API
`serviceusage.googleapis.com` habilitada manualmente (o Terraform precisa dela para habilitar as
demais).

## Saídas

- `bucket_estado_nome` e `backend_hcl_sugerido`: copie para `infra/ambientes/dev/backend.hcl`.
- `projeto_numero`: usado em políticas de IAM nas sprints seguintes.

## Nunca destruir

Este diretório não deve passar por `terraform destroy`: as APIs ficam habilitadas
(`disable_on_destroy = false`), o bucket guarda o estado dos ambientes e Firebase/Identity Platform
não podem ser removidos pela API. O ciclo `destroy` + `apply` da Sprint 3 aplica-se aos ambientes
(`infra/ambientes/*`), não ao bootstrap.
