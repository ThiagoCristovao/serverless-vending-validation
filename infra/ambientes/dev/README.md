# infra/ambientes/dev — ambiente de desenvolvimento

Raiz Terraform do ambiente `dev`. Compõe os módulos de [`infra/modulos`](../../modulos) e usa o
estado remoto no bucket criado pelo [bootstrap](../../bootstrap).

Na Sprint 0 este diretório só define convenções (prefixo `svv-dev`, rótulos) e o backend; os
módulos entram nas Sprints 3 e 5. Outros ambientes (`homolog`, `prod`) seguem o mesmo padrão em
diretórios irmãos, com `prefix` próprio no backend.

## Como executar

```bash
cp infra/ambientes/dev/backend.hcl.example infra/ambientes/dev/backend.hcl
cp infra/ambientes/dev/terraform.tfvars.example infra/ambientes/dev/terraform.tfvars
# editar ambos
make infra-dev-init
make infra-dev-plan
```

Na CI e no `make verificar`, o diretório é validado com `terraform init -backend=false`, sem
credenciais.
