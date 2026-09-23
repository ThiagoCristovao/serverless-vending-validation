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

## Recursos irreversíveis ficam no bootstrap

Habilitar o Firebase e o Identity Platform não pode ser desfeito pela API. Esses dois recursos são
geridos em [`infra/bootstrap/firebase.tf`](../../bootstrap/firebase.tf), aplicado uma vez por projeto
e nunca destruído. Assim, tudo neste ambiente (banco, tópicos, contas, app Android, alertas, painel)
é destruído e recriado de fato por `terraform destroy` + `terraform apply` (critério da Sprint 3).

## Recriação do Firestore após `destroy`

Ao excluir o banco `(default)`, o Firestore reserva esse ID por cerca de **5 minutos**. Um `apply`
imediato falha em `google_firestore_database` com `Database ID '(default)' is not available ...
retry in N seconds`; os demais recursos são criados normalmente. Basta repetir o `apply` após a
espera. Os três índices compostos levam cerca de 7 minutos para ficar prontos.
