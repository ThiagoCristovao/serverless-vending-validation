# infra/ambientes/dev — ambiente de desenvolvimento

Raiz Terraform do ambiente `dev`. Compõe os módulos de [`infra/modulos`](../../modulos) e usa o
estado remoto no bucket criado pelo [bootstrap](../../bootstrap) (prefixo `ambientes/dev`).

Sprint 3 (implementada): `iam`, `autenticacao`, `firestore`, `pubsub`, `observabilidade`.
Sprint 5 (comentada em `main.tf`): `funcao`, `gateway`.

Outros ambientes (`homolog`, `prod`) seguem o mesmo padrão em diretórios irmãos, com `prefix`
próprio no backend e `protecao_exclusao = true` no Firestore.

## Como executar

```bash
cp infra/ambientes/dev/backend.hcl.example infra/ambientes/dev/backend.hcl
cp infra/ambientes/dev/terraform.tfvars.example infra/ambientes/dev/terraform.tfvars
# editar ambos
make infra-dev-init
make infra-dev-plan
make infra-dev-apply
make segredo-hmac-gerar        # uma vez: cria a primeira versão da chave HMAC
```

Na CI e no `make verificar`, o diretório é validado com `terraform init -backend=false`, sem
credenciais.

## Recursos que não podem ser desfeitos

Habilitar o Firebase e o Identity Platform no projeto são ações sem "desfazer" na API. No
`terraform destroy` esses dois recursos apenas saem do estado; o `apply` seguinte volta a
gerenciá-los. O restante (banco, tópicos, contas, alertas, painel) é destruído e recriado de fato.
